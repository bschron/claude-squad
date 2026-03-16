package overlay

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PermissionToggle is an embeddable on/off toggle for --dangerously-skip-permissions.
type PermissionToggle struct {
	skipPerms bool
	focused   bool
	width     int
}

// NewPermissionToggle creates a new permission toggle with the given default.
func NewPermissionToggle(defaultSkipPerms bool) *PermissionToggle {
	return &PermissionToggle{
		skipPerms: defaultSkipPerms,
	}
}

// Focus gives the toggle focus.
func (pt *PermissionToggle) Focus() {
	pt.focused = true
}

// Blur removes focus from the toggle.
func (pt *PermissionToggle) Blur() {
	pt.focused = false
}

// SetWidth sets the rendering width.
func (pt *PermissionToggle) SetWidth(w int) {
	pt.width = w
}

// HandleKeyPress processes a key event. Returns true if consumed.
func (pt *PermissionToggle) HandleKeyPress(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyLeft, tea.KeyRight:
		pt.skipPerms = !pt.skipPerms
		return true
	}
	return false
}

// GetSkipPermissions returns the current toggle value.
func (pt *PermissionToggle) GetSkipPermissions() bool {
	return pt.skipPerms
}

// SetSkipPermissions sets the toggle value.
func (pt *PermissionToggle) SetSkipPermissions(v bool) {
	pt.skipPerms = v
}

var (
	ptLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("62")).
			Bold(true)

	ptSelectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("0"))

	ptDimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
)

// Render renders the permission toggle.
func (pt *PermissionToggle) Render() string {
	var s strings.Builder
	s.WriteString(ptLabelStyle.Render("Skip Permissions"))

	if pt.focused {
		s.WriteString(ptDimStyle.Render("  \u2190/\u2192 to change"))
	}
	s.WriteString("\n\n")

	onLabel := " on "
	offLabel := " off "

	if pt.skipPerms {
		if pt.focused {
			s.WriteString(ptSelectedStyle.Render(onLabel))
		} else {
			s.WriteString(onLabel)
		}
		s.WriteString(ptDimStyle.Render(" | "))
		s.WriteString(ptDimStyle.Render(offLabel))
	} else {
		s.WriteString(ptDimStyle.Render(onLabel))
		s.WriteString(ptDimStyle.Render(" | "))
		if pt.focused {
			s.WriteString(ptSelectedStyle.Render(offLabel))
		} else {
			s.WriteString(offLabel)
		}
	}

	return s.String()
}
