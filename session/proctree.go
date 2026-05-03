package session

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ProcessSnapshot is a single-shot view of the system process tree, cheap to
// query repeatedly. Built once per metadata tick and shared across instances
// so we run `ps` exactly once instead of N times.
type ProcessSnapshot struct {
	children map[int][]int
	parents  map[int]int
	pcpu     map[int]float64
	cmdline  map[int]string
}

// SnapshotProcesses returns a fresh snapshot of the current process tree
// with each process's current CPU usage percentage and full argv string.
// Returns nil on failure — callers must treat nil as "no signal".
//
// The pcpu value comes from `ps -o pcpu`, which on macOS is the average CPU
// usage over the process's lifetime as the kernel sees it, decayed toward
// recent activity. Idle daemons (vite waiting for requests, MCP servers
// waiting for stdin) report 0.0; processes doing real work report >0.
//
// cmdline is the full command line, used by HasFreshWatchedFile to detect
// activity through log files referenced in poll/monitor scripts (e.g.
// `until grep -q ... /tmp/smoke.log; do sleep 5; done`).
func SnapshotProcesses() *ProcessSnapshot {
	cmd := exec.Command("ps", "-axo", "pid=,ppid=,pcpu=,command=")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	snap := &ProcessSnapshot{
		children: make(map[int][]int, 512),
		parents:  make(map[int]int, 512),
		pcpu:     make(map[int]float64, 512),
		cmdline:  make(map[int]string, 512),
	}
	for _, line := range strings.Split(string(out), "\n") {
		// Three numeric fields then everything else is the command.
		var pid, ppid int
		var pcpu float64
		var cmdStart int
		// Parse pid
		s := strings.TrimLeft(line, " ")
		idx := strings.IndexByte(s, ' ')
		if idx <= 0 {
			continue
		}
		v, err := strconv.Atoi(s[:idx])
		if err != nil {
			continue
		}
		pid = v
		s = strings.TrimLeft(s[idx:], " ")
		// Parse ppid
		idx = strings.IndexByte(s, ' ')
		if idx <= 0 {
			continue
		}
		v, err = strconv.Atoi(s[:idx])
		if err != nil {
			continue
		}
		ppid = v
		s = strings.TrimLeft(s[idx:], " ")
		// Parse pcpu
		idx = strings.IndexByte(s, ' ')
		if idx <= 0 {
			continue
		}
		f, err := strconv.ParseFloat(s[:idx], 64)
		if err != nil {
			continue
		}
		pcpu = f
		s = strings.TrimLeft(s[idx:], " ")
		_ = cmdStart
		snap.children[ppid] = append(snap.children[ppid], pid)
		snap.parents[pid] = ppid
		snap.pcpu[pid] = pcpu
		snap.cmdline[pid] = s
	}
	return snap
}

// pathRe matches path-shaped tokens ending in a log/output extension.
// Allows both absolute (`/tmp/foo.log`) and relative (`.claude/x/foo.log`)
// paths — the caller resolves relative ones against a known cwd. Requires
// at least one path separator anywhere in the match so bare `v8.log` style
// version strings don't trigger.
var pathRe = regexp.MustCompile(`[A-Za-z0-9._+-]*/[A-Za-z0-9._/+-]+\.(log|output|out|txt|jsonl)`)

// WorktreeRoots returns PIDs to count as "associated with this worktree"
// for activity tracking. Always includes panePID. Also includes any
// process whose argv references the worktreePath as a path component
// (delimited by /, whitespace, or quote) — this catches detached test
// runners launched with nohup/setsid whose PPID=1 and whose work happens
// entirely outside the panePID descendant tree.
//
// Excludes:
//   - Ancestors of panePID (tmux session/server invoked with `-c <worktree>`,
//     login shells, etc.). These wrap panePID, not its work, and tmux's own
//     pcpu drifts above the threshold during normal pane rendering — adding
//     it as a root would falsely trip the activity signal even when the
//     pane's contents are static.
//   - Processes whose argv continues into a nested worktree
//     (`<worktreePath>/.claude/worktrees/<other>/...`) so a parent worktree
//     doesn't claim a child worktree's processes.
//
// Note: cross-contamination via shared node_modules paths (e.g. a child's
// playwright child process loading playwright/lib from the parent
// worktree's hoisted node_modules) is handled at the activity-signal
// level, not here.
func (s *ProcessSnapshot) WorktreeRoots(worktreePath string, panePID int) []int {
	roots := []int{panePID}
	if s == nil || worktreePath == "" {
		return roots
	}
	ancestors := map[int]bool{}
	for p := s.parents[panePID]; p > 1; p = s.parents[p] {
		if ancestors[p] {
			break
		}
		ancestors[p] = true
	}
	seen := map[int]bool{panePID: true}
	for pid, cmd := range s.cmdline {
		if seen[pid] || ancestors[pid] {
			continue
		}
		idx := strings.Index(cmd, worktreePath)
		if idx < 0 {
			continue
		}
		end := idx + len(worktreePath)
		if end < len(cmd) {
			next := cmd[end]
			if next != '/' && next != ' ' && next != '\t' && next != '"' && next != '\'' {
				continue
			}
			if strings.HasPrefix(cmd[end:], "/.claude/worktrees/") {
				continue
			}
		}
		roots = append(roots, pid)
		seen[pid] = true
	}
	return roots
}

// HasFreshWatchedFile walks the descendants of every root and returns true
// if any of their argvs reference a log/output file that was modified
// within the threshold window. cwdHint is used as the base for relative
// paths (typically the worktree path) — pass "" to skip relative resolution.
//
// This is the escape hatch for the case where Claude runs a polling shell
// (`until grep -q ... .claude/test-run-output/foo.log; do sleep N; done`)
// while the actual workload (e.g. `playwright test`) is detached with
// PPID=1 — so CPU-on-descendants alone reads zero, but the log file is
// still growing. Many such shells reference the log via a relative path
// because Claude's bash tool runs with cwd=worktree.
func (s *ProcessSnapshot) HasFreshWatchedFile(roots []int, cwdHint string, threshold time.Duration) bool {
	if s == nil || len(roots) == 0 {
		return false
	}
	cutoff := time.Now().Add(-threshold)
	visited := map[int]bool{}
	checked := map[string]bool{}
	check := func(cmd string) bool {
		for _, p := range pathRe.FindAllString(cmd, -1) {
			resolved := p
			if !strings.HasPrefix(p, "/") {
				if cwdHint == "" {
					continue
				}
				resolved = filepath.Join(cwdHint, p)
			}
			if checked[resolved] {
				continue
			}
			checked[resolved] = true
			info, err := os.Stat(resolved)
			if err != nil {
				continue
			}
			if info.ModTime().After(cutoff) {
				return true
			}
		}
		return false
	}
	for _, root := range roots {
		if visited[root] {
			continue
		}
		visited[root] = true
		// The root's own argv counts (it might be the polling shell itself,
		// matched via WorktreeRoots).
		if check(s.cmdline[root]) {
			return true
		}
		queue := []int{root}
		for len(queue) > 0 {
			pid := queue[0]
			queue = queue[1:]
			for _, child := range s.children[pid] {
				if visited[child] {
					continue
				}
				visited[child] = true
				queue = append(queue, child)
				if check(s.cmdline[child]) {
					return true
				}
			}
		}
	}
	return false
}

// DescendantCPU returns the sum of pcpu across every root (panePID
// included) and every descendant. Including panePID itself is load-bearing:
// during extended thinking and subagent dispatch, Claude's binary spends
// minutes parsing API responses and routing MCP calls while the JSONL
// transcript stays silent and descendants (MCP servers waiting on stdin)
// read 0%. Without panePID in the sum, those phases look idle.
//
// Empirical baseline: idle Claude with all MCP servers loaded reads 0.0%
// even after 8h+ of uptime (lifetime-averaged pcpu decays to zero when the
// process truly idles); active Claude in extended thinking reads 2–8%;
// running `playwright test` between page loads dips to ~0.9%; full-throttle
// test execution crosses 5%. Threshold lives in the caller
// (descendantCPUThreshold).
func (s *ProcessSnapshot) DescendantCPU(roots []int) float64 {
	if s == nil || len(roots) == 0 {
		return 0
	}
	visited := map[int]bool{}
	var total float64
	for _, root := range roots {
		if visited[root] {
			continue
		}
		visited[root] = true
		total += s.pcpu[root]
		queue := []int{root}
		for len(queue) > 0 {
			pid := queue[0]
			queue = queue[1:]
			for _, child := range s.children[pid] {
				if visited[child] {
					continue
				}
				visited[child] = true
				total += s.pcpu[child]
				queue = append(queue, child)
			}
		}
	}
	return total
}
