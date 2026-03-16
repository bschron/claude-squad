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

	// Add instances in different status groups.
	// item-1: Running (group 0), item-2: Ready/Idle (group 1), item-3: Paused/Completed (group 2)
	inst1 := makeTestInstance("item-1", session.Running, "b1")
	inst2 := makeTestInstance("item-2", session.Ready, "b2")
	inst3 := makeTestInstance("item-3", session.Paused, "b3")

	list.AddInstance(inst1)
	list.AddInstance(inst2)
	list.AddInstance(inst3)

	// Layout with dividers between groups:
	//   y=0..4: header (5 lines)
	//   y=5..8: item-1 (Running group, 4 lines)
	//   y=9..11: divider to IDLE (3 lines)
	//   y=12..15: item-2 (Idle group, 4 lines)
	//   y=16..18: divider to COMPLETED (3 lines)
	//   y=19..22: item-3 (Completed group, 4 lines)

	t.Run("header area returns -1", func(t *testing.T) {
		for y := 0; y < 5; y++ {
			assert.Equal(t, -1, list.IndexAtY(y), "Y=%d should be in header", y)
		}
	})

	t.Run("first item area returns 0", func(t *testing.T) {
		for y := 5; y < 9; y++ {
			assert.Equal(t, 0, list.IndexAtY(y), "Y=%d should map to item 0", y)
		}
	})

	t.Run("divider between running and idle returns -1", func(t *testing.T) {
		for y := 9; y < 12; y++ {
			assert.Equal(t, -1, list.IndexAtY(y), "Y=%d should be in divider", y)
		}
	})

	t.Run("second item area returns 1", func(t *testing.T) {
		for y := 12; y < 16; y++ {
			assert.Equal(t, 1, list.IndexAtY(y), "Y=%d should map to item 1", y)
		}
	})

	t.Run("divider between idle and completed returns -1", func(t *testing.T) {
		for y := 16; y < 19; y++ {
			assert.Equal(t, -1, list.IndexAtY(y), "Y=%d should be in divider", y)
		}
	})

	t.Run("third item area returns 2", func(t *testing.T) {
		for y := 19; y < 23; y++ {
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

func TestList_IndexAtY_SameStatusGroup(t *testing.T) {
	sp := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	list := ui.NewList(&sp, false)
	list.SetSize(80, 60)

	// Two items in the same group — no divider between them, just normal gap.
	inst1 := makeTestInstance("run-1", session.Running, "b1")
	inst2 := makeTestInstance("run-2", session.Running, "b2")
	list.AddInstance(inst1)
	list.AddInstance(inst2)

	// Layout:
	//   y=0..4: header
	//   y=5..8: run-1 (4 lines)
	//   y=9..10: gap (2 lines, normal within-group gap)
	//   y=11..14: run-2 (4 lines)

	t.Run("first item", func(t *testing.T) {
		for y := 5; y < 9; y++ {
			assert.Equal(t, 0, list.IndexAtY(y), "Y=%d should map to item 0", y)
		}
	})

	t.Run("gap within group returns -1", func(t *testing.T) {
		for y := 9; y < 11; y++ {
			assert.Equal(t, -1, list.IndexAtY(y), "Y=%d should be in gap", y)
		}
	})

	t.Run("second item", func(t *testing.T) {
		for y := 11; y < 15; y++ {
			assert.Equal(t, 1, list.IndexAtY(y), "Y=%d should map to item 1", y)
		}
	})
}

func TestList_IndexAtY_EmptyList(t *testing.T) {
	sp := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	list := ui.NewList(&sp, false)
	list.SetSize(80, 60)

	assert.Equal(t, -1, list.IndexAtY(0))
	assert.Equal(t, -1, list.IndexAtY(5))
	assert.Equal(t, -1, list.IndexAtY(10))
}

func TestList_Navigation_CrossGroup(t *testing.T) {
	sp := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	list := ui.NewList(&sp, false)
	list.SetSize(80, 60)

	// Insert in non-display order: Paused(idx=0), Running(idx=1), Ready(idx=2)
	inst1 := makeTestInstance("paused", session.Paused, "b1")
	inst2 := makeTestInstance("running", session.Running, "b2")
	inst3 := makeTestInstance("ready", session.Ready, "b3")
	list.AddInstance(inst1)
	list.AddInstance(inst2)
	list.AddInstance(inst3)

	// Force display order to be built by rendering.
	_ = list.String()

	// Display order: Running(idx=1), Ready(idx=2), Paused(idx=0)
	// Initial selection is idx=0 (Paused), which is last in display order.

	// Select the first display item (Running, idx=1).
	list.SetSelectedIndex(1)
	assert.Equal(t, inst2, list.GetSelectedInstance(), "should start at Running")

	// Down should go to Ready (idx=2).
	list.Down()
	assert.Equal(t, inst3, list.GetSelectedInstance(), "Down from Running should go to Ready")

	// Down again should go to Paused (idx=0).
	list.Down()
	assert.Equal(t, inst1, list.GetSelectedInstance(), "Down from Ready should go to Paused")

	// Down at end should stay.
	list.Down()
	assert.Equal(t, inst1, list.GetSelectedInstance(), "Down at end should stay")

	// Up should go back to Ready (idx=2).
	list.Up()
	assert.Equal(t, inst3, list.GetSelectedInstance(), "Up from Paused should go to Ready")

	// Up again should go to Running (idx=1).
	list.Up()
	assert.Equal(t, inst2, list.GetSelectedInstance(), "Up from Ready should go to Running")

	// Up at start should stay.
	list.Up()
	assert.Equal(t, inst2, list.GetSelectedInstance(), "Up at start should stay")
}

func TestList_SetSelectedIndex(t *testing.T) {
	sp := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	list := ui.NewList(&sp, false)
	list.SetSize(80, 60)

	inst1 := makeTestInstance("sel-1", session.Running, "b1")
	inst2 := makeTestInstance("sel-2", session.Ready, "b2")
	list.AddInstance(inst1)
	list.AddInstance(inst2)

	assert.Equal(t, inst1, list.GetSelectedInstance())

	list.SetSelectedIndex(1)
	assert.Equal(t, inst2, list.GetSelectedInstance())

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

	list.SelectInstance(inst2)
	assert.Equal(t, inst2, list.GetSelectedInstance())

	list.SelectInstance(inst1)
	assert.Equal(t, inst1, list.GetSelectedInstance())
}
