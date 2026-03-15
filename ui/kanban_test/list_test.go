package kanban_test

import (
	"claude-squad/session"
	"claude-squad/ui"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/stretchr/testify/assert"
)

func TestList_IndexAtY(t *testing.T) {
	sp := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	list := ui.NewList(&sp, false)
	list.SetSize(80, 60)

	// Add some instances.
	inst1 := makeTestInstance("item-1", session.Running, "b1")
	inst2 := makeTestInstance("item-2", session.Ready, "b2")
	inst3 := makeTestInstance("item-3", session.Paused, "b3")

	list.AddInstance(inst1)
	list.AddInstance(inst2)
	list.AddInstance(inst3)

	t.Run("header area returns -1", func(t *testing.T) {
		// The first 5 lines are header (2 blank + 1 title + 2 blank).
		for y := 0; y < 5; y++ {
			assert.Equal(t, -1, list.IndexAtY(y), "Y=%d should be in header", y)
		}
	})

	t.Run("first item area returns 0", func(t *testing.T) {
		// First item starts at y=5, spans 4 lines (stride 6 with gap 2).
		for y := 5; y < 9; y++ {
			assert.Equal(t, 0, list.IndexAtY(y), "Y=%d should map to item 0", y)
		}
	})

	t.Run("gap between items returns -1", func(t *testing.T) {
		// Gap after first item: y=9 and y=10.
		for y := 9; y < 11; y++ {
			assert.Equal(t, -1, list.IndexAtY(y), "Y=%d should be in gap", y)
		}
	})

	t.Run("second item area returns 1", func(t *testing.T) {
		// Second item starts at y=11, spans 4 lines.
		for y := 11; y < 15; y++ {
			assert.Equal(t, 1, list.IndexAtY(y), "Y=%d should map to item 1", y)
		}
	})

	t.Run("third item area returns 2", func(t *testing.T) {
		// Third item starts at y=17, spans 4 lines.
		for y := 17; y < 21; y++ {
			assert.Equal(t, 2, list.IndexAtY(y), "Y=%d should map to item 2", y)
		}
	})

	t.Run("far below all items returns -1", func(t *testing.T) {
		assert.Equal(t, -1, list.IndexAtY(100))
	})

	t.Run("negative Y returns -1", func(t *testing.T) {
		assert.Equal(t, -1, list.IndexAtY(-5))
	})
}

func TestList_IndexAtY_EmptyList(t *testing.T) {
	sp := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	list := ui.NewList(&sp, false)
	list.SetSize(80, 60)

	// No instances added.
	assert.Equal(t, -1, list.IndexAtY(0))
	assert.Equal(t, -1, list.IndexAtY(5))
	assert.Equal(t, -1, list.IndexAtY(10))
}

func TestList_SetSelectedIndex(t *testing.T) {
	sp := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	list := ui.NewList(&sp, false)
	list.SetSize(80, 60)

	inst1 := makeTestInstance("sel-1", session.Running, "b1")
	inst2 := makeTestInstance("sel-2", session.Ready, "b2")
	list.AddInstance(inst1)
	list.AddInstance(inst2)

	// Default selection is index 0.
	assert.Equal(t, inst1, list.GetSelectedInstance())

	// Set to index 1.
	list.SetSelectedIndex(1)
	assert.Equal(t, inst2, list.GetSelectedInstance())

	// Out of bounds should be a no-op.
	list.SetSelectedIndex(100)
	assert.Equal(t, inst2, list.GetSelectedInstance(), "out-of-bounds SetSelectedIndex should be no-op")
}

func TestList_SelectInstance(t *testing.T) {
	sp := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	list := ui.NewList(&sp, false)
	list.SetSize(80, 60)

	inst1 := makeTestInstance("find-1", session.Running, "b1")
	inst2 := makeTestInstance("find-2", session.Ready, "b2")
	list.AddInstance(inst1)
	list.AddInstance(inst2)

	// Select by pointer.
	list.SelectInstance(inst2)
	assert.Equal(t, inst2, list.GetSelectedInstance())

	list.SelectInstance(inst1)
	assert.Equal(t, inst1, list.GetSelectedInstance())
}
