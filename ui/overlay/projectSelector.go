package overlay

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const maxVisibleSelectorItems = 8

// ProjectSelectorItem represents a project entry in the single-select dialog.
type ProjectSelectorItem struct {
	RepoPath    string
	DisplayName string
	IsCurrent   bool
}

// ProjectSelectorOverlay is a single-select overlay for choosing a project.
type ProjectSelectorOverlay struct {
	items     []ProjectSelectorItem
	cursor    int
	width     int
	scrollOff int
}

// NewProjectSelectorOverlay creates a new project selector overlay.
func NewProjectSelectorOverlay(items []ProjectSelectorItem) *ProjectSelectorOverlay {
	return &ProjectSelectorOverlay{
		items: items,
		width: 50,
	}
}

// HandleKeyPress processes a key press. Returns (shouldClose, confirmed).
func (p *ProjectSelectorOverlay) HandleKeyPress(msg tea.KeyMsg) (bool, bool) {
	switch msg.Type {
	case tea.KeyUp:
		p.moveUp()
		return false, false
	case tea.KeyDown:
		p.moveDown()
		return false, false
	case tea.KeyEnter:
		return true, true
	case tea.KeyEsc:
		return true, false
	}
	return false, false
}

func (p *ProjectSelectorOverlay) moveUp() {
	if p.cursor > 0 {
		p.cursor--
		if p.cursor < p.scrollOff {
			p.scrollOff = p.cursor
		}
	}
}

func (p *ProjectSelectorOverlay) moveDown() {
	if p.cursor < len(p.items)-1 {
		p.cursor++
		if p.cursor >= p.scrollOff+maxVisibleSelectorItems {
			p.scrollOff = p.cursor - maxVisibleSelectorItems + 1
		}
	}
}

// GetSelectedPath returns the repo path at the current cursor position.
func (p *ProjectSelectorOverlay) GetSelectedPath() string {
	if p.cursor >= 0 && p.cursor < len(p.items) {
		return p.items[p.cursor].RepoPath
	}
	return ""
}

// SetWidth sets the rendering width.
func (p *ProjectSelectorOverlay) SetWidth(w int) {
	p.width = w
}

// Render renders the project selector overlay.
func (p *ProjectSelectorOverlay) Render() string {
	var b strings.Builder

	b.WriteString(projPickerTitleStyle.Render("Select Project"))
	b.WriteString("\n\n")

	visibleEnd := p.scrollOff + maxVisibleSelectorItems
	if visibleEnd > len(p.items) {
		visibleEnd = len(p.items)
	}

	for i := p.scrollOff; i < visibleEnd; i++ {
		item := p.items[i]

		cursor := "  "
		if i == p.cursor {
			cursor = projPickerCursorStyle.Render("> ")
		}

		label := item.DisplayName
		if item.IsCurrent {
			label = projPickerCurrentStyle.Render(fmt.Sprintf("%s (current)", item.DisplayName))
		}

		b.WriteString(fmt.Sprintf("%s%s", cursor, label))
		if i < visibleEnd-1 {
			b.WriteString("\n")
		}
	}

	if len(p.items) > maxVisibleSelectorItems {
		b.WriteString("\n")
		b.WriteString(projPickerFooterStyle.Render(fmt.Sprintf("  (%d/%d shown)", maxVisibleSelectorItems, len(p.items))))
	}

	b.WriteString("\n\n")
	b.WriteString(projPickerFooterStyle.Render("↑/↓: navigate  enter: select  esc: cancel"))

	return projPickerStyle.Width(p.width).Render(b.String())
}
