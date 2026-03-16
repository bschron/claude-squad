package session

import (
	"claude-squad/config"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
)

// notesDir returns the path to the notes directory, creating it if needed.
func notesDir() (string, error) {
	configDir, err := config.GetConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(configDir, "notes")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create notes directory: %w", err)
	}
	return dir, nil
}

// noteKey returns a deterministic filename for a note scoped to repo + session title.
func noteKey(repoPath, sessionTitle string) string {
	h := sha256.Sum256([]byte(repoPath + "/" + sessionTitle))
	return fmt.Sprintf("%x.md", h[:8])
}

// LoadNote reads the note for the given repo and session. Returns "" if no note exists.
func LoadNote(repoPath, sessionTitle string) (string, error) {
	dir, err := notesDir()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(dir, noteKey(repoPath, sessionTitle)))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// SaveNote writes a note for the given repo and session. If content is empty, the note file is deleted.
func SaveNote(repoPath, sessionTitle string, content string) error {
	dir, err := notesDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, noteKey(repoPath, sessionTitle))
	if content == "" {
		return deleteFile(path)
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// DeleteNote removes the note file for the given repo and session.
func DeleteNote(repoPath, sessionTitle string) error {
	dir, err := notesDir()
	if err != nil {
		return err
	}
	return deleteFile(filepath.Join(dir, noteKey(repoPath, sessionTitle)))
}

func deleteFile(path string) error {
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
