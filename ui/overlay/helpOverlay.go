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
	configIndexEffort        = 0
	configIndexModel         = 1
	configIndexPermissions   = 2
	configIndexSoundAlert    = 3
	configIndexAlertSound    = 4
	configIndexInstanceLimit = 5
	configIndexAutoQuit      = 6
	configItemCount          = 7
)

// HelpOverlay combines read-only help text with an editable Configs section.
type HelpOverlay struct {
	helpContent        string
	effortPicker       *EffortPicker
	modelPicker        *ModelPicker
	permissionPicker   *PermissionModePicker
	soundToggle        *SoundToggle
	soundPicker        *SoundPicker
	instanceLimitPicker *InstanceLimitPicker
	autoQuitToggle      *AutoQuitToggle
	configMode         bool
	configIndex        int // 0=effort, 1=model, 2=permissions, 3=sound alert, 4=alert sound, 5=instance limit, 6=auto-quit
	onSave             func(config.EffortLevel, config.ModelOption, config.PermissionMode, bool, config.SoundOption, int, bool)
	width              int
}

// NewHelpOverlay creates a new help overlay with the given content and defaults.
func NewHelpOverlay(
	helpContent string,
	defaultEffort config.EffortLevel,
	defaultModel config.ModelOption,
	defaultPermissionMode config.PermissionMode,
	defaultSoundAlert bool,
	defaultAlertSound config.SoundOption,
	defaultInstanceLimit int,
	defaultAutoQuit bool,
	onSave func(config.EffortLevel, config.ModelOption, config.PermissionMode, bool, config.SoundOption, int, bool),
) *HelpOverlay {
	return &HelpOverlay{
		helpContent:         helpContent,
		effortPicker:        NewEffortPicker(defaultEffort),
		modelPicker:         NewModelPicker(defaultModel),
		permissionPicker:    NewPermissionModePicker(defaultPermissionMode),
		soundToggle:         NewSoundToggle(defaultSoundAlert),
		soundPicker:         NewSoundPicker(defaultAlertSound),
		instanceLimitPicker: NewInstanceLimitPicker(defaultInstanceLimit),
		autoQuitToggle:      NewAutoQuitToggle(defaultAutoQuit),
		onSave:              onSave,
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
	if h.permissionPicker != nil {
		h.permissionPicker.SetWidth(innerWidth)
	}
	if h.soundToggle != nil {
		h.soundToggle.SetWidth(innerWidth)
	}
	if h.soundPicker != nil {
		h.soundPicker.SetWidth(innerWidth)
	}
	if h.instanceLimitPicker != nil {
		h.instanceLimitPicker.SetWidth(innerWidth)
	}
	if h.autoQuitToggle != nil {
		h.autoQuitToggle.SetWidth(innerWidth)
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
		h.permissionPicker.Focus()
	case configIndexSoundAlert:
		h.soundToggle.Focus()
	case configIndexAlertSound:
		h.soundPicker.Focus()
	case configIndexInstanceLimit:
		h.instanceLimitPicker.Focus()
	case configIndexAutoQuit:
		h.autoQuitToggle.Focus()
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
		h.permissionPicker.Blur()
	case configIndexSoundAlert:
		h.soundToggle.Blur()
	case configIndexAlertSound:
		h.soundPicker.Blur()
	case configIndexInstanceLimit:
		h.instanceLimitPicker.Blur()
	case configIndexAutoQuit:
		h.autoQuitToggle.Blur()
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
			// Skip sound picker if sound alert is off
			if h.configIndex == configIndexAlertSound && !h.soundToggle.GetEnabled() {
				h.configIndex = configIndexSoundAlert
			}
			h.focusCurrentConfig()
			return false
		case tea.KeyDown:
			h.blurCurrentConfig()
			h.configIndex = (h.configIndex + 1) % configItemCount
			// Skip sound picker if sound alert is off
			if h.configIndex == configIndexAlertSound && !h.soundToggle.GetEnabled() {
				h.configIndex = (h.configIndex + 1) % configItemCount
			}
			h.focusCurrentConfig()
			return false
		case tea.KeyLeft, tea.KeyRight:
			switch h.configIndex {
			case configIndexEffort:
				h.effortPicker.HandleKeyPress(msg)
			case configIndexModel:
				h.modelPicker.HandleKeyPress(msg)
			case configIndexPermissions:
				h.permissionPicker.HandleKeyPress(msg)
			case configIndexSoundAlert:
				h.soundToggle.HandleKeyPress(msg)
			case configIndexAlertSound:
				h.soundPicker.HandleKeyPress(msg)
			case configIndexInstanceLimit:
				h.instanceLimitPicker.HandleKeyPress(msg)
			case configIndexAutoQuit:
				h.autoQuitToggle.HandleKeyPress(msg)
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
			h.permissionPicker.GetSelectedMode(),
			h.soundToggle.GetEnabled(),
			h.soundPicker.GetSelectedSound(),
			h.instanceLimitPicker.GetSelectedLimit(),
			h.autoQuitToggle.GetEnabled(),
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

	// Permission mode picker
	if h.configMode && h.configIndex == configIndexPermissions {
		b.WriteString(hoActiveIndicator.Render("> "))
	} else if h.configMode {
		b.WriteString("  ")
	}
	b.WriteString(h.permissionPicker.Render())
	b.WriteString("\n\n")

	// Sound alert toggle
	if h.configMode && h.configIndex == configIndexSoundAlert {
		b.WriteString(hoActiveIndicator.Render("> "))
	} else if h.configMode {
		b.WriteString("  ")
	}
	b.WriteString(h.soundToggle.Render())
	b.WriteString("\n\n")

	// Alert sound picker (only visible when sound alert is enabled)
	if h.soundToggle.GetEnabled() {
		if h.configMode && h.configIndex == configIndexAlertSound {
			b.WriteString(hoActiveIndicator.Render("> "))
		} else if h.configMode {
			b.WriteString("  ")
		}
		b.WriteString(h.soundPicker.Render())
		b.WriteString("\n\n")
	}

	// Instance limit picker
	if h.configMode && h.configIndex == configIndexInstanceLimit {
		b.WriteString(hoActiveIndicator.Render("> "))
	} else if h.configMode {
		b.WriteString("  ")
	}
	b.WriteString(h.instanceLimitPicker.Render())
	b.WriteString("\n\n")

	// Auto-quit interactive toggle
	if h.configMode && h.configIndex == configIndexAutoQuit {
		b.WriteString(hoActiveIndicator.Render("> "))
	} else if h.configMode {
		b.WriteString("  ")
	}
	b.WriteString(h.autoQuitToggle.Render())
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
