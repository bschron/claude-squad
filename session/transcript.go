package session

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// transcriptActiveThreshold is the window during which the most recent JSONL
// transcript modification counts as "Claude is doing something". Sized to
// cover the gap between bursts when subagents (Stage code review,
// security review) are in extended-thinking phases — empirically those
// subagent transcripts can go 20–30s between writes while still actively
// reasoning. Combined with the 6s idle debounce, true idle reaches Ready in
// ~30s + 6s after activity stops.
const transcriptActiveThreshold = 30 * time.Second

// backgroundTaskActiveThreshold is the window for considering Claude's
// /tmp/claude-<uid>/.../tasks/*.output files as "actively being written to".
// Long-running test runners and build tools batch output every 20–30s.
const backgroundTaskActiveThreshold = 30 * time.Second

// projectPathEncodeRe matches characters Claude Code normalizes to '-' when
// turning a working-directory path into a folder name under ~/.claude/projects.
// Empirically the rule replaces /, ., _ (and other non-[A-Za-z0-9-] chars).
var projectPathEncodeRe = regexp.MustCompile(`[^A-Za-z0-9-]`)

// encodeProjectPath mirrors Claude Code's path-to-folder-name encoding.
// Example: "/Users/x/CodeApps/foo/.claude/worktrees/bar"
//       → "-Users-x-CodeApps-foo--claude-worktrees-bar"
func encodeProjectPath(p string) string {
	return projectPathEncodeRe.ReplaceAllString(p, "-")
}

// claudeProjectsDir returns ~/.claude/projects, or "" if the home dir lookup
// fails (callers treat "" as "no transcript info available").
func claudeProjectsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", "projects")
}

// transcriptRecentlyModified reports whether any .jsonl transcript for this
// worktree was modified within the threshold window. Two locations are
// checked:
//
//   - Top-level: ~/.claude/projects/<encoded>/*.jsonl — main session
//     transcripts (one per `claude --continue` invocation).
//   - Subagents: ~/.claude/projects/<encoded>/<session>/subagents/agent-*.jsonl —
//     subagent transcripts (Stage code review, security review, etc.) which
//     are appended in real time while the subagent is doing work, even when
//     the main transcript and pane content stay static.
//
// Returns false if the folder is missing or unreadable.
func transcriptRecentlyModified(worktreePath string, threshold time.Duration) bool {
	if worktreePath == "" {
		return false
	}
	base := claudeProjectsDir()
	if base == "" {
		return false
	}
	dir := filepath.Join(base, encodeProjectPath(worktreePath))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	cutoff := time.Now().Add(-threshold)
	for _, e := range entries {
		if !e.IsDir() {
			if filepath.Ext(e.Name()) != ".jsonl" {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			if info.ModTime().After(cutoff) {
				return true
			}
			continue
		}
		// Per-session directory — look inside subagents/ for agent-*.jsonl.
		subDir := filepath.Join(dir, e.Name(), "subagents")
		subEntries, err := os.ReadDir(subDir)
		if err != nil {
			continue
		}
		for _, se := range subEntries {
			if se.IsDir() || filepath.Ext(se.Name()) != ".jsonl" {
				continue
			}
			info, err := se.Info()
			if err != nil {
				continue
			}
			if info.ModTime().After(cutoff) {
				return true
			}
		}
	}
	return false
}

// claudeTasksRoot returns /tmp/claude-<uid>, where Claude Code stores live
// background-task output files. Each project gets a subdir named with the
// same path encoding as ~/.claude/projects.
func claudeTasksRoot() string {
	return fmt.Sprintf("/tmp/claude-%d", os.Getuid())
}

// backgroundTaskActive reports whether any of Claude's background-task .output
// files for this worktree was written to within the threshold window. The
// layout is /tmp/claude-<uid>/<encoded>/<session-id>/tasks/<id>.output and
// these files are appended in real time while shells launched via Claude's
// "run in background" feature are still producing output. This catches the
// long-running E2E / build / monitor case where the visible pane is stable
// and the JSONL transcript is silent (tool result hasn't returned yet).
func backgroundTaskActive(worktreePath string, threshold time.Duration) bool {
	if worktreePath == "" {
		return false
	}
	encoded := encodeProjectPath(worktreePath)
	root := filepath.Join(claudeTasksRoot(), encoded)
	sessions, err := os.ReadDir(root)
	if err != nil {
		return false
	}
	cutoff := time.Now().Add(-threshold)
	for _, sess := range sessions {
		if !sess.IsDir() {
			continue
		}
		tasksDir := filepath.Join(root, sess.Name(), "tasks")
		entries, err := os.ReadDir(tasksDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || filepath.Ext(e.Name()) != ".output" {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			if info.ModTime().After(cutoff) {
				return true
			}
		}
	}
	return false
}
