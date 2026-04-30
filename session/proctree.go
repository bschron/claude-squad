package session

import (
	"os"
	"os/exec"
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
		snap.pcpu[pid] = pcpu
		snap.cmdline[pid] = s
	}
	return snap
}

// absPathRe matches Unix-style absolute paths plausibly pointing at log /
// output files. Anchored to start with `/`; rejects whitespace and shell
// metacharacters. Captures things like `/tmp/smoke-timeline-8.log` and
// `/private/tmp/foo.output` from `ps` argv dumps.
var absPathRe = regexp.MustCompile(`/[A-Za-z0-9._/+-]+\.(log|output|out|txt|jsonl)`)

// HasFreshWatchedFile walks the descendants of root and returns true if any
// of their argvs reference an absolute file path that was modified within
// the threshold window. This is the escape hatch for the case where Claude
// runs a polling shell (`until grep ... /tmp/X.log; do sleep N; done`) while
// the actual workload (e.g. `playwright test`) is detached with PPID=1 — so
// CPU-on-descendants alone reads zero, but the log file is still growing.
func (s *ProcessSnapshot) HasFreshWatchedFile(root int, threshold time.Duration) bool {
	if s == nil || root == 0 {
		return false
	}
	cutoff := time.Now().Add(-threshold)
	queue := []int{root}
	visited := map[int]bool{root: true}
	checked := map[string]bool{}
	for len(queue) > 0 {
		pid := queue[0]
		queue = queue[1:]
		for _, child := range s.children[pid] {
			if visited[child] {
				continue
			}
			visited[child] = true
			queue = append(queue, child)
			for _, p := range absPathRe.FindAllString(s.cmdline[child], -1) {
				if checked[p] {
					continue
				}
				checked[p] = true
				info, err := os.Stat(p)
				if err != nil {
					continue
				}
				if info.ModTime().After(cutoff) {
					return true
				}
			}
		}
	}
	return false
}

// DescendantCPU returns the sum of pcpu across every descendant of root.
// Excludes root itself — we only care about whether children are doing work.
//
// Empirical baseline on this codebase: idle Claude (with all MCP servers
// loaded) sums to 0.0%; vite dev server idle sums to 0.0%; running
// `playwright test` or `npm install` or any compute-heavy descendant easily
// crosses 1% within the first sample. Threshold of 1.0 cleanly separates
// "doing work" from "alive but idle".
func (s *ProcessSnapshot) DescendantCPU(root int) float64 {
	if s == nil || root == 0 {
		return 0
	}
	queue := []int{root}
	visited := map[int]bool{root: true}
	var total float64
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
	return total
}
