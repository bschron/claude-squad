package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHasClaudeConversationHistory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	worktree := "/Users/x/CodeApps/foo/.claude/worktrees/bar"
	projectDir := filepath.Join(home, ".claude", "projects", encodeProjectPath(worktree))

	mkProjectDir := func(t *testing.T) {
		t.Helper()
		if err := os.MkdirAll(projectDir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	writeFile := func(t *testing.T, name string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(projectDir, name), []byte("{}"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	t.Run("empty worktree path is conservative", func(t *testing.T) {
		if !hasClaudeConversationHistory("") {
			t.Fatal("expected true for empty path (conservative)")
		}
	})

	t.Run("missing project folder means no history", func(t *testing.T) {
		// projectDir does not exist yet under this fresh temp HOME.
		if hasClaudeConversationHistory(worktree) {
			t.Fatal("expected false when project folder is absent")
		}
	})

	t.Run("empty project folder means no history", func(t *testing.T) {
		mkProjectDir(t)
		if hasClaudeConversationHistory(worktree) {
			t.Fatal("expected false when project folder has no .jsonl")
		}
	})

	t.Run("subdir only (no top-level jsonl) means no history", func(t *testing.T) {
		mkProjectDir(t)
		if err := os.MkdirAll(filepath.Join(projectDir, "some-session", "subagents"), 0o755); err != nil {
			t.Fatalf("mkdir subdir: %v", err)
		}
		if hasClaudeConversationHistory(worktree) {
			t.Fatal("expected false when only subdirectories exist")
		}
	})

	t.Run("top-level jsonl means history exists", func(t *testing.T) {
		mkProjectDir(t)
		writeFile(t, "abc123.jsonl")
		if !hasClaudeConversationHistory(worktree) {
			t.Fatal("expected true when a top-level .jsonl transcript exists")
		}
	})
}
