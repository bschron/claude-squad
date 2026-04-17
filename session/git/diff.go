package git

import (
	"fmt"
	"strings"
)

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

// QuickStats returns only the added/removed line counts using `git diff --shortstat`,
// which produces a single short line regardless of diff size. Use this on the
// frequent metadata tick to avoid streaming multi-MB diffs for every instance.
func (g *GitWorktree) QuickStats() *DiffStats {
	stats := &DiffStats{}

	if _, err := g.runGitCommand(g.worktreePath, "add", "-N", "."); err != nil {
		stats.Error = err
		return stats
	}

	out, err := g.runGitCommand(g.worktreePath, "--no-pager", "diff", "--shortstat", g.GetBaseCommitSHA())
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
	return stats
}

// Diff returns the git diff between the worktree and the base branch along with statistics,
// including the full diff content. This is O(diff size) and should be invoked on demand
// (e.g. when the Diff tab is visible), not on the metadata tick.
func (g *GitWorktree) Diff() *DiffStats {
	stats := &DiffStats{}

	// -N stages untracked files (intent to add), including them in the diff
	_, err := g.runGitCommand(g.worktreePath, "add", "-N", ".")
	if err != nil {
		stats.Error = err
		return stats
	}

	content, err := g.runGitCommand(g.worktreePath, "--no-pager", "diff", g.GetBaseCommitSHA())
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
