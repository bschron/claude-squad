package overlay

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	projPickerStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2)

	projPickerTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("62"))

	projPickerCheckStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("62"))

	projPickerCurrentStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Italic(true)

	projPickerFooterStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Italic(true)

	projPickerCursorStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("62"))
)

const maxVisibleProjects = 8

// ProjectPickerItem represents a project entry in the picker.
type ProjectPickerItem struct {
	RepoPath     string
	DisplayName  string
	SessionCount int
	IsCurrent    bool
	Selected     bool
}

// ProjectPicker is a multi-select overlay for choosing projects.
type ProjectPicker struct {
	items     []ProjectPickerItem
	cursor    int
	width     int
	scrollOff int
}

// NewProjectPicker creates a new project picker overlay.
func NewProjectPicker(items []ProjectPickerItem) *ProjectPicker {
	return &ProjectPicker{
		items: items,
		width: 50,
	}
}

// HandleKeyPress processes a key press. Returns (shouldClose, confirmed).
func (p *ProjectPicker) HandleKeyPress(msg tea.KeyMsg) (bool, bool) {
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

	switch msg.String() {
	case " ":
		if !p.items[p.cursor].IsCurrent {
			p.items[p.cursor].Selected = !p.items[p.cursor].Selected
		}
	}

	return false, false
}

func (p *ProjectPicker) moveUp() {
	if p.cursor > 0 {
		p.cursor--
		if p.cursor < p.scrollOff {
			p.scrollOff = p.cursor
		}
	}
}

func (p *ProjectPicker) moveDown() {
	if p.cursor < len(p.items)-1 {
		p.cursor++
		if p.cursor >= p.scrollOff+maxVisibleProjects {
			p.scrollOff = p.cursor - maxVisibleProjects + 1
		}
	}
}

// GetSelectedPaths returns repo paths of selected non-current items.
func (p *ProjectPicker) GetSelectedPaths() []string {
	var paths []string
	for _, item := range p.items {
		if item.Selected && !item.IsCurrent {
			paths = append(paths, item.RepoPath)
		}
	}
	return paths
}

// SetWidth sets the rendering width.
func (p *ProjectPicker) SetWidth(w int) {
	p.width = w
}

// Render renders the project picker overlay.
func (p *ProjectPicker) Render() string {
	var b strings.Builder

	b.WriteString(projPickerTitleStyle.Render("Projects"))
	b.WriteString("\n\n")

	visibleEnd := p.scrollOff + maxVisibleProjects
	if visibleEnd > len(p.items) {
		visibleEnd = len(p.items)
	}

	for i := p.scrollOff; i < visibleEnd; i++ {
		item := p.items[i]

		cursor := "  "
		if i == p.cursor {
			cursor = projPickerCursorStyle.Render("> ")
		}

		checkbox := "[ ]"
		if item.Selected || item.IsCurrent {
			checkbox = projPickerCheckStyle.Render("[x]")
		}

		label := item.DisplayName
		if item.IsCurrent {
			label = projPickerCurrentStyle.Render(fmt.Sprintf("%s (current)", item.DisplayName))
		}

		sessionText := fmt.Sprintf(" - %d session", item.SessionCount)
		if item.SessionCount != 1 {
			sessionText += "s"
		}

		b.WriteString(fmt.Sprintf("%s%s %s%s", cursor, checkbox, label, sessionText))
		if i < visibleEnd-1 {
			b.WriteString("\n")
		}
	}

	if len(p.items) > maxVisibleProjects {
		b.WriteString("\n")
		b.WriteString(projPickerFooterStyle.Render(fmt.Sprintf("  (%d/%d shown)", maxVisibleProjects, len(p.items))))
	}

	b.WriteString("\n\n")
	b.WriteString(projPickerFooterStyle.Render("↑/↓: navigate  space: toggle  enter: save  esc: cancel"))

	return projPickerStyle.Width(p.width).Render(b.String())
}
