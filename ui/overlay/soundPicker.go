package overlay

import (
	"claude-squad/config"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SoundPicker is an embeddable component for selecting an alert sound.
type SoundPicker struct {
	options []config.SoundOption
	cursor  int
	focused bool
	width   int
}

// NewSoundPicker creates a new sound picker with the cursor set to the given default.
func NewSoundPicker(defaultSound config.SoundOption) *SoundPicker {
	sp := &SoundPicker{
		options: config.ValidSoundOptions,
	}
	for i, o := range sp.options {
		if o == defaultSound {
			sp.cursor = i
			break
		}
	}
	return sp
}

// Focus gives the sound picker focus.
func (sp *SoundPicker) Focus() {
	sp.focused = true
}

// Blur removes focus from the sound picker.
func (sp *SoundPicker) Blur() {
	sp.focused = false
}

// SetWidth sets the rendering width.
func (sp *SoundPicker) SetWidth(w int) {
	sp.width = w
}

// HandleKeyPress processes a key event. Returns true if consumed.
func (sp *SoundPicker) HandleKeyPress(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyLeft:
		if sp.cursor > 0 {
			sp.cursor--
		}
		return true
	case tea.KeyRight:
		if sp.cursor < len(sp.options)-1 {
			sp.cursor++
		}
		return true
	}
	return false
}

// GetSelectedSound returns the currently selected sound option.
func (sp *SoundPicker) GetSelectedSound() config.SoundOption {
	if sp.cursor < 0 || sp.cursor >= len(sp.options) {
		return config.DefaultSound
	}
	return sp.options[sp.cursor]
}

var (
	spLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("62")).
			Bold(true)

	spSelectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("0"))

	spDimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
)

// Render renders the sound picker.
func (sp *SoundPicker) Render() string {
	var s strings.Builder
	s.WriteString(spLabelStyle.Render("Alert Sound"))

	if sp.focused {
		s.WriteString(spDimStyle.Render("  ←/→ to change"))
	}
	s.WriteString("\n\n")

	for i, o := range sp.options {
		label := " " + config.SoundDisplayLabels[o] + " "
		if i == sp.cursor && sp.focused {
			s.WriteString(spSelectedStyle.Render(label))
		} else if i == sp.cursor {
			s.WriteString(label)
		} else {
			s.WriteString(spDimStyle.Render(label))
		}
		if i < len(sp.options)-1 {
			s.WriteString(spDimStyle.Render(" | "))
		}
	}

	return s.String()
}
