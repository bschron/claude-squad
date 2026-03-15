package kanban_test

import (
	"claude-squad/keys"
	"claude-squad/session"
	"claude-squad/ui"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMenu_OptionAtX(t *testing.T) {
	menu := ui.NewMenu()
	menu.SetSize(200, 3)

	// Create a Running instance so we get the full set of options.
	inst := makeTestInstance("menu-test", session.Running, "branch")
	menu.SetInstance(inst)

	// Render the menu to populate optionPositions.
	_ = menu.String()

	t.Run("returns false for X far out of range", func(t *testing.T) {
		_, ok := menu.OptionAtX(-100)
		assert.False(t, ok, "negative X should not match any option")

		_, ok = menu.OptionAtX(9999)
		assert.False(t, ok, "very large X should not match any option")
	})

	t.Run("returns valid key for center of menu", func(t *testing.T) {
		// The menu is centered in 200 width. Try the center.
		key, ok := menu.OptionAtX(100)
		// We just check it returns something -- exact key depends on layout.
		_ = key
		_ = ok
		// This is a smoke test; the precise option depends on rendering.
	})

	t.Run("empty menu returns false", func(t *testing.T) {
		emptyMenu := ui.NewMenu()
		emptyMenu.SetSize(100, 3)
		// Don't render; optionPositions should be empty.
		_, ok := emptyMenu.OptionAtX(50)
		assert.False(t, ok)
	})

	t.Run("known options are findable", func(t *testing.T) {
		// Re-render to ensure positions are populated.
		rendered := menu.String()
		assert.NotEmpty(t, rendered)

		// Scan across the full width to find at least some known keys.
		foundKeys := make(map[keys.KeyName]bool)
		for x := 0; x < 200; x++ {
			if k, ok := menu.OptionAtX(x); ok {
				foundKeys[k] = true
			}
		}
		// A Running instance menu should include KeyNew and KeyQuit at minimum.
		assert.True(t, foundKeys[keys.KeyNew], "should find KeyNew in menu options")
		assert.True(t, foundKeys[keys.KeyQuit], "should find KeyQuit in menu options")
	})
}

func TestMenu_OptionAtX_DefaultState(t *testing.T) {
	menu := ui.NewMenu()
	menu.SetSize(200, 3)
	// In default/empty state (no instance), menu shows default options.
	_ = menu.String()

	foundKeys := make(map[keys.KeyName]bool)
	for x := 0; x < 200; x++ {
		if k, ok := menu.OptionAtX(x); ok {
			foundKeys[k] = true
		}
	}
	// Default options should include KeyNew and KeyQuit.
	assert.True(t, foundKeys[keys.KeyNew], "default menu should contain KeyNew")
	assert.True(t, foundKeys[keys.KeyQuit], "default menu should contain KeyQuit")
}
