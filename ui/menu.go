package ui

import (
	"claude-squad/keys"
	"strings"

	"claude-squad/session"

	"github.com/charmbracelet/lipgloss"
)

var keyStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
	Light: "#655F5F",
	Dark:  "#7F7A7A",
})

var descStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
	Light: "#7A7474",
	Dark:  "#9C9494",
})

var sepStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
	Light: "#DDDADA",
	Dark:  "#3C3C3C",
})

var actionGroupStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("99"))

var separator = " • "
var verticalSeparator = " │ "

var menuStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("205"))

// MenuState represents different states the menu can be in
type MenuState int

const (
	StateDefault MenuState = iota
	StateEmpty
	StateNewInstance
	StatePrompt
)

// optionPosition stores the X-range of a rendered menu option for click detection.
type optionPosition struct {
	startX int
	endX   int
	key    keys.KeyName
}

// menuGroup defines a contiguous range of menu options.
type menuGroup struct {
	start, end int
}

type Menu struct {
	options       []keys.KeyName
	groups        []menuGroup // dynamic group boundaries for separators
	actionGroupIdx int       // which group index is the action group (-1 for none)
	height, width int
	state         MenuState
	instance      *session.Instance
	activeTab     int
	kanbanVisible bool

	// keyDown is the key which is pressed. The default is -1.
	keyDown keys.KeyName

	// optionPositions caches positions from the last String() render.
	optionPositions []optionPosition
}

var defaultMenuOptions = []keys.KeyName{keys.KeyNew, keys.KeyPrompt, keys.KeyHelp, keys.KeyQuit}
var newInstanceMenuOptions = []keys.KeyName{keys.KeySubmitName}
var promptMenuOptions = []keys.KeyName{keys.KeySubmitName}

func NewMenu() *Menu {
	return &Menu{
		options:        defaultMenuOptions,
		groups:         nil,
		actionGroupIdx: -1,
		state:          StateEmpty,
		activeTab:      0,
		keyDown:        -1,
	}
}

// SetKanbanVisible updates kanban visibility and refreshes menu options.
func (m *Menu) SetKanbanVisible(visible bool) {
	m.kanbanVisible = visible
	m.updateOptions()
}

func (m *Menu) Keydown(name keys.KeyName) {
	m.keyDown = name
}

func (m *Menu) ClearKeydown() {
	m.keyDown = -1
}

// SetState updates the menu state and options accordingly
func (m *Menu) SetState(state MenuState) {
	m.state = state
	m.updateOptions()
}

// SetInstance updates the current instance and refreshes menu options
func (m *Menu) SetInstance(instance *session.Instance) {
	m.instance = instance
	// Only change the state if we're not in a special state (NewInstance or Prompt)
	if m.state != StateNewInstance && m.state != StatePrompt {
		if m.instance != nil {
			m.state = StateDefault
		} else {
			m.state = StateEmpty
		}
	}
	m.updateOptions()
}

// SetActiveTab updates the currently active tab
func (m *Menu) SetActiveTab(tab int) {
	m.activeTab = tab
	m.updateOptions()
}

// updateOptions updates the menu options based on current state and instance
func (m *Menu) updateOptions() {
	switch m.state {
	case StateEmpty:
		m.options = defaultMenuOptions
	case StateDefault:
		if m.instance != nil {
			// When there is an instance, show that instance's options
			m.addInstanceOptions()
		} else {
			// When there is no instance, show the empty state
			m.options = defaultMenuOptions
		}
	case StateNewInstance:
		m.options = newInstanceMenuOptions
	case StatePrompt:
		m.options = promptMenuOptions
	}
}

func (m *Menu) addInstanceOptions() {
	// Loading instances only get minimal options
	if m.instance != nil && m.instance.Status == session.Loading {
		m.options = []keys.KeyName{keys.KeyNew, keys.KeyHelp, keys.KeyQuit}
		m.groups = nil
		m.actionGroupIdx = -1
		return
	}

	// Instance management group
	instanceGroup := []keys.KeyName{keys.KeyNew, keys.KeyKill}

	// Action group
	actionGroup := []keys.KeyName{keys.KeyEnter, keys.KeySubmit}
	if m.instance.Status == session.Paused {
		actionGroup = append(actionGroup, keys.KeyResume)
	} else if !m.instance.TmuxAlive() {
		actionGroup = append(actionGroup, keys.KeyResume)
	} else {
		actionGroup = append(actionGroup, keys.KeyCheckout)
	}

	// Navigation group (when in diff tab)
	if m.activeTab == DiffTab || m.activeTab == TerminalTab {
		actionGroup = append(actionGroup, keys.KeyShiftUp)
	}

	// Kanban navigation group (when kanban is visible)
	var kanbanGroup []keys.KeyName
	if m.kanbanVisible {
		kanbanGroup = []keys.KeyName{keys.KeyLeft, keys.KeyRight, keys.KeyYank}
	}

	// System group
	systemGroup := []keys.KeyName{keys.KeyKanban, keys.KeyTab, keys.KeyHelp, keys.KeyQuit}

	// Build options and compute group boundaries
	options := make([]keys.KeyName, 0, len(instanceGroup)+len(actionGroup)+len(kanbanGroup)+len(systemGroup))
	var groups []menuGroup

	g1Start := len(options)
	options = append(options, instanceGroup...)
	groups = append(groups, menuGroup{g1Start, len(options)})

	g2Start := len(options)
	options = append(options, actionGroup...)
	groups = append(groups, menuGroup{g2Start, len(options)})
	m.actionGroupIdx = 1

	if len(kanbanGroup) > 0 {
		g3Start := len(options)
		options = append(options, kanbanGroup...)
		groups = append(groups, menuGroup{g3Start, len(options)})
	}

	g4Start := len(options)
	options = append(options, systemGroup...)
	groups = append(groups, menuGroup{g4Start, len(options)})

	m.options = options
	m.groups = groups
}

// SetSize sets the width of the window. The menu will be centered horizontally within this width.
func (m *Menu) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *Menu) String() string {
	var s strings.Builder

	// Track option positions for click detection.
	m.optionPositions = nil
	cursor := 0

	for i, k := range m.options {
		binding := keys.GlobalkeyBindings[k]

		var (
			localActionStyle = actionGroupStyle
			localKeyStyle    = keyStyle
			localDescStyle   = descStyle
		)
		if m.keyDown == k {
			localActionStyle = localActionStyle.Underline(true)
			localKeyStyle = localKeyStyle.Underline(true)
			localDescStyle = localDescStyle.Underline(true)
		}

		var inActionGroup bool
		switch m.state {
		case StateEmpty:
			inActionGroup = i <= 1
		default:
			if m.actionGroupIdx >= 0 && m.actionGroupIdx < len(m.groups) {
				ag := m.groups[m.actionGroupIdx]
				inActionGroup = i >= ag.start && i < ag.end
			}
		}

		startPos := cursor
		if inActionGroup {
			s.WriteString(localActionStyle.Render(binding.Help().Key))
			cursor += len(binding.Help().Key)
			s.WriteString(" ")
			cursor++
			s.WriteString(localActionStyle.Render(binding.Help().Desc))
			cursor += len(binding.Help().Desc)
		} else {
			s.WriteString(localKeyStyle.Render(binding.Help().Key))
			cursor += len(binding.Help().Key)
			s.WriteString(" ")
			cursor++
			s.WriteString(localDescStyle.Render(binding.Help().Desc))
			cursor += len(binding.Help().Desc)
		}
		m.optionPositions = append(m.optionPositions, optionPosition{
			startX: startPos,
			endX:   cursor,
			key:    k,
		})

		// Add appropriate separator
		if i != len(m.options)-1 {
			isGroupEnd := false
			for _, group := range m.groups {
				if i == group.end-1 {
					s.WriteString(sepStyle.Render(verticalSeparator))
					cursor += len(verticalSeparator)
					isGroupEnd = true
					break
				}
			}
			if !isGroupEnd {
				s.WriteString(sepStyle.Render(separator))
				cursor += len(separator)
			}
		}
	}

	centeredMenuText := menuStyle.Render(s.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, centeredMenuText)
}

// OptionAtX returns the key name for the menu option at the given X
// coordinate (local to the menu panel). The X position accounts for the
// centered rendering of the menu text within the menu width.
func (m *Menu) OptionAtX(localX int) (keys.KeyName, bool) {
	if len(m.optionPositions) == 0 {
		return 0, false
	}

	// The menu text is centered in m.width. Compute the left offset.
	totalTextWidth := 0
	if len(m.optionPositions) > 0 {
		totalTextWidth = m.optionPositions[len(m.optionPositions)-1].endX
	}
	leftPad := (m.width - totalTextWidth) / 2
	if leftPad < 0 {
		leftPad = 0
	}

	adjustedX := localX - leftPad
	for _, op := range m.optionPositions {
		if adjustedX >= op.startX && adjustedX < op.endX {
			return op.key, true
		}
	}
	return 0, false
}
