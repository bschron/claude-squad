package overlay

import (
	"claude-squad/config"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// EffortPicker is an embeddable component for selecting an effort level.
// It displays a horizontal selector with left/right arrow navigation.
type EffortPicker struct {
	levels  []config.EffortLevel
	cursor  int
	focused bool
	width   int
}

// NewEffortPicker creates a new effort picker with the cursor set to the given default.
func NewEffortPicker(defaultEffort config.EffortLevel) *EffortPicker {
	ep := &EffortPicker{
		levels: config.ValidEffortLevels,
	}
	// Set cursor to match default
	for i, l := range ep.levels {
		if l == defaultEffort {
			ep.cursor = i
			break
		}
	}
	return ep
}

// Focus gives the effort picker focus.
func (ep *EffortPicker) Focus() {
	ep.focused = true
}

// Blur removes focus from the effort picker.
func (ep *EffortPicker) Blur() {
	ep.focused = false
}

// SetWidth sets the rendering width.
func (ep *EffortPicker) SetWidth(w int) {
	ep.width = w
}

// HandleKeyPress processes a key event. Returns true if consumed.
func (ep *EffortPicker) HandleKeyPress(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyLeft:
		if ep.cursor > 0 {
			ep.cursor--
		}
		return true
	case tea.KeyRight:
		if ep.cursor < len(ep.levels)-1 {
			ep.cursor++
		}
		return true
	}
	return false
}

// GetSelectedEffort returns the currently selected effort level.
func (ep *EffortPicker) GetSelectedEffort() config.EffortLevel {
	if ep.cursor < 0 || ep.cursor >= len(ep.levels) {
		return config.DefaultEffortLevel
	}
	return ep.levels[ep.cursor]
}

// SetEffort sets the cursor to the given effort level.
func (ep *EffortPicker) SetEffort(effort config.EffortLevel) {
	for i, l := range ep.levels {
		if l == effort {
			ep.cursor = i
			return
		}
	}
}

var (
	epLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("62")).
			Bold(true)

	epSelectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("0"))

	epDimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
)

// Render renders the effort picker.
func (ep *EffortPicker) Render() string {
	var s strings.Builder
	s.WriteString(epLabelStyle.Render("Effort"))

	if ep.focused {
		s.WriteString(epDimStyle.Render("  \u2190/\u2192 to change"))
	}
	s.WriteString("\n\n")

	for i, l := range ep.levels {
		label := " " + string(l) + " "
		if i == ep.cursor && ep.focused {
			s.WriteString(epSelectedStyle.Render(label))
		} else if i == ep.cursor {
			s.WriteString(label)
		} else {
			s.WriteString(epDimStyle.Render(label))
		}
		if i < len(ep.levels)-1 {
			s.WriteString(epDimStyle.Render(" | "))
		}
	}

	return s.String()
}
