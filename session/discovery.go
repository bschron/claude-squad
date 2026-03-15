package session

import (
	"claude-squad/log"
	"claude-squad/session/git"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DiscoveredSession represents a Claude Code worktree session found on disk.
type DiscoveredSession struct {
	WorktreeName    string // e.g. "ClaudeWorktrees"
	WorktreePath    string // e.g. "/Users/.../project/.claude/worktrees/ClaudeWorktrees"
	TmuxSessionName string // e.g. "claude-squad_worktree-ClaudeWorktrees"
	BranchName      string // e.g. "worktree-ClaudeWorktrees"
	RepoPath        string
}

// DiscoverClaudeWorktrees scans the .claude/worktrees/ directory under the project's
// git root and returns sessions that have an active tmux session.
func DiscoverClaudeWorktrees(projectDir string) ([]DiscoveredSession, error) {
	repoRoot, err := git.FindGitRepoRoot(projectDir)
	if err != nil {
		return nil, err
	}

	worktreesDir := filepath.Join(repoRoot, ".claude", "worktrees")
	entries, err := os.ReadDir(worktreesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	projectDirname := filepath.Base(repoRoot)
	// tmux replaces dots with underscores
	sanitizedProject := strings.ReplaceAll(projectDirname, ".", "_")

	var results []DiscoveredSession
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		expectedTmux := sanitizedProject + "_worktree-" + name
		// Also sanitize any dots in the worktree name
		expectedTmux = strings.ReplaceAll(expectedTmux, ".", "_")

		// Check if tmux session exists
		checkCmd := exec.Command("tmux", "has-session", "-t="+expectedTmux)
		if checkCmd.Run() != nil {
			continue
		}

		branchName := "worktree-" + name

		// Try to get the actual branch from the worktree
		wtPath := filepath.Join(worktreesDir, name)
		branchCmd := exec.Command("git", "-C", wtPath, "branch", "--show-current")
		if out, err := branchCmd.Output(); err == nil {
			if b := strings.TrimSpace(string(out)); b != "" {
				branchName = b
			}
		}

		results = append(results, DiscoveredSession{
			WorktreeName:    name,
			WorktreePath:    wtPath,
			TmuxSessionName: expectedTmux,
			BranchName:      branchName,
			RepoPath:        repoRoot,
		})

		log.InfoLog.Printf("Discovered Claude Code worktree: %s (tmux: %s)", name, expectedTmux)
	}

	return results, nil
}
