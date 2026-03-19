package overlay

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// AutoQuitToggle is an embeddable on/off toggle for auto-quit interactive mode.
type AutoQuitToggle struct {
	enabled bool
	focused bool
	width   int
}

// NewAutoQuitToggle creates a new auto-quit toggle with the given default.
func NewAutoQuitToggle(defaultEnabled bool) *AutoQuitToggle {
	return &AutoQuitToggle{
		enabled: defaultEnabled,
	}
}

// Focus gives the toggle focus.
func (aq *AutoQuitToggle) Focus() {
	aq.focused = true
}

// Blur removes focus from the toggle.
func (aq *AutoQuitToggle) Blur() {
	aq.focused = false
}

// SetWidth sets the rendering width.
func (aq *AutoQuitToggle) SetWidth(w int) {
	aq.width = w
}

// HandleKeyPress processes a key event. Returns true if consumed.
func (aq *AutoQuitToggle) HandleKeyPress(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyLeft, tea.KeyRight:
		aq.enabled = !aq.enabled
		return true
	}
	return false
}

// GetEnabled returns the current toggle value.
func (aq *AutoQuitToggle) GetEnabled() bool {
	return aq.enabled
}

// Render renders the auto-quit toggle.
func (aq *AutoQuitToggle) Render() string {
	var s strings.Builder
	s.WriteString(stLabelStyle.Render("Auto-Quit Interactive"))

	if aq.focused {
		s.WriteString(stDimStyle.Render("  ←/→ to change"))
	}
	s.WriteString("\n\n")

	onLabel := " on "
	offLabel := " off "

	if aq.enabled {
		if aq.focused {
			s.WriteString(stSelectedStyle.Render(onLabel))
		} else {
			s.WriteString(onLabel)
		}
		s.WriteString(stDimStyle.Render(" | "))
		s.WriteString(stDimStyle.Render(offLabel))
	} else {
		s.WriteString(stDimStyle.Render(onLabel))
		s.WriteString(stDimStyle.Render(" | "))
		if aq.focused {
			s.WriteString(stSelectedStyle.Render(offLabel))
		} else {
			s.WriteString(offLabel)
		}
	}

	return s.String()
}
