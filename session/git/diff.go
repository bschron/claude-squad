package git

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// quickStatsTimeout bounds a single QuickStats run. The diff itself is
// typically <2s; 30s catches truly hung git/FS operations without breaking
// legitimate large-repo runs.
const quickStatsTimeout = 30 * time.Second

// fullDiffTimeout is more generous because Diff() streams the full content.
const fullDiffTimeout = 2 * time.Minute

// DiffStats holds statistics about the changes in a diff
type DiffStats struct {
	// Content is the full diff content
	Content string
	// Added is the number of added lines
	Added int
	// Removed is the number of removed lines
	Removed int
	// Error holds any error that occurred during diff computation
	// This allows propagating setup errors (like missing base commit) without breaking the flow
	Error error
}

func (d *DiffStats) IsEmpty() bool {
	return d.Added == 0 && d.Removed == 0 && d.Content == ""
}

// runGitCommandCtx is like runGitCommand but uses a context-bound exec so the
// process is killed if the timeout fires. Only used for diff/quick-stats —
// long-running ops (push, fetch, large worktree adds) keep the unbounded path.
func (g *GitWorktree) runGitCommandCtx(ctx context.Context, path string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", path}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git command failed: %s (%w)", output, err)
	}
	return string(output), nil
}

// QuickStats returns only the added/removed line counts using `git diff --shortstat`,
// which produces a single short line regardless of diff size. Use this on the
// frequent metadata tick to avoid streaming multi-MB diffs for every instance.
//
// Concurrent callers for the same worktree are deduped: a second caller that
// races a still-running QuickStats receives the last successful result instead
// of spawning another git process. This prevents tick-stacking under load.
func (g *GitWorktree) QuickStats() *DiffStats {
	if !g.quickStatsRunning.CompareAndSwap(false, true) {
		if cached := g.lastQuickStats.Load(); cached != nil {
			cachedCopy := *cached
			return &cachedCopy
		}
		return &DiffStats{}
	}
	defer g.quickStatsRunning.Store(false)

	ctx, cancel := context.WithTimeout(context.Background(), quickStatsTimeout)
	defer cancel()

	stats := &DiffStats{}

	if _, err := g.runGitCommandCtx(ctx, g.worktreePath, "add", "-N", "."); err != nil {
		stats.Error = err
		return stats
	}

	out, err := g.runGitCommandCtx(ctx, g.worktreePath, "--no-pager", "diff", "--shortstat", g.GetBaseCommitSHA())
	if err != nil {
		stats.Error = err
		return stats
	}

	// Output is either empty (no changes) or one line like:
	//   " 2 files changed, 15 insertions(+), 3 deletions(-)"
	// Any of the insertion/deletion clauses may be missing when the value is zero.
	for _, part := range strings.Split(strings.TrimSpace(out), ",") {
		part = strings.TrimSpace(part)
		switch {
		case strings.Contains(part, "insertion"):
			fmt.Sscanf(part, "%d", &stats.Added)
		case strings.Contains(part, "deletion"):
			fmt.Sscanf(part, "%d", &stats.Removed)
		}
	}

	cached := *stats
	g.lastQuickStats.Store(&cached)
	return stats
}

// Diff returns the git diff between the worktree and the base branch along with statistics,
// including the full diff content. This is O(diff size) and should be invoked on demand
// (e.g. when the Diff tab is visible), not on the metadata tick.
func (g *GitWorktree) Diff() *DiffStats {
	if !g.diffRunning.CompareAndSwap(false, true) {
		// A full diff is already in flight; return an empty placeholder.
		// The caller will see the existing content via ApplyDiffStats which
		// preserves the previous Content.
		return &DiffStats{}
	}
	defer g.diffRunning.Store(false)

	ctx, cancel := context.WithTimeout(context.Background(), fullDiffTimeout)
	defer cancel()

	stats := &DiffStats{}

	// -N stages untracked files (intent to add), including them in the diff
	if _, err := g.runGitCommandCtx(ctx, g.worktreePath, "add", "-N", "."); err != nil {
		stats.Error = err
		return stats
	}

	content, err := g.runGitCommandCtx(ctx, g.worktreePath, "--no-pager", "diff", g.GetBaseCommitSHA())
	if err != nil {
		stats.Error = err
		return stats
	}
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			stats.Added++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			stats.Removed++
		}
	}
	stats.Content = content

	return stats
}
