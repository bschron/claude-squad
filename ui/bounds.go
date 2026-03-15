package ui

// Rect represents a rectangular area for mouse hit testing
type Rect struct {
	X, Y, Width, Height int
}

// Contains returns true if the point (x, y) is inside the rectangle
func (r Rect) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.Width && y >= r.Y && y < r.Y+r.Height
}
