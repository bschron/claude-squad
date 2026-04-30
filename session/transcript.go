package session

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// transcriptActiveThreshold is the window during which the most recent JSONL
// transcript modification counts as "Claude is doing something". Picked to
// cover normal streaming/tool-call cadence without flapping when the model
// briefly pauses between turns.
const transcriptActiveThreshold = 3 * time.Second

// backgroundTaskActiveThreshold is the window for considering Claude's
// /tmp/claude-<uid>/.../tasks/*.output files as "actively being written to".
// Larger than the transcript window because long-running shells can buffer
// output for several seconds between flushes.
const backgroundTaskActiveThreshold = 8 * time.Second

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

// transcriptRecentlyModified reports whether the most recently modified .jsonl
// transcript inside the project's Claude folder was touched within the
// threshold window. Returns false if the folder is missing or unreadable —
// callers treat that as "no signal", not "definitely idle".
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
		if e.IsDir() || filepath.Ext(e.Name()) != ".jsonl" {
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
