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
	action   string // empty for card body, "send"/"open"/"stop"/"resume"/"delete" for buttons
}

// KanbanBoard renders instances organized into status columns.
type KanbanBoard struct {
	width, height int
	columns       [3][]*session.Instance // [Running, Idle, Completed]
	scrollOffset  [3]int
	selectedInst  *session.Instance
	spinner       *spinner.Model
	cardBounds    []cardBound
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
// updates the selected instance highlight.
func (kb *KanbanBoard) UpdateInstances(instances []*session.Instance, selected *session.Instance) {
	// Clear columns
	kb.columns = [3][]*session.Instance{}
	kb.selectedInst = selected

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

	// Clamp scroll offsets and auto-scroll to keep selected visible
	for i := 0; i < 3; i++ {
		if kb.scrollOffset[i] >= len(kb.columns[i]) {
			kb.scrollOffset[i] = 0
		}
		// If the selected instance is in this column, make sure it's visible
		if selected != nil {
			for idx, inst := range kb.columns[i] {
				if inst == selected {
					if idx < kb.scrollOffset[i] {
						kb.scrollOffset[i] = idx
					}
					// We can't know exactly how many cards fit without rendering,
					// but scrolling to the selected index is a safe bet.
					break
				}
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

		// The last line of the card interior is the button row.
		// Card has 1 border top + 4 content lines + 1 border bottom = 6 lines.
		// Button row is at cardY + cardH - 2 (1 border bottom + 0-indexed).
		buttonY := cardY + cardH - 2
		if buttonY < cardY {
			buttonY = cardY
		}

		// Register button bounds based on instance status.
		// Buttons are laid out inside the card: border(1) + padding(1) = offset 2
		btnX := colX + 3 // border + padding
		buttonDefs := cardButtonDefs(inst)
		for _, bd := range buttonDefs {
			btnW := len(bd.label) + 2 // padding(0,1) adds 2 chars
			kb.cardBounds = append(kb.cardBounds, cardBound{
				rect:     Rect{X: btnX, Y: buttonY, Width: btnW, Height: 1},
				instance: inst,
				action:   bd.action,
			})
			btnX += btnW + 1 // +1 for space separator
		}

		// Register the card body bound (covers the whole card for general selection)
		kb.cardBounds = append(kb.cardBounds, cardBound{
			rect:     Rect{X: colX + 1, Y: cardY, Width: cardWidth, Height: cardH},
			instance: inst,
			action:   "",
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

// buttonDef describes a button label and its action identifier.
type buttonDef struct {
	label  string
	action string
}

// cardButtonDefs returns the button definitions for a given instance status.
func cardButtonDefs(inst *session.Instance) []buttonDef {
	switch inst.Status {
	case session.Running, session.Loading:
		return []buttonDef{
			{label: "SEND", action: "send"},
			{label: "OPEN", action: "open"},
			{label: "STOP", action: "stop"},
		}
	case session.Ready:
		return []buttonDef{
			{label: "SEND", action: "send"},
			{label: "OPEN", action: "open"},
			{label: "STOP", action: "stop"},
		}
	case session.Paused:
		return []buttonDef{
			{label: "RESUME", action: "resume"},
			{label: "DELETE", action: "delete"},
		}
	}
	return nil
}

// HandleClick tests the given local coordinates against recorded card bounds
// and returns the matching instance and action (if any).
func (kb *KanbanBoard) HandleClick(localX, localY int) (*session.Instance, string) {
	for _, cb := range kb.cardBounds {
		if cb.rect.Contains(localX, localY) {
			return cb.instance, cb.action
		}
	}
	return nil, ""
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
