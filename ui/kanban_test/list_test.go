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

	// Layout with group headers (first divider absorbed into header at split[4]):
	//   y=0..4: header (5 lines, includes first group divider)
	//   y=5..8: item-1 (Running group, 4 lines)
	//   y=9: IDLE divider (1 line)
	//   y=10..13: item-2 (Idle group, 4 lines)
	//   y=14: COMPLETED divider (1 line)
	//   y=15..18: item-3 (Completed group, 4 lines)

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
		assert.Equal(t, -1, list.IndexAtY(9), "Y=9 should be in divider")
	})

	t.Run("second item area returns 1", func(t *testing.T) {
		for y := 10; y < 14; y++ {
			assert.Equal(t, 1, list.IndexAtY(y), "Y=%d should map to item 1", y)
		}
	})

	t.Run("divider between idle and completed returns -1", func(t *testing.T) {
		assert.Equal(t, -1, list.IndexAtY(14), "Y=14 should be in divider")
	})

	t.Run("third item area returns 2", func(t *testing.T) {
		for y := 15; y < 19; y++ {
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

	// Layout (first divider absorbed into header at split[4]):
	//   y=0..4: header (includes RUNNING label)
	//   y=5..8: run-1 (4 lines)
	//   y=9: gap (1 line)
	//   y=10..13: run-2 (4 lines)

	t.Run("first item", func(t *testing.T) {
		for y := 5; y < 9; y++ {
			assert.Equal(t, 0, list.IndexAtY(y), "Y=%d should map to item 0", y)
		}
	})

	t.Run("gap within group returns -1", func(t *testing.T) {
		assert.Equal(t, -1, list.IndexAtY(9), "Y=9 should be in gap")
	})

	t.Run("second item", func(t *testing.T) {
		for y := 10; y < 14; y++ {
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
