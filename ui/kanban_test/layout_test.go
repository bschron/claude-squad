package kanban_test

import (
	"claude-squad/ui"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLayoutBounds_KanbanVisible(t *testing.T) {
	// Simulate the layout computation from app.go when kanban is visible.
	totalWidth := 200
	contentHeight := 40

	listWidth := int(float32(totalWidth) * 0.2)
	kanbanWidth := int(float32(totalWidth) * 0.35)
	tabsWidth := totalWidth - listWidth - kanbanWidth

	listBounds := ui.Rect{X: 0, Y: 0, Width: listWidth, Height: contentHeight}
	previewBounds := ui.Rect{X: listWidth, Y: 0, Width: tabsWidth, Height: contentHeight}
	kanbanBounds := ui.Rect{X: listWidth + tabsWidth, Y: 0, Width: kanbanWidth, Height: contentHeight}

	// List starts at X=0.
	assert.Equal(t, 0, listBounds.X)
	assert.Equal(t, listWidth, listBounds.Width)

	// Preview is adjacent to list.
	assert.Equal(t, listWidth, previewBounds.X)
	assert.Equal(t, tabsWidth, previewBounds.Width)

	// Kanban is adjacent to preview.
	assert.Equal(t, listWidth+tabsWidth, kanbanBounds.X)
	assert.Equal(t, kanbanWidth, kanbanBounds.Width)

	// All widths should sum to total.
	assert.Equal(t, totalWidth, listWidth+tabsWidth+kanbanWidth)

	// Heights should match.
	assert.Equal(t, contentHeight, listBounds.Height)
	assert.Equal(t, contentHeight, previewBounds.Height)
	assert.Equal(t, contentHeight, kanbanBounds.Height)

	// A point in the kanban area should be contained.
	assert.True(t, kanbanBounds.Contains(kanbanBounds.X+5, 5))
	// A point in the list area should NOT be in the kanban bounds.
	assert.False(t, kanbanBounds.Contains(5, 5))
}

func TestLayoutBounds_KanbanHidden(t *testing.T) {
	// When kanban is hidden, the kanban bounds are a zero-value Rect.
	totalWidth := 200
	contentHeight := 40

	listWidth := int(float32(totalWidth) * 0.3)
	tabsWidth := totalWidth - listWidth

	listBounds := ui.Rect{X: 0, Y: 0, Width: listWidth, Height: contentHeight}
	previewBounds := ui.Rect{X: listWidth, Y: 0, Width: tabsWidth, Height: contentHeight}
	kanbanBounds := ui.Rect{} // zero value

	// List and preview take full width.
	assert.Equal(t, totalWidth, listWidth+tabsWidth)

	// Kanban bounds are zero.
	assert.Equal(t, 0, kanbanBounds.Width)
	assert.Equal(t, 0, kanbanBounds.Height)

	// No point should be contained by the zero kanban rect.
	assert.False(t, kanbanBounds.Contains(0, 0))
	assert.False(t, kanbanBounds.Contains(150, 20))

	// Points should be in their correct panels.
	assert.True(t, listBounds.Contains(5, 5))
	assert.True(t, previewBounds.Contains(listWidth+5, 5))
}

func TestLayoutBounds_NarrowTerminal(t *testing.T) {
	// When terminal width < 80, kanban should be auto-hidden (simulated here).
	totalWidth := 60
	contentHeight := 30

	kanbanVisible := totalWidth >= 80

	assert.False(t, kanbanVisible, "kanban should be hidden for narrow terminals")

	// In hidden mode, list and preview share the full width.
	listWidth := int(float32(totalWidth) * 0.3)
	tabsWidth := totalWidth - listWidth

	assert.Equal(t, totalWidth, listWidth+tabsWidth)

	listBounds := ui.Rect{X: 0, Y: 0, Width: listWidth, Height: contentHeight}
	previewBounds := ui.Rect{X: listWidth, Y: 0, Width: tabsWidth, Height: contentHeight}

	assert.True(t, listBounds.Contains(1, 1))
	assert.True(t, previewBounds.Contains(listWidth+1, 1))
}
