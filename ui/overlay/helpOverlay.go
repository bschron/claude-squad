package overlay

import (
	"claude-squad/config"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	hoStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2)

	hoDividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	hoFooterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)

	hoHeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("62")).
			Bold(true)

	hoActiveIndicator = lipgloss.NewStyle().
				Foreground(lipgloss.Color("62")).
				Bold(true)
)

const (
	configIndexEffort      = 0
	configIndexModel       = 1
	configIndexPermissions = 2
	configItemCount        = 3
)

// HelpOverlay combines read-only help text with an editable Configs section.
type HelpOverlay struct {
	helpContent      string
	effortPicker     *EffortPicker
	modelPicker      *ModelPicker
	permissionToggle *PermissionToggle
	configMode       bool
	configIndex      int // 0=effort, 1=model, 2=permissions
	onSave           func(config.EffortLevel, config.ModelOption, bool)
	width            int
}

// NewHelpOverlay creates a new help overlay with the given content and defaults.
func NewHelpOverlay(
	helpContent string,
	defaultEffort config.EffortLevel,
	defaultModel config.ModelOption,
	defaultSkipPerms bool,
	onSave func(config.EffortLevel, config.ModelOption, bool),
) *HelpOverlay {
	return &HelpOverlay{
		helpContent:      helpContent,
		effortPicker:     NewEffortPicker(defaultEffort),
		modelPicker:      NewModelPicker(defaultModel),
		permissionToggle: NewPermissionToggle(defaultSkipPerms),
		onSave:           onSave,
	}
}

// SetWidth sets the rendering width.
func (h *HelpOverlay) SetWidth(width int) {
	h.width = width
	innerWidth := width - 6
	if h.effortPicker != nil {
		h.effortPicker.SetWidth(innerWidth)
	}
	if h.modelPicker != nil {
		h.modelPicker.SetWidth(innerWidth)
	}
	if h.permissionToggle != nil {
		h.permissionToggle.SetWidth(innerWidth)
	}
}

// focusCurrentConfig focuses the config item at configIndex.
func (h *HelpOverlay) focusCurrentConfig() {
	switch h.configIndex {
	case configIndexEffort:
		h.effortPicker.Focus()
	case configIndexModel:
		h.modelPicker.Focus()
	case configIndexPermissions:
		h.permissionToggle.Focus()
	}
}

// blurCurrentConfig blurs the config item at configIndex.
func (h *HelpOverlay) blurCurrentConfig() {
	switch h.configIndex {
	case configIndexEffort:
		h.effortPicker.Blur()
	case configIndexModel:
		h.modelPicker.Blur()
	case configIndexPermissions:
		h.permissionToggle.Blur()
	}
}

// HandleKeyPress processes a key press. Returns true if the overlay should close.
func (h *HelpOverlay) HandleKeyPress(msg tea.KeyMsg) bool {
	if h.configMode {
		switch msg.Type {
		case tea.KeyTab, tea.KeyEsc:
			// Exit config mode
			h.configMode = false
			h.blurCurrentConfig()
			return false
		case tea.KeyUp:
			h.blurCurrentConfig()
			h.configIndex = (h.configIndex - 1 + configItemCount) % configItemCount
			h.focusCurrentConfig()
			return false
		case tea.KeyDown:
			h.blurCurrentConfig()
			h.configIndex = (h.configIndex + 1) % configItemCount
			h.focusCurrentConfig()
			return false
		case tea.KeyLeft, tea.KeyRight:
			switch h.configIndex {
			case configIndexEffort:
				h.effortPicker.HandleKeyPress(msg)
			case configIndexModel:
				h.modelPicker.HandleKeyPress(msg)
			case configIndexPermissions:
				h.permissionToggle.HandleKeyPress(msg)
			}
			return false
		default:
			return false
		}
	}

	// View mode
	if msg.Type == tea.KeyTab {
		h.configMode = true
		h.configIndex = configIndexEffort
		h.focusCurrentConfig()
		return false
	}

	// Any other key dismisses
	if h.onSave != nil {
		h.onSave(
			h.effortPicker.GetSelectedEffort(),
			h.modelPicker.GetSelectedModel(),
			h.permissionToggle.GetSkipPermissions(),
		)
	}
	return true
}

// Render renders the help overlay.
func (h *HelpOverlay) Render() string {
	innerWidth := h.width - 6
	if innerWidth < 1 {
		innerWidth = 1
	}

	divider := hoDividerStyle.Render(strings.Repeat("\u2500", innerWidth))

	var b strings.Builder
	b.WriteString(h.helpContent)
	b.WriteString("\n\n")
	b.WriteString(divider)
	b.WriteString("\n\n")
	b.WriteString(hoHeaderStyle.Render("Configs"))
	b.WriteString("\n\n")

	// Effort picker
	if h.configMode && h.configIndex == configIndexEffort {
		b.WriteString(hoActiveIndicator.Render("> "))
	} else if h.configMode {
		b.WriteString("  ")
	}
	b.WriteString(h.effortPicker.Render())
	b.WriteString("\n\n")

	// Model picker
	if h.configMode && h.configIndex == configIndexModel {
		b.WriteString(hoActiveIndicator.Render("> "))
	} else if h.configMode {
		b.WriteString("  ")
	}
	b.WriteString(h.modelPicker.Render())
	b.WriteString("\n\n")

	// Permission toggle
	if h.configMode && h.configIndex == configIndexPermissions {
		b.WriteString(hoActiveIndicator.Render("> "))
	} else if h.configMode {
		b.WriteString("  ")
	}
	b.WriteString(h.permissionToggle.Render())
	b.WriteString("\n\n")

	b.WriteString(divider)
	b.WriteString("\n\n")

	if h.configMode {
		b.WriteString(hoFooterStyle.Render("tab/esc: done | \u2191/\u2193: select config | \u2190/\u2192: change value"))
	} else {
		b.WriteString(hoFooterStyle.Render("tab: edit configs | any key: close"))
	}

	return hoStyle.Width(h.width).Render(b.String())
}
