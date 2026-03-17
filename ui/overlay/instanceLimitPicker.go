package overlay

import (
	"claude-squad/config"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// InstanceLimitPicker is an embeddable component for selecting an instance limit.
type InstanceLimitPicker struct {
	options []int
	cursor  int
	focused bool
	width   int
}

// NewInstanceLimitPicker creates a new instance limit picker with the cursor set to the given default.
func NewInstanceLimitPicker(defaultLimit int) *InstanceLimitPicker {
	p := &InstanceLimitPicker{
		options: config.ValidInstanceLimits,
	}
	for i, v := range p.options {
		if v == defaultLimit {
			p.cursor = i
			return p
		}
	}
	// If not found in options, find closest
	for i, v := range p.options {
		if v >= defaultLimit {
			p.cursor = i
			return p
		}
	}
	p.cursor = len(p.options) - 1
	return p
}

// Focus gives the picker focus.
func (p *InstanceLimitPicker) Focus() {
	p.focused = true
}

// Blur removes focus from the picker.
func (p *InstanceLimitPicker) Blur() {
	p.focused = false
}

// SetWidth sets the rendering width.
func (p *InstanceLimitPicker) SetWidth(w int) {
	p.width = w
}

// HandleKeyPress processes a key event. Returns true if consumed.
func (p *InstanceLimitPicker) HandleKeyPress(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyLeft:
		if p.cursor > 0 {
			p.cursor--
		}
		return true
	case tea.KeyRight:
		if p.cursor < len(p.options)-1 {
			p.cursor++
		}
		return true
	}
	return false
}

// GetSelectedLimit returns the currently selected instance limit.
func (p *InstanceLimitPicker) GetSelectedLimit() int {
	if p.cursor < 0 || p.cursor >= len(p.options) {
		return config.DefaultInstanceLimit
	}
	return p.options[p.cursor]
}

// Render renders the instance limit picker.
func (p *InstanceLimitPicker) Render() string {
	var s strings.Builder
	s.WriteString(epLabelStyle.Render("Instance Limit"))

	if p.focused {
		s.WriteString(epDimStyle.Render("  ←/→ to change"))
	}
	s.WriteString("\n\n")

	for i, v := range p.options {
		label := fmt.Sprintf(" %d ", v)
		if i == p.cursor && p.focused {
			s.WriteString(epSelectedStyle.Render(label))
		} else if i == p.cursor {
			s.WriteString(label)
		} else {
			s.WriteString(epDimStyle.Render(label))
		}
		if i < len(p.options)-1 {
			s.WriteString(epDimStyle.Render(" | "))
		}
	}

	return s.String()
}
