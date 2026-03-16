package overlay

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SoundToggle is an embeddable on/off toggle for sound alerts.
type SoundToggle struct {
	enabled bool
	focused bool
	width   int
}

// NewSoundToggle creates a new sound toggle with the given default.
func NewSoundToggle(defaultEnabled bool) *SoundToggle {
	return &SoundToggle{
		enabled: defaultEnabled,
	}
}

// Focus gives the toggle focus.
func (st *SoundToggle) Focus() {
	st.focused = true
}

// Blur removes focus from the toggle.
func (st *SoundToggle) Blur() {
	st.focused = false
}

// SetWidth sets the rendering width.
func (st *SoundToggle) SetWidth(w int) {
	st.width = w
}

// HandleKeyPress processes a key event. Returns true if consumed.
func (st *SoundToggle) HandleKeyPress(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyLeft, tea.KeyRight:
		st.enabled = !st.enabled
		return true
	}
	return false
}

// GetEnabled returns the current toggle value.
func (st *SoundToggle) GetEnabled() bool {
	return st.enabled
}

var (
	stLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("62")).
			Bold(true)

	stSelectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("0"))

	stDimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
)

// Render renders the sound toggle.
func (st *SoundToggle) Render() string {
	var s strings.Builder
	s.WriteString(stLabelStyle.Render("Sound Alert"))

	if st.focused {
		s.WriteString(stDimStyle.Render("  ←/→ to change"))
	}
	s.WriteString("\n\n")

	onLabel := " on "
	offLabel := " off "

	if st.enabled {
		if st.focused {
			s.WriteString(stSelectedStyle.Render(onLabel))
		} else {
			s.WriteString(onLabel)
		}
		s.WriteString(stDimStyle.Render(" | "))
		s.WriteString(stDimStyle.Render(offLabel))
	} else {
		s.WriteString(stDimStyle.Render(onLabel))
		s.WriteString(stDimStyle.Render(" | "))
		if st.focused {
			s.WriteString(stSelectedStyle.Render(offLabel))
		} else {
			s.WriteString(offLabel)
		}
	}

	return s.String()
}
