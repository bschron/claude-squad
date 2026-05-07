package overlay

import (
	"claude-squad/config"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PermissionModePicker is a 3-state picker for selecting the session's
// permission mode (normal / bypass / auto). It mirrors the EffortPicker
// horizontal-selector pattern.
type PermissionModePicker struct {
	modes   []config.PermissionMode
	cursor  int
	focused bool
	width   int
}

// NewPermissionModePicker creates a new picker with the cursor set to the given default.
func NewPermissionModePicker(defaultMode config.PermissionMode) *PermissionModePicker {
	pp := &PermissionModePicker{
		modes: config.ValidPermissionModes,
	}
	for i, m := range pp.modes {
		if m == defaultMode {
			pp.cursor = i
			break
		}
	}
	return pp
}

func (pp *PermissionModePicker) Focus() {
	pp.focused = true
}

func (pp *PermissionModePicker) Blur() {
	pp.focused = false
}

func (pp *PermissionModePicker) SetWidth(w int) {
	pp.width = w
}

// HandleKeyPress processes a key event. Returns true if consumed.
func (pp *PermissionModePicker) HandleKeyPress(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyLeft:
		if pp.cursor > 0 {
			pp.cursor--
		}
		return true
	case tea.KeyRight:
		if pp.cursor < len(pp.modes)-1 {
			pp.cursor++
		}
		return true
	}
	return false
}

// GetSelectedMode returns the currently selected permission mode.
func (pp *PermissionModePicker) GetSelectedMode() config.PermissionMode {
	if pp.cursor < 0 || pp.cursor >= len(pp.modes) {
		return config.DefaultPermissionMode
	}
	return pp.modes[pp.cursor]
}

// SetMode positions the cursor at the given mode.
func (pp *PermissionModePicker) SetMode(mode config.PermissionMode) {
	for i, m := range pp.modes {
		if m == mode {
			pp.cursor = i
			return
		}
	}
}

var (
	pmpLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("62")).
			Bold(true)

	pmpSelectedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("62")).
				Foreground(lipgloss.Color("0"))

	pmpDimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
)

// Render renders the permission mode picker.
func (pp *PermissionModePicker) Render() string {
	var s strings.Builder
	s.WriteString(pmpLabelStyle.Render("Permission Mode"))

	if pp.focused {
		s.WriteString(pmpDimStyle.Render("  ←/→ to change"))
	}
	s.WriteString("\n\n")

	for i, m := range pp.modes {
		label := " " + string(m) + " "
		if i == pp.cursor && pp.focused {
			s.WriteString(pmpSelectedStyle.Render(label))
		} else if i == pp.cursor {
			s.WriteString(label)
		} else {
			s.WriteString(pmpDimStyle.Render(label))
		}
		if i < len(pp.modes)-1 {
			s.WriteString(pmpDimStyle.Render(" | "))
		}
	}

	return s.String()
}
