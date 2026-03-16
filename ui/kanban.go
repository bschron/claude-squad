package ui

import (
	"claude-squad/session"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

// cardBound tracks the screen position of a rendered card for click detection.
type cardBound struct {
	rect     Rect
	instance *session.Instance
}

// KanbanBoard renders instances organized into status columns.
type KanbanBoard struct {
	width, height int
	columns       [3][]*session.Instance // [Running, Idle, Completed]
	scrollOffset  [3]int
	selectedInst  *session.Instance
	spinner       *spinner.Model
	cardBounds    []cardBound
	cursorCol     int // 0-2: which column
	cursorIdx     int // index within column
}

// NewKanbanBoard creates a new kanban board panel.
func NewKanbanBoard(spinner *spinner.Model) *KanbanBoard {
	return &KanbanBoard{
		spinner: spinner,
	}
}

// SetSize sets the rendering dimensions of the board.
func (kb *KanbanBoard) SetSize(width, height int) {
	kb.width = width
	kb.height = height
}

// UpdateInstances classifies instances into columns based on their status and
// updates the selected instance highlight based on the cursor position.
func (kb *KanbanBoard) UpdateInstances(instances []*session.Instance, selected *session.Instance) {
	// Save the instance at the current cursor position before reclassifying
	var cursorInst *session.Instance
	if kb.cursorCol >= 0 && kb.cursorCol < 3 && kb.cursorIdx >= 0 && kb.cursorIdx < len(kb.columns[kb.cursorCol]) {
		cursorInst = kb.columns[kb.cursorCol][kb.cursorIdx]
	}

	// Clear columns
	kb.columns = [3][]*session.Instance{}

	for _, inst := range instances {
		switch inst.Status {
		case session.Running, session.Loading:
			kb.columns[0] = append(kb.columns[0], inst)
		case session.Ready:
			kb.columns[1] = append(kb.columns[1], inst)
		case session.Paused:
			kb.columns[2] = append(kb.columns[2], inst)
		}
	}

	// Restore cursor position: find the same instance after reclassification
	if cursorInst != nil {
		found := false
		for col := 0; col < 3; col++ {
			for idx, inst := range kb.columns[col] {
				if inst == cursorInst {
					kb.cursorCol = col
					kb.cursorIdx = idx
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			kb.clampCursor()
		}
	} else {
		kb.clampCursor()
	}

	// Set selectedInst from cursor position
	if kb.cursorCol >= 0 && kb.cursorCol < 3 && kb.cursorIdx >= 0 && kb.cursorIdx < len(kb.columns[kb.cursorCol]) {
		kb.selectedInst = kb.columns[kb.cursorCol][kb.cursorIdx]
	} else {
		kb.selectedInst = selected
	}

	// Clamp scroll offsets and auto-scroll to keep cursor visible
	for i := 0; i < 3; i++ {
		if kb.scrollOffset[i] >= len(kb.columns[i]) {
			kb.scrollOffset[i] = 0
		}
		if i == kb.cursorCol && kb.selectedInst != nil {
			if kb.cursorIdx < kb.scrollOffset[i] {
				kb.scrollOffset[i] = kb.cursorIdx
			}
		}
	}
}

// columnHeaders are the display names for each column.
var columnHeaders = [3]string{"RUNNING", "IDLE", "COMPLETED"}

// String renders the kanban board as a string.
func (kb *KanbanBoard) String() string {
	if kb.width == 0 || kb.height == 0 {
		return ""
	}

	// Reset card bounds for click detection
	kb.cardBounds = nil

	colWidth := kb.width / 3
	lastColWidth := kb.width - colWidth*2

	var cols []string
	for i := 0; i < 3; i++ {
		w := colWidth
		if i == 2 {
			w = lastColWidth
		}
		cols = append(cols, kb.renderColumn(i, w))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, cols...)
}

// renderColumn renders a single column with its header and cards.
func (kb *KanbanBoard) renderColumn(colIdx, width int) string {
	count := len(kb.columns[colIdx])
	header := fmt.Sprintf(" %s [%d] ", columnHeaders[colIdx], count)

	headerStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Bold(true).
		Width(width).
		AlignHorizontal(lipgloss.Center)

	renderedHeader := headerStyle.Render(header)

	// Card area height = total height minus header (1 line) minus spacing (1 line)
	cardAreaHeight := kb.height - 3

	// Render cards
	var cardLines []string
	instances := kb.columns[colIdx]
	cardWidth := width - 2 // padding on each side

	if cardWidth < 10 {
		cardWidth = 10
	}

	visibleStart := kb.scrollOffset[colIdx]

	usedHeight := 0
	for idx := visibleStart; idx < len(instances); idx++ {
		inst := instances[idx]
		card := renderCard(inst, inst == kb.selectedInst, cardWidth, kb.spinner)
		cardH := lipgloss.Height(card)

		if usedHeight+cardH > cardAreaHeight && usedHeight > 0 {
			break
		}

		// Track card bounds for click detection.
		colX := colIdx * (kb.width / 3)
		cardY := 2 + usedHeight // 1 header line + 1 spacing line

		kb.cardBounds = append(kb.cardBounds, cardBound{
			rect:     Rect{X: colX + 1, Y: cardY, Width: cardWidth, Height: cardH},
			instance: inst,
		})

		cardLines = append(cardLines, card)
		usedHeight += cardH
	}

	body := strings.Join(cardLines, "\n")

	column := lipgloss.JoinVertical(lipgloss.Left,
		renderedHeader,
		"",
		body,
	)

	return lipgloss.Place(width, kb.height, lipgloss.Left, lipgloss.Top, column)
}

// HandleClick tests the given local coordinates against recorded card bounds
// and returns the matching instance (if any).
func (kb *KanbanBoard) HandleClick(localX, localY int) *session.Instance {
	for _, cb := range kb.cardBounds {
		if cb.rect.Contains(localX, localY) {
			return cb.instance
		}
	}
	return nil
}

// ScrollColumn scrolls the given column index by delta lines.
func (kb *KanbanBoard) ScrollColumn(colIdx, delta int) {
	if colIdx < 0 || colIdx > 2 {
		return
	}
	kb.scrollOffset[colIdx] += delta
	if kb.scrollOffset[colIdx] < 0 {
		kb.scrollOffset[colIdx] = 0
	}
	max := len(kb.columns[colIdx]) - 1
	if max < 0 {
		max = 0
	}
	if kb.scrollOffset[colIdx] > max {
		kb.scrollOffset[colIdx] = max
	}
}

// ColumnAtX returns the column index (0-2) for the given local X coordinate.
func (kb *KanbanBoard) ColumnAtX(localX int) int {
	if kb.width == 0 {
		return 0
	}
	colWidth := kb.width / 3
	col := localX / colWidth
	if col > 2 {
		col = 2
	}
	return col
}

// clampCursor ensures cursorCol and cursorIdx are within valid bounds.
func (kb *KanbanBoard) clampCursor() {
	// Find a non-empty column starting from cursorCol
	if kb.cursorCol < 0 {
		kb.cursorCol = 0
	}
	if kb.cursorCol > 2 {
		kb.cursorCol = 2
	}
	colLen := len(kb.columns[kb.cursorCol])
	if colLen == 0 {
		// Try to find any non-empty column
		for i := 0; i < 3; i++ {
			if len(kb.columns[i]) > 0 {
				kb.cursorCol = i
				colLen = len(kb.columns[i])
				break
			}
		}
	}
	if kb.cursorIdx < 0 {
		kb.cursorIdx = 0
	}
	if colLen > 0 && kb.cursorIdx >= colLen {
		kb.cursorIdx = colLen - 1
	}
}

// CursorUp moves the cursor up within the current column.
func (kb *KanbanBoard) CursorUp() {
	if kb.cursorIdx > 0 {
		kb.cursorIdx--
	}
	// Adjust scroll offset to keep cursor visible
	if kb.cursorIdx < kb.scrollOffset[kb.cursorCol] {
		kb.scrollOffset[kb.cursorCol] = kb.cursorIdx
	}
	kb.syncSelectedFromCursor()
}

// CursorDown moves the cursor down within the current column.
func (kb *KanbanBoard) CursorDown() {
	colLen := len(kb.columns[kb.cursorCol])
	if kb.cursorIdx < colLen-1 {
		kb.cursorIdx++
	}
	kb.syncSelectedFromCursor()
}

// CursorLeft moves the cursor to the left column, skipping empty columns.
func (kb *KanbanBoard) CursorLeft() {
	for col := kb.cursorCol - 1; col >= 0; col-- {
		if len(kb.columns[col]) > 0 {
			kb.cursorCol = col
			if kb.cursorIdx >= len(kb.columns[col]) {
				kb.cursorIdx = len(kb.columns[col]) - 1
			}
			kb.syncSelectedFromCursor()
			return
		}
	}
}

// CursorRight moves the cursor to the right column, skipping empty columns.
func (kb *KanbanBoard) CursorRight() {
	for col := kb.cursorCol + 1; col <= 2; col++ {
		if len(kb.columns[col]) > 0 {
			kb.cursorCol = col
			if kb.cursorIdx >= len(kb.columns[col]) {
				kb.cursorIdx = len(kb.columns[col]) - 1
			}
			kb.syncSelectedFromCursor()
			return
		}
	}
}

// syncSelectedFromCursor updates selectedInst to match the current cursor position.
func (kb *KanbanBoard) syncSelectedFromCursor() {
	if kb.cursorCol >= 0 && kb.cursorCol < 3 {
		col := kb.columns[kb.cursorCol]
		if kb.cursorIdx >= 0 && kb.cursorIdx < len(col) {
			kb.selectedInst = col[kb.cursorIdx]
		}
	}
}

// GetCursorInstance returns the instance at the current cursor position, or nil.
func (kb *KanbanBoard) GetCursorInstance() *session.Instance {
	if kb.cursorCol < 0 || kb.cursorCol > 2 {
		return nil
	}
	col := kb.columns[kb.cursorCol]
	if kb.cursorIdx < 0 || kb.cursorIdx >= len(col) {
		return nil
	}
	return col[kb.cursorIdx]
}

// SetCursorToInstance finds the given instance across columns and sets the cursor to it.
func (kb *KanbanBoard) SetCursorToInstance(inst *session.Instance) {
	if inst == nil {
		return
	}
	for col := 0; col < 3; col++ {
		for idx, candidate := range kb.columns[col] {
			if candidate == inst {
				kb.cursorCol = col
				kb.cursorIdx = idx
				kb.selectedInst = inst
				return
			}
		}
	}
}
