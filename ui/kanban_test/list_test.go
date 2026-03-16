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

	// Layout with group headers:
	//   y=0..4: header (5 lines)
	//   y=5..6: RUNNING label (2 lines: label + \n)
	//   y=7..10: item-1 (Running group, 4 lines)
	//   y=11..13: IDLE divider (3 lines: \n + label + \n)
	//   y=14..17: item-2 (Idle group, 4 lines)
	//   y=18..20: COMPLETED divider (3 lines: \n + label + \n)
	//   y=21..24: item-3 (Completed group, 4 lines)

	t.Run("header area returns -1", func(t *testing.T) {
		for y := 0; y < 5; y++ {
			assert.Equal(t, -1, list.IndexAtY(y), "Y=%d should be in header", y)
		}
	})

	t.Run("first group label returns -1", func(t *testing.T) {
		for y := 5; y < 7; y++ {
			assert.Equal(t, -1, list.IndexAtY(y), "Y=%d should be in group label", y)
		}
	})

	t.Run("first item area returns 0", func(t *testing.T) {
		for y := 7; y < 11; y++ {
			assert.Equal(t, 0, list.IndexAtY(y), "Y=%d should map to item 0", y)
		}
	})

	t.Run("divider between running and idle returns -1", func(t *testing.T) {
		for y := 11; y < 14; y++ {
			assert.Equal(t, -1, list.IndexAtY(y), "Y=%d should be in divider", y)
		}
	})

	t.Run("second item area returns 1", func(t *testing.T) {
		for y := 14; y < 18; y++ {
			assert.Equal(t, 1, list.IndexAtY(y), "Y=%d should map to item 1", y)
		}
	})

	t.Run("divider between idle and completed returns -1", func(t *testing.T) {
		for y := 18; y < 21; y++ {
			assert.Equal(t, -1, list.IndexAtY(y), "Y=%d should be in divider", y)
		}
	})

	t.Run("third item area returns 2", func(t *testing.T) {
		for y := 21; y < 25; y++ {
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

	// Two items in the same group — group header + normal gap between items.
	inst1 := makeTestInstance("run-1", session.Running, "b1")
	inst2 := makeTestInstance("run-2", session.Running, "b2")
	list.AddInstance(inst1)
	list.AddInstance(inst2)

	// Layout:
	//   y=0..4: header
	//   y=5..6: RUNNING label (2 lines)
	//   y=7..10: run-1 (4 lines)
	//   y=11..12: gap (2 lines, normal within-group gap)
	//   y=13..16: run-2 (4 lines)

	t.Run("first item", func(t *testing.T) {
		for y := 7; y < 11; y++ {
			assert.Equal(t, 0, list.IndexAtY(y), "Y=%d should map to item 0", y)
		}
	})

	t.Run("gap within group returns -1", func(t *testing.T) {
		for y := 11; y < 13; y++ {
			assert.Equal(t, -1, list.IndexAtY(y), "Y=%d should be in gap", y)
		}
	})

	t.Run("second item", func(t *testing.T) {
		for y := 13; y < 17; y++ {
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
	list.SetSelectedIndex(1)
	assert.Equal(t, inst2, list.GetSelectedInstance(), "should start at Running")

	list.Down()
	assert.Equal(t, inst3, list.GetSelectedInstance(), "Down from Running should go to Ready")

	list.Down()
	assert.Equal(t, inst1, list.GetSelectedInstance(), "Down from Ready should go to Paused")

	list.Down()
	assert.Equal(t, inst1, list.GetSelectedInstance(), "Down at end should stay")

	list.Up()
	assert.Equal(t, inst3, list.GetSelectedInstance(), "Up from Paused should go to Ready")

	list.Up()
	assert.Equal(t, inst2, list.GetSelectedInstance(), "Up from Ready should go to Running")

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
