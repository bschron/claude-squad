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
	StateInteractive
	StateNotes
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

var defaultMenuOptions = []keys.KeyName{keys.KeyNew, keys.KeyPrompt, keys.KeyProjectPicker, keys.KeyHelp, keys.KeyQuit}
var newInstanceMenuOptions = []keys.KeyName{keys.KeySubmitName}
var promptMenuOptions = []keys.KeyName{keys.KeySubmitName}
var interactiveMenuOptions = []keys.KeyName{}

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
	if m.state != StateNewInstance && m.state != StatePrompt && m.state != StateInteractive && m.state != StateNotes {
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
	// Reset groups upfront; addInstanceOptions() will set them when appropriate.
	m.groups = nil
	m.actionGroupIdx = -1

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
	case StateInteractive:
		m.options = interactiveMenuOptions
	case StateNotes:
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
	instanceGroup := []keys.KeyName{keys.KeyNew, keys.KeyKill, keys.KeyNotes}

	// Action group
	actionGroup := []keys.KeyName{keys.KeyEnter, keys.KeyInteractive, keys.KeySubmit}
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
	systemGroup := []keys.KeyName{keys.KeyKanban, keys.KeyTab, keys.KeyProjectPicker, keys.KeyHelp, keys.KeyQuit}

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

// renderItem renders a single menu item and returns its styled text and plain-text width.
func (m *Menu) renderItem(i int, k keys.KeyName) (string, int) {
	binding := keys.GlobalkeyBindings[k]

	localActionStyle := actionGroupStyle
	localKeyStyle := keyStyle
	localDescStyle := descStyle
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

	var text string
	if inActionGroup {
		text = localActionStyle.Render(binding.Help().Key) + " " + localActionStyle.Render(binding.Help().Desc)
	} else {
		text = localKeyStyle.Render(binding.Help().Key) + " " + localDescStyle.Render(binding.Help().Desc)
	}
	width := len(binding.Help().Key) + 1 + len(binding.Help().Desc)
	return text, width
}

func (m *Menu) String() string {
	if m.state == StateInteractive {
		hint := descStyle.Render("ctrl+q") + sepStyle.Render(" exit interactive")
		centeredHint := menuStyle.Render(hint)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, centeredHint)
	}

	// Track option positions for click detection.
	m.optionPositions = nil

	// Determine effective groups. If none defined, treat all items as one group.
	effectiveGroups := m.groups
	if len(effectiveGroups) == 0 {
		effectiveGroups = []menuGroup{{0, len(m.options)}}
	}

	// Render each group as a segment with its styled text and plain-text width.
	type groupSegment struct {
		text  string
		width int
	}
	segments := make([]groupSegment, len(effectiveGroups))
	for gi, g := range effectiveGroups {
		var text string
		width := 0
		for i := g.start; i < g.end; i++ {
			itemText, itemWidth := m.renderItem(i, m.options[i])
			text += itemText
			width += itemWidth
			if i < g.end-1 {
				text += sepStyle.Render(separator)
				width += len(separator)
			}
		}
		segments[gi] = groupSegment{text: text, width: width}
	}

	// Calculate total single-line width.
	vertSepWidth := len(verticalSeparator)
	totalWidth := 0
	for i, seg := range segments {
		totalWidth += seg.width
		if i < len(segments)-1 {
			totalWidth += vertSepWidth
		}
	}

	// recordPositions appends optionPositions for a single rendered row,
	// given the indices of the segments on that row and the row's total width.
	recordPositions := func(segIndices []int, rowWidth int) {
		leftPad := (m.width - rowWidth) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		cursor := leftPad
		for si, segIdx := range segIndices {
			g := effectiveGroups[segIdx]
			for i := g.start; i < g.end; i++ {
				k := m.options[i]
				_, itemWidth := m.renderItem(i, k)
				startPos := cursor
				cursor += itemWidth
				m.optionPositions = append(m.optionPositions, optionPosition{
					startX: startPos,
					endX:   cursor,
					key:    k,
				})
				if i < g.end-1 {
					cursor += len(separator)
				}
			}
			if si < len(segIndices)-1 {
				cursor += vertSepWidth
			}
		}
	}

	// Single-line case: fits or only one group.
	if totalWidth <= m.width || len(segments) <= 1 {
		var s strings.Builder
		for gi, g := range effectiveGroups {
			for i := g.start; i < g.end; i++ {
				k := m.options[i]
				itemText, _ := m.renderItem(i, k)
				s.WriteString(itemText)
				if i < g.end-1 {
					s.WriteString(sepStyle.Render(separator))
				}
			}
			if gi < len(segments)-1 {
				s.WriteString(sepStyle.Render(verticalSeparator))
			}
		}

		allSegs := make([]int, len(segments))
		for i := range segments {
			allSegs[i] = i
		}
		recordPositions(allSegs, totalWidth)

		centeredMenuText := menuStyle.Render(s.String())
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, centeredMenuText)
	}

	// Multi-line case: greedily fill rows at group boundaries.
	type row struct {
		text     string
		width    int
		segments []int
	}
	var rows []row
	var currentText string
	currentWidth := 0
	var currentSegs []int
	for i, seg := range segments {
		needWidth := seg.width
		if currentWidth > 0 {
			needWidth += vertSepWidth
		}
		if currentWidth > 0 && currentWidth+needWidth > m.width {
			rows = append(rows, row{text: currentText, width: currentWidth, segments: currentSegs})
			currentText = ""
			currentWidth = 0
			currentSegs = nil
		}
		if currentWidth > 0 {
			currentText += sepStyle.Render(verticalSeparator)
			currentWidth += vertSepWidth
		}
		currentText += segments[i].text
		currentWidth += seg.width
		currentSegs = append(currentSegs, i)
	}
	if currentWidth > 0 {
		rows = append(rows, row{text: currentText, width: currentWidth, segments: currentSegs})
	}

	renderedRows := make([]string, len(rows))
	for i, r := range rows {
		recordPositions(r.segments, r.width)
		renderedRows[i] = lipgloss.PlaceHorizontal(m.width, lipgloss.Center, menuStyle.Render(r.text))
	}

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Left, renderedRows...))
}

// OptionAtX returns the key name for the menu option at the given X
// coordinate (local to the menu panel). Positions are stored absolutely
// (already centered within m.width), so this is a direct lookup. When the
// menu wraps across multiple rows, X ranges from different rows can overlap;
// the first registered position wins.
func (m *Menu) OptionAtX(localX int) (keys.KeyName, bool) {
	for _, op := range m.optionPositions {
		if localX >= op.startX && localX < op.endX {
			return op.key, true
		}
	}
	return 0, false
}

// HasOption reports whether a given key was rendered in the most recent
// String() call. Useful for tests that want to verify a key is present
// regardless of which row it landed on.
func (m *Menu) HasOption(k keys.KeyName) bool {
	for _, op := range m.optionPositions {
		if op.key == k {
			return true
		}
	}
	return false
}
