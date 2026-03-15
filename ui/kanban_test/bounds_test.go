package kanban_test

import (
	"claude-squad/ui"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRect_Contains(t *testing.T) {
	r := ui.Rect{X: 10, Y: 20, Width: 30, Height: 15}

	tests := []struct {
		name   string
		x, y   int
		expect bool
	}{
		{"inside center", 25, 27, true},
		{"top-left corner (inclusive)", 10, 20, true},
		{"right edge (exclusive)", 40, 25, false},
		{"bottom edge (exclusive)", 25, 35, false},
		{"just inside right", 39, 25, true},
		{"just inside bottom", 25, 34, true},
		{"left of rect", 9, 25, false},
		{"above rect", 25, 19, false},
		{"far outside", 100, 100, false},
		{"negative coords", -1, -1, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expect, r.Contains(tc.x, tc.y))
		})
	}
}

func TestRect_Contains_ZeroSize(t *testing.T) {
	r := ui.Rect{X: 5, Y: 5, Width: 0, Height: 0}

	// A zero-sized rect should contain nothing.
	assert.False(t, r.Contains(5, 5), "zero-width and zero-height rect should contain no point")
	assert.False(t, r.Contains(0, 0))
	assert.False(t, r.Contains(6, 6))
}

func TestRect_Contains_UnitSize(t *testing.T) {
	r := ui.Rect{X: 3, Y: 7, Width: 1, Height: 1}

	assert.True(t, r.Contains(3, 7), "unit rect should contain its own origin")
	assert.False(t, r.Contains(4, 7), "unit rect should not contain the next column")
	assert.False(t, r.Contains(3, 8), "unit rect should not contain the next row")
}
