package overlay

import (
	"claude-squad/config"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ModelPicker is an embeddable component for selecting a model option.
// It displays a horizontal selector with left/right arrow navigation.
type ModelPicker struct {
	options []config.ModelOption
	cursor  int
	focused bool
	width   int
}

// NewModelPicker creates a new model picker with the cursor set to the given default.
func NewModelPicker(defaultModel config.ModelOption) *ModelPicker {
	mp := &ModelPicker{
		options: config.ValidModelOptions,
	}
	for i, o := range mp.options {
		if o == defaultModel {
			mp.cursor = i
			break
		}
	}
	return mp
}

// Focus gives the model picker focus.
func (mp *ModelPicker) Focus() {
	mp.focused = true
}

// Blur removes focus from the model picker.
func (mp *ModelPicker) Blur() {
	mp.focused = false
}

// SetWidth sets the rendering width.
func (mp *ModelPicker) SetWidth(w int) {
	mp.width = w
}

// HandleKeyPress processes a key event. Returns true if consumed.
func (mp *ModelPicker) HandleKeyPress(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyLeft:
		if mp.cursor > 0 {
			mp.cursor--
		}
		return true
	case tea.KeyRight:
		if mp.cursor < len(mp.options)-1 {
			mp.cursor++
		}
		return true
	}
	return false
}

// GetSelectedModel returns the currently selected model option.
func (mp *ModelPicker) GetSelectedModel() config.ModelOption {
	if mp.cursor < 0 || mp.cursor >= len(mp.options) {
		return config.ModelDefault
	}
	return mp.options[mp.cursor]
}

// SetModel sets the cursor to the given model option.
func (mp *ModelPicker) SetModel(model config.ModelOption) {
	for i, o := range mp.options {
		if o == model {
			mp.cursor = i
			return
		}
	}
}

var (
	mpLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("62")).
			Bold(true)

	mpSelectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("0"))

	mpDimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
)

// Render renders the model picker.
func (mp *ModelPicker) Render() string {
	var s strings.Builder
	s.WriteString(mpLabelStyle.Render("Model"))

	if mp.focused {
		s.WriteString(mpDimStyle.Render("  \u2190/\u2192 to change"))
	}
	s.WriteString("\n\n")

	for i, o := range mp.options {
		label := " " + config.ModelDisplayLabels[o] + " "
		if i == mp.cursor && mp.focused {
			s.WriteString(mpSelectedStyle.Render(label))
		} else if i == mp.cursor {
			s.WriteString(label)
		} else {
			s.WriteString(mpDimStyle.Render(label))
		}
		if i < len(mp.options)-1 {
			s.WriteString(mpDimStyle.Render(" | "))
		}
	}

	return s.String()
}
