package app

import (
	"claude-squad/config"
	"claude-squad/keys"
	"claude-squad/log"
	"claude-squad/session"
	"claude-squad/session/git"
	"claude-squad/sound"
	"claude-squad/ui"
	"claude-squad/ui/overlay"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

const GlobalInstanceLimit = 10

// Run is the main entrypoint into the application.
func Run(ctx context.Context, program string, autoYes bool, projectDir string) error {
	p := tea.NewProgram(
		newHome(ctx, program, autoYes, projectDir),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(), // Mouse scroll
	)
	_, err := p.Run()
	return err
}

type state int

const (
	stateDefault state = iota
	// stateNew is the state when the user is creating a new instance.
	stateNew
	// statePrompt is the state when the user is entering a prompt.
	statePrompt
	// stateHelp is the state when a help screen is displayed.
	stateHelp
	// stateConfirm is the state when a confirmation modal is displayed.
	stateConfirm
	// stateInteractive is the state when user is typing into the tmux session preview.
	stateInteractive
	// stateNotes is the state when user is editing a session note.
	stateNotes
)

// layoutBounds tracks the screen rectangles of each panel for mouse hit testing.
type layoutBounds struct {
	list    ui.Rect
	preview ui.Rect
	kanban  ui.Rect
	menu    ui.Rect
}

type home struct {
	ctx context.Context

	// -- Storage and Configuration --

	program    string
	autoYes    bool
	projectDir string

	// storage is the interface for saving/loading data to/from the app's state
	storage *session.Storage
	// appConfig stores persistent application configuration
	appConfig *config.Config
	// appState stores persistent application state like seen help screens
	appState config.AppState

	// -- State --

	// state is the current discrete state of the application
	state state
	// newInstanceFinalizer is called when the state is stateNew and then you press enter.
	// It registers the new instance in the list after the instance has been started.
	newInstanceFinalizer func()

	// promptAfterName tracks if we should enter prompt mode after naming
	promptAfterName bool
	// pendingExecAttach stores a tea.ExecCommand when a help screen must be shown before attaching
	pendingExecAttach tea.ExecCommand
	// pendingConfirmAction stores the action to execute after confirmation dialog is accepted
	pendingConfirmAction tea.Cmd

	// keySent is used to manage underlining menu items
	keySent bool

	// -- UI Components --

	// list displays the list of instances
	list *ui.List
	// menu displays the bottom menu
	menu *ui.Menu
	// tabbedWindow displays the tabbed window with preview and diff panes
	tabbedWindow *ui.TabbedWindow
	// kanban displays the kanban board panel
	kanban *ui.KanbanBoard
	// kanbanVisible tracks whether the kanban panel is shown
	kanbanVisible bool
	// bounds stores the layout rectangles for mouse hit testing
	bounds layoutBounds
	// errBox displays error messages
	errBox *ui.ErrBox
	// global spinner instance. we plumb this down to where it's needed
	spinner spinner.Model
	// textInputOverlay handles text input with state
	textInputOverlay *overlay.TextInputOverlay
	// textOverlay displays text information
	textOverlay *overlay.TextOverlay
	// helpOverlay displays the general help screen with editable configs
	helpOverlay *overlay.HelpOverlay
	// confirmationOverlay displays confirmation modals
	confirmationOverlay *overlay.ConfirmationOverlay
	// projectConfig stores per-project configuration (e.g. default effort level)
	projectConfig *config.ProjectConfig
}

func newHome(ctx context.Context, program string, autoYes bool, projectDir string) *home {
	// Load application config
	appConfig := config.LoadConfig()

	// Load application state
	appState := config.LoadState()

	// Initialize storage
	storage, err := session.NewStorage(appState)
	if err != nil {
		fmt.Printf("Failed to initialize storage: %v\n", err)
		os.Exit(1)
	}

	// Resolve git repo root for project filtering
	gitRepoRoot, err := git.FindGitRepoRoot(projectDir)
	if err != nil {
		fmt.Printf("Failed to find git repo root: %v\n", err)
		os.Exit(1)
	}

	// Load per-project configuration
	projectConfig := config.LoadProjectConfig(gitRepoRoot)

	h := &home{
		ctx:           ctx,
		spinner:       spinner.New(spinner.WithSpinner(spinner.MiniDot)),
		menu:          ui.NewMenu(),
		tabbedWindow:  ui.NewTabbedWindow(ui.NewPreviewPane(), ui.NewDiffPane(), ui.NewTerminalPane()),
		errBox:        ui.NewErrBox(),
		storage:       storage,
		appConfig:     appConfig,
		program:       program,
		autoYes:       autoYes,
		projectDir:    gitRepoRoot,
		state:         stateDefault,
		appState:      appState,
		projectConfig: projectConfig,
	}
	h.kanban = ui.NewKanbanBoard(&h.spinner)
	h.list = ui.NewList(&h.spinner, autoYes)

	// Load saved instances filtered by current project
	instances, err := storage.LoadInstancesForProject(gitRepoRoot)
	if err != nil {
		fmt.Printf("Failed to load instances: %v\n", err)
		os.Exit(1)
	}

	// Add loaded instances to the list
	for _, instance := range instances {
		// Call the finalizer immediately.
		h.list.AddInstance(instance)()
		if autoYes {
			instance.AutoYes = true
		}
	}

	// Discover Claude Code worktrees
	discovered, err := session.DiscoverClaudeWorktrees(projectDir)
	if err == nil {
		for _, ds := range discovered {
			// Skip if an instance with the same title already exists
			if h.hasInstanceWithTitle(ds.WorktreeName) {
				continue
			}
			extInstance, err := session.NewExternalInstance(ds)
			if err != nil {
				log.ErrorLog.Printf("Failed to create external instance %s: %v", ds.WorktreeName, err)
				continue
			}
			h.list.AddInstance(extInstance)()
		}
	} else {
		log.ErrorLog.Printf("Failed to discover Claude Code worktrees: %v", err)
	}

	return h
}

// hasInstanceWithTitle checks if any instance in the list has the given title.
func (m *home) hasInstanceWithTitle(title string) bool {
	for _, inst := range m.list.GetInstances() {
		if inst.Title == title {
			return true
		}
	}
	return false
}

// updateHandleWindowSizeEvent sets the sizes of the components.
// The components will try to render inside their bounds.
func (m *home) updateHandleWindowSizeEvent(msg tea.WindowSizeMsg) {
	// Auto-hide kanban when terminal is too narrow
	if msg.Width < 80 {
		if m.kanbanVisible {
			// Sync list selection from kanban cursor before hiding
			if inst := m.kanban.GetCursorInstance(); inst != nil {
				m.list.SelectInstance(inst)
			}
			m.kanbanVisible = false
			m.menu.SetKanbanVisible(false)
		}
	}

	// Menu takes 10% of height, list and window take 90%
	contentHeight := int(float32(msg.Height) * 0.9)
	menuHeight := msg.Height - contentHeight - 1     // minus 1 for error box
	m.errBox.SetSize(int(float32(msg.Width)*0.9), 1) // error box takes 1 row

	var listWidth, tabsWidth, kanbanWidth int

	if m.kanbanVisible {
		// 2-panel layout: kanban replaces list, kanban + preview
		listWidth = 0
		kanbanWidth = int(float32(msg.Width) * 0.4)
		tabsWidth = msg.Width - kanbanWidth
	} else {
		// 2-panel layout: 30% list, 70% preview
		listWidth = int(float32(msg.Width) * 0.3)
		tabsWidth = msg.Width - listWidth
		kanbanWidth = 0
	}

	m.tabbedWindow.SetSize(tabsWidth, contentHeight)
	m.list.SetSize(listWidth, contentHeight)
	m.kanban.SetSize(kanbanWidth, contentHeight)

	// Populate layout bounds for mouse hit testing.
	if m.kanbanVisible {
		// Kanban at left, preview at right
		m.bounds.list = ui.Rect{}
		m.bounds.kanban = ui.Rect{X: 0, Y: 0, Width: kanbanWidth, Height: contentHeight}
		m.bounds.preview = ui.Rect{X: kanbanWidth, Y: 0, Width: tabsWidth, Height: contentHeight}
	} else {
		// List at left, preview at right
		m.bounds.list = ui.Rect{X: 0, Y: 0, Width: listWidth, Height: contentHeight}
		m.bounds.preview = ui.Rect{X: listWidth, Y: 0, Width: tabsWidth, Height: contentHeight}
		m.bounds.kanban = ui.Rect{}
	}
	m.bounds.menu = ui.Rect{X: 0, Y: contentHeight, Width: msg.Width, Height: menuHeight + 1}

	if m.textInputOverlay != nil {
		m.textInputOverlay.SetSize(int(float32(msg.Width)*0.6), int(float32(msg.Height)*0.4))
	}
	if m.textOverlay != nil {
		m.textOverlay.SetWidth(int(float32(msg.Width) * 0.6))
	}
	if m.helpOverlay != nil {
		m.helpOverlay.SetWidth(int(float32(msg.Width) * 0.6))
	}

	previewWidth, previewHeight := m.tabbedWindow.GetPreviewSize()
	if err := m.list.SetSessionPreviewSize(previewWidth, previewHeight); err != nil {
		log.ErrorLog.Print(err)
	}
	m.menu.SetSize(msg.Width, menuHeight)
}

func (m *home) Init() tea.Cmd {
	// Upon starting, we want to start the spinner. Whenever we get a spinner.TickMsg, we
	// update the spinner, which sends a new spinner.TickMsg. I think this lasts forever lol.
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			time.Sleep(100 * time.Millisecond)
			return previewTickMsg{}
		},
		tickUpdateMetadataCmd,
	)
}

func (m *home) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case hideErrMsg:
		m.errBox.Clear()
	case previewTickMsg:
		cmd := m.instanceChanged()
		return m, tea.Batch(
			cmd,
			func() tea.Msg {
				time.Sleep(100 * time.Millisecond)
				return previewTickMsg{}
			},
		)
	case keyupMsg:
		m.menu.ClearKeydown()
		return m, nil
	case tickUpdateMetadataMessage:
		shouldPlaySound := false
		for _, instance := range m.list.GetInstances() {
			if !instance.Started() || instance.Paused() {
				continue
			}
			instance.CheckAndHandleTrustPrompt()
			updated, prompt := instance.HasUpdated()
			if updated {
				instance.SetStatus(session.Running)
			} else {
				if prompt {
					instance.TapEnter()
				} else {
					if instance.Status == session.Running {
						shouldPlaySound = true
					}
					instance.SetStatus(session.Ready)
				}
			}
			if err := instance.UpdateDiffStats(); err != nil {
				log.WarningLog.Printf("could not update diff stats: %v", err)
			}
		}
		if shouldPlaySound && m.projectConfig.GetSoundAlert() {
			sound.Play(m.projectConfig.GetAlertSound())
		}
		return m, tickUpdateMetadataCmd
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress {
			x, y := msg.X, msg.Y

			switch msg.Button {
			case tea.MouseButtonLeft:
				return m.handleMouseClick(x, y)
			case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
				return m.handleMouseWheel(msg.Button, x, y)
			}
		}
		return m, nil
	case branchSearchDebounceMsg:
		// Debounce timer fired — check if this is still the current filter version
		if m.textInputOverlay == nil {
			return m, nil
		}
		if msg.version != m.textInputOverlay.BranchFilterVersion() {
			return m, nil // stale, a newer debounce is pending
		}
		return m, m.runBranchSearch(msg.filter, msg.version)
	case branchSearchResultMsg:
		if m.textInputOverlay != nil {
			m.textInputOverlay.SetBranchResults(msg.branches, msg.version)
		}
		return m, nil
	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	case tea.WindowSizeMsg:
		m.updateHandleWindowSizeEvent(msg)
		return m, nil
	case error:
		// Handle errors from confirmation actions
		return m, m.handleError(msg)
	case instanceChangedMsg:
		// Handle instance changed after confirmation action
		return m, m.instanceChanged()
	case instanceStartedMsg:
		// Select the instance that just started (or failed)
		m.list.SelectInstance(msg.instance)

		if msg.err != nil {
			m.list.Kill()
			return m, tea.Batch(m.handleError(msg.err), m.instanceChanged())
		}

		// Save after successful start
		if err := m.storage.SaveInstancesForProject(m.projectDir, m.list.GetInstances()); err != nil {
			return m, m.handleError(err)
		}
		if m.autoYes {
			msg.instance.AutoYes = true
		}

		if msg.promptAfterName {
			m.state = statePrompt
			m.menu.SetState(ui.StatePrompt)
			m.textInputOverlay = m.newPromptOverlay()
		} else {
			// If instance has a prompt (set from Shift+N flow), send it now
			if msg.instance.Prompt != "" {
				if err := msg.instance.SendPrompt(msg.instance.Prompt); err != nil {
					log.ErrorLog.Printf("failed to send prompt: %v", err)
				}
				msg.instance.Prompt = ""
			}
			m.menu.SetState(ui.StateDefault)
			m.showHelpScreen(helpStart(msg.instance), nil)
		}

		return m, tea.Batch(tea.WindowSize(), m.instanceChanged())
	case attachFinishedMsg:
		m.state = stateDefault
		m.menu.SetState(ui.StateDefault)
		if msg.err != nil {
			return m, tea.Batch(m.handleError(msg.err), tea.WindowSize())
		}
		return m, tea.WindowSize()
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *home) handleQuit() (tea.Model, tea.Cmd) {
	if err := m.storage.SaveInstancesForProject(m.projectDir, m.list.GetInstances()); err != nil {
		return m, m.handleError(err)
	}
	return m, tea.Quit
}

// handleInteractiveState handles key events in interactive mode, forwarding them to the tmux session.
func (m *home) handleInteractiveState(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Escape exits interactive mode
	if msg.Type == tea.KeyEsc || msg.Type == tea.KeyCtrlC {
		m.state = stateDefault
		m.menu.SetState(ui.StateDefault)
		m.tabbedWindow.SetPreviewInteractive(false)
		return m, nil
	}

	selected := m.getActiveInstance()
	if selected == nil || !selected.TmuxAlive() {
		// Instance died, exit interactive mode
		m.state = stateDefault
		m.menu.SetState(ui.StateDefault)
		m.tabbedWindow.SetPreviewInteractive(false)
		return m, nil
	}

	translated := translateKeyMsg(msg)
	if translated != "" {
		if err := selected.SendKeys(translated); err != nil {
			log.ErrorLog.Printf("interactive send keys failed: %v", err)
		}
	}
	return m, nil
}

// translateKeyMsg converts a bubbletea KeyMsg into the byte string to send to a tmux PTY.
func translateKeyMsg(msg tea.KeyMsg) string {
	switch msg.Type {
	case tea.KeyRunes:
		return string(msg.Runes)
	case tea.KeyEnter:
		return "\r"
	case tea.KeyBackspace:
		return "\x7f"
	case tea.KeyTab:
		return "\t"
	case tea.KeySpace:
		return " "
	case tea.KeyUp:
		return "\x1b[A"
	case tea.KeyDown:
		return "\x1b[B"
	case tea.KeyRight:
		return "\x1b[C"
	case tea.KeyLeft:
		return "\x1b[D"
	case tea.KeyCtrlA:
		return "\x01"
	case tea.KeyCtrlB:
		return "\x02"
	case tea.KeyCtrlD:
		return "\x04"
	case tea.KeyCtrlE:
		return "\x05"
	case tea.KeyCtrlF:
		return "\x06"
	case tea.KeyCtrlG:
		return "\x07"
	case tea.KeyCtrlH:
		return "\x08"
	case tea.KeyCtrlJ:
		return "\x0a"
	case tea.KeyCtrlK:
		return "\x0b"
	case tea.KeyCtrlL:
		return "\x0c"
	case tea.KeyCtrlN:
		return "\x0e"
	case tea.KeyCtrlO:
		return "\x0f"
	case tea.KeyCtrlP:
		return "\x10"
	case tea.KeyCtrlQ:
		return "\x11"
	case tea.KeyCtrlR:
		return "\x12"
	case tea.KeyCtrlS:
		return "\x13"
	case tea.KeyCtrlT:
		return "\x14"
	case tea.KeyCtrlU:
		return "\x15"
	case tea.KeyCtrlV:
		return "\x16"
	case tea.KeyCtrlW:
		return "\x17"
	case tea.KeyCtrlX:
		return "\x18"
	case tea.KeyCtrlY:
		return "\x19"
	case tea.KeyCtrlZ:
		return "\x1a"
	case tea.KeyDelete:
		return "\x1b[3~"
	case tea.KeyHome:
		return "\x1b[H"
	case tea.KeyEnd:
		return "\x1b[F"
	default:
		// For any unhandled key type, try the string representation
		s := msg.String()
		if len(s) == 1 {
			return s
		}
		return ""
	}
}

func (m *home) handleMenuHighlighting(msg tea.KeyMsg) (cmd tea.Cmd, returnEarly bool) {
	// Handle menu highlighting when you press a button. We intercept it here and immediately return to
	// update the ui while re-sending the keypress. Then, on the next call to this, we actually handle the keypress.
	if m.keySent {
		m.keySent = false
		return nil, false
	}
	if m.state == statePrompt || m.state == stateHelp || m.state == stateConfirm || m.state == stateInteractive || m.state == stateNotes {
		return nil, false
	}
	// If it's in the global keymap, we should try to highlight it.
	name, ok := keys.GlobalKeyStringsMap[msg.String()]
	if !ok {
		return nil, false
	}

	if active := m.getActiveInstance(); active != nil && active.Paused() && name == keys.KeyEnter {
		return nil, false
	}
	if name == keys.KeyShiftDown || name == keys.KeyShiftUp {
		return nil, false
	}

	// Skip the menu highlighting if the key is not in the map or we are using the shift up and down keys.
	// TODO: cleanup: when you press enter on stateNew, we use keys.KeySubmitName. We should unify the keymap.
	if name == keys.KeyEnter && m.state == stateNew {
		name = keys.KeySubmitName
	}
	m.keySent = true
	return tea.Batch(
		func() tea.Msg { return msg },
		m.keydownCallback(name)), true
}

func (m *home) handleKeyPress(msg tea.KeyMsg) (mod tea.Model, cmd tea.Cmd) {
	cmd, returnEarly := m.handleMenuHighlighting(msg)
	if returnEarly {
		return m, cmd
	}

	if m.state == stateHelp {
		return m.handleHelpState(msg)
	}

	if m.state == stateInteractive {
		return m.handleInteractiveState(msg)
	}

	if m.state == stateNew {
		// Handle quit commands first. Don't handle q because the user might want to type that.
		if msg.String() == "ctrl+c" {
			m.state = stateDefault
			m.promptAfterName = false
			m.list.Kill()
			return m, tea.Sequence(
				tea.WindowSize(),
				func() tea.Msg {
					m.menu.SetState(ui.StateDefault)
					return nil
				},
			)
		}

		instance := m.list.GetInstances()[m.list.NumInstances()-1]
		switch msg.Type {
		// Start the instance (enable previews etc) and go back to the main menu state.
		case tea.KeyEnter:
			if len(instance.Title) == 0 {
				return m, m.handleError(fmt.Errorf("title cannot be empty"))
			}

			// If promptAfterName, show prompt+branch overlay before starting
			if m.promptAfterName {
				m.promptAfterName = false
				m.state = statePrompt
				m.menu.SetState(ui.StatePrompt)
				m.textInputOverlay = m.newPromptOverlay()
				// Trigger initial branch search (no debounce, version 0)
				initialSearch := m.runBranchSearch("", m.textInputOverlay.BranchFilterVersion())
				return m, tea.Batch(tea.WindowSize(), initialSearch)
			}

			// Set Loading status and finalize into the list immediately
			instance.Effort = m.projectConfig.DefaultEffort
			instance.Model = m.projectConfig.DefaultModel
			instance.SkipPermissions = m.projectConfig.GetSkipPermissions()
			instance.SetStatus(session.Loading)
			m.newInstanceFinalizer()
			m.promptAfterName = false
			m.state = stateDefault
			m.menu.SetState(ui.StateDefault)

			// Return a tea.Cmd that runs instance.Start in the background
			startCmd := func() tea.Msg {
				err := instance.Start(true)
				return instanceStartedMsg{
					instance:        instance,
					err:             err,
					promptAfterName: false,
				}
			}

			return m, tea.Batch(tea.WindowSize(), m.instanceChanged(), startCmd)
		case tea.KeyRunes:
			if runewidth.StringWidth(instance.Title) >= 32 {
				return m, m.handleError(fmt.Errorf("title cannot be longer than 32 characters"))
			}
			if err := instance.SetTitle(instance.Title + string(msg.Runes)); err != nil {
				return m, m.handleError(err)
			}
		case tea.KeyBackspace:
			runes := []rune(instance.Title)
			if len(runes) == 0 {
				return m, nil
			}
			if err := instance.SetTitle(string(runes[:len(runes)-1])); err != nil {
				return m, m.handleError(err)
			}
		case tea.KeySpace:
			if err := instance.SetTitle(instance.Title + " "); err != nil {
				return m, m.handleError(err)
			}
		case tea.KeyEsc:
			m.list.Kill()
			m.state = stateDefault
			m.instanceChanged()

			return m, tea.Sequence(
				tea.WindowSize(),
				func() tea.Msg {
					m.menu.SetState(ui.StateDefault)
					return nil
				},
			)
		default:
		}
		return m, nil
	} else if m.state == statePrompt {
		// Handle cancel via ctrl+c before delegating to the overlay
		if msg.String() == "ctrl+c" {
			return m, m.cancelPromptOverlay()
		}

		// Use the new TextInputOverlay component to handle all key events
		shouldClose, branchFilterChanged := m.textInputOverlay.HandleKeyPress(msg)

		// Check if the form was submitted or canceled
		if shouldClose {
			selected := m.list.GetSelectedInstance()
			if selected == nil {
				return m, nil
			}

			if m.textInputOverlay.IsCanceled() {
				return m, m.cancelPromptOverlay()
			}

			if m.textInputOverlay.IsSubmitted() {
				prompt := m.textInputOverlay.GetValue()
				selectedBranch := m.textInputOverlay.GetSelectedBranch()
				selectedProgram := m.textInputOverlay.GetSelectedProgram()
				selectedEffort := m.textInputOverlay.GetSelectedEffort()

				selectedModel := m.textInputOverlay.GetSelectedModel()
				selectedSkipPerms := m.textInputOverlay.GetSkipPermissions()

				if !selected.Started() {
					// Shift+N flow: instance not started yet — set branch, start, then send prompt
					if selectedBranch != "" {
						selected.SetSelectedBranch(selectedBranch)
					}
					if selectedProgram != "" {
						selected.Program = selectedProgram
					}
					if selectedEffort != "" {
						selected.Effort = selectedEffort
					}
					selected.Model = selectedModel
					selected.SkipPermissions = selectedSkipPerms
					selected.Prompt = prompt

					// Finalize into list and start
					selected.SetStatus(session.Loading)
					m.newInstanceFinalizer()
					m.textInputOverlay = nil
					m.state = stateDefault
					m.menu.SetState(ui.StateDefault)

					startCmd := func() tea.Msg {
						err := selected.Start(true)
						return instanceStartedMsg{
							instance:        selected,
							err:             err,
							promptAfterName: false,
							selectedBranch:  selectedBranch,
						}
					}

					return m, tea.Batch(tea.WindowSize(), m.instanceChanged(), startCmd)
				}

				// Regular flow: instance already running, just send prompt
				if err := selected.SendPrompt(prompt); err != nil {
					return m, m.handleError(err)
				}
			}

			// Close the overlay and reset state
			m.textInputOverlay = nil
			m.state = stateDefault
			return m, tea.Sequence(
				tea.WindowSize(),
				func() tea.Msg {
					m.menu.SetState(ui.StateDefault)
					m.showHelpScreen(helpStart(selected), nil)
					return nil
				},
			)
		}

		// Schedule a debounced branch search if the filter changed
		if branchFilterChanged {
			filter := m.textInputOverlay.BranchFilter()
			version := m.textInputOverlay.BranchFilterVersion()
			return m, m.scheduleBranchSearch(filter, version)
		}

		return m, nil
	}

	// Handle notes state
	if m.state == stateNotes {
		if msg.String() == "ctrl+c" {
			m.textInputOverlay = nil
			m.state = stateDefault
			m.menu.SetState(ui.StateDefault)
			return m, tea.WindowSize()
		}

		shouldClose, _ := m.textInputOverlay.HandleKeyPress(msg)
		if shouldClose {
			selected := m.getActiveInstance()
			if m.textInputOverlay.IsSubmitted() && selected != nil {
				content := m.textInputOverlay.GetValue()
				if err := session.SaveNote(m.projectDir, selected.Title, content); err != nil {
					log.ErrorLog.Printf("failed to save note: %v", err)
				}
			}
			m.textInputOverlay = nil
			m.state = stateDefault
			m.menu.SetState(ui.StateDefault)
			return m, tea.WindowSize()
		}
		return m, nil
	}

	// Handle confirmation state
	if m.state == stateConfirm {
		shouldClose := m.confirmationOverlay.HandleKeyPress(msg)
		if shouldClose {
			m.state = stateDefault
			m.confirmationOverlay = nil
			cmd := m.pendingConfirmAction
			m.pendingConfirmAction = nil
			return m, cmd
		}
		return m, nil
	}

	// Exit scrolling mode when ESC is pressed and preview pane is in scrolling mode
	// Check if Escape key was pressed and we're not in the diff tab (meaning we're in preview tab)
	// Always check for escape key first to ensure it doesn't get intercepted elsewhere
	if msg.Type == tea.KeyEsc {
		// If in preview tab and in scroll mode, exit scroll mode
		if m.tabbedWindow.IsInPreviewTab() && m.tabbedWindow.IsPreviewInScrollMode() {
			selected := m.getActiveInstance()
			err := m.tabbedWindow.ResetPreviewToNormalMode(selected)
			if err != nil {
				return m, m.handleError(err)
			}
			return m, m.instanceChanged()
		}
		// If in terminal tab and in scroll mode, exit scroll mode
		if m.tabbedWindow.IsInTerminalTab() && m.tabbedWindow.IsTerminalInScrollMode() {
			m.tabbedWindow.ResetTerminalToNormalMode()
			return m, m.instanceChanged()
		}
	}

	// Handle quit commands first
	if msg.String() == "ctrl+c" || msg.String() == "q" {
		return m.handleQuit()
	}

	name, ok := keys.GlobalKeyStringsMap[msg.String()]
	if !ok {
		return m, nil
	}

	switch name {
	case keys.KeyHelp:
		return m.showHelpScreen(helpTypeGeneral{}, nil)
	case keys.KeyPrompt:
		if m.list.NumInstances() >= GlobalInstanceLimit {
			return m, m.handleError(
				fmt.Errorf("you can't create more than %d instances", GlobalInstanceLimit))
		}

		// Start a background fetch so branches are up to date by the time the picker opens
		fetchCmd := func() tea.Msg {
			currentDir, _ := os.Getwd()
			git.FetchBranches(currentDir)
			return nil
		}

		instance, err := session.NewInstance(session.InstanceOptions{
			Title:   "",
			Path:    ".",
			Program: m.program,
		})
		if err != nil {
			return m, m.handleError(err)
		}

		m.newInstanceFinalizer = m.list.AddInstance(instance)
		m.list.SetSelectedInstance(m.list.NumInstances() - 1)
		m.state = stateNew
		m.menu.SetState(ui.StateNewInstance)
		m.promptAfterName = true

		return m, fetchCmd
	case keys.KeyNew:
		if m.list.NumInstances() >= GlobalInstanceLimit {
			return m, m.handleError(
				fmt.Errorf("you can't create more than %d instances", GlobalInstanceLimit))
		}
		instance, err := session.NewInstance(session.InstanceOptions{
			Title:   "",
			Path:    ".",
			Program: m.program,
		})
		if err != nil {
			return m, m.handleError(err)
		}

		m.newInstanceFinalizer = m.list.AddInstance(instance)
		m.list.SetSelectedInstance(m.list.NumInstances() - 1)
		m.state = stateNew
		m.menu.SetState(ui.StateNewInstance)

		return m, nil
	case keys.KeyUp:
		if m.kanbanVisible {
			m.kanban.CursorUp()
		} else {
			m.list.Up()
		}
		return m, m.instanceChanged()
	case keys.KeyDown:
		if m.kanbanVisible {
			m.kanban.CursorDown()
		} else {
			m.list.Down()
		}
		return m, m.instanceChanged()
	case keys.KeyLeft:
		if m.kanbanVisible {
			m.kanban.CursorLeft()
			return m, m.instanceChanged()
		}
		return m, nil
	case keys.KeyRight:
		if m.kanbanVisible {
			m.kanban.CursorRight()
			return m, m.instanceChanged()
		}
		return m, nil
	case keys.KeyShiftUp:
		m.tabbedWindow.ScrollUp()
		return m, m.instanceChanged()
	case keys.KeyShiftDown:
		m.tabbedWindow.ScrollDown()
		return m, m.instanceChanged()
	case keys.KeyTab:
		m.tabbedWindow.Toggle()
		m.menu.SetActiveTab(m.tabbedWindow.GetActiveTab())
		return m, m.instanceChanged()
	case keys.KeyKill:
		selected := m.getActiveInstance()
		if selected == nil || selected.Status == session.Loading {
			return m, nil
		}
		// Create the kill action as a tea.Cmd
		killAction := func() tea.Msg {
			// Get worktree and check if branch is checked out
			worktree, err := selected.GetGitWorktree()
			if err != nil {
				return err
			}

			checkedOut, err := worktree.IsBranchCheckedOut()
			if err != nil {
				return err
			}

			if checkedOut {
				return fmt.Errorf("instance %s is currently checked out", selected.Title)
			}

			// Clean up terminal session for this instance
			m.tabbedWindow.CleanupTerminalForInstance(selected.Title)

			// Delete from storage (skip for external sessions - they aren't persisted)
			if !selected.IsExternal() {
				if err := m.storage.DeleteInstance(selected.Title); err != nil {
					return err
				}
			}

			// Clean up note file
			_ = session.DeleteNote(m.projectDir, selected.Title)

			// Then kill the instance
			m.list.Kill()
			return instanceChangedMsg{}
		}

		// Show confirmation modal
		message := fmt.Sprintf("[!] Kill session '%s'?", selected.Title)
		return m, m.confirmAction(message, killAction)
	case keys.KeySubmit:
		selected := m.getActiveInstance()
		if selected == nil || selected.Status == session.Loading {
			return m, nil
		}
		if selected.IsExternal() {
			return m, m.handleError(fmt.Errorf("cannot push from external session '%s' (managed by Claude Code)", selected.Title))
		}

		// Create the push action as a tea.Cmd
		pushAction := func() tea.Msg {
			// Default commit message with timestamp
			commitMsg := fmt.Sprintf("[claudesquad] update from '%s' on %s", selected.Title, time.Now().Format(time.RFC822))
			worktree, err := selected.GetGitWorktree()
			if err != nil {
				return err
			}
			if err = worktree.PushChanges(commitMsg, true); err != nil {
				return err
			}
			return nil
		}

		// Show confirmation modal
		message := fmt.Sprintf("[!] Push changes from session '%s'?", selected.Title)
		return m, m.confirmAction(message, pushAction)
	case keys.KeyCheckout:
		selected := m.getActiveInstance()
		if selected == nil || selected.Status == session.Loading {
			return m, nil
		}

		if selected.IsExternal() {
			// Show confirmation before taking ownership of external session
			message := fmt.Sprintf("[!] Checkout external session '%s'? This will stop the running Claude Code process.", selected.Title)
			checkoutAction := func() tea.Msg {
				selected.SetManaged()
				if err := selected.Pause(); err != nil {
					return err
				}
				m.tabbedWindow.CleanupTerminalForInstance(selected.Title)
				m.instanceChanged()
				return nil
			}
			return m, m.confirmAction(message, checkoutAction)
		}

		// Show help screen before pausing
		m.showHelpScreen(helpTypeInstanceCheckout{}, func() {
			if err := selected.Pause(); err != nil {
				m.handleError(err)
			}
			m.tabbedWindow.CleanupTerminalForInstance(selected.Title)
			m.instanceChanged()
		})
		return m, nil
	case keys.KeyResume:
		selected := m.getActiveInstance()
		if selected == nil || selected.Status == session.Loading {
			return m, nil
		}
		if selected.Paused() {
			if err := selected.Resume(); err != nil {
				return m, m.handleError(err)
			}
		} else if !selected.TmuxAlive() {
			if err := selected.Revive(); err != nil {
				return m, m.handleError(err)
			}
		}
		return m, tea.WindowSize()
	case keys.KeyYank:
		if !m.kanbanVisible {
			return m, nil
		}
		selected := m.getActiveInstance()
		if selected == nil {
			return m, nil
		}
		tmuxName := selected.GetTmuxSessionName()
		if tmuxName == "" {
			return m, m.handleError(fmt.Errorf("no tmux session name for '%s'", selected.Title))
		}
		if err := copyToClipboard(tmuxName); err != nil {
			return m, m.handleError(fmt.Errorf("failed to copy to clipboard: %w", err))
		}
		return m, m.handleError(fmt.Errorf("Copied tmux session name: %s", tmuxName))
	case keys.KeyEnter:
		if m.list.NumInstances() == 0 {
			return m, nil
		}
		selected := m.getActiveInstance()
		if selected == nil || selected.Paused() || selected.Status == session.Loading || !selected.TmuxAlive() {
			return m, nil
		}

		// Build the exec command
		var execCmd tea.ExecCommand
		var err error
		if m.tabbedWindow.IsInTerminalTab() {
			execCmd, err = m.tabbedWindow.ExecAttachTerminal()
		} else {
			execCmd, err = m.list.ExecAttach()
		}
		if err != nil {
			return m, m.handleError(err)
		}

		// Check if help screen needs to be shown
		attachCmd := tea.Exec(execCmd, func(err error) tea.Msg {
			return attachFinishedMsg{err: err}
		})

		flag := helpTypeInstanceAttach{}.mask()
		if (m.appState.GetHelpScreensSeen() & flag) == 0 {
			// Store command, show help first
			m.pendingExecAttach = execCmd
			m.showHelpScreen(helpTypeInstanceAttach{}, nil)
			return m, nil
		}
		// Help already seen, attach directly
		return m, attachCmd
	case keys.KeyInteractive:
		selected := m.getActiveInstance()
		if selected == nil || selected.Paused() || selected.Status == session.Loading || !selected.TmuxAlive() {
			return m, nil
		}
		if !m.tabbedWindow.IsInPreviewTab() {
			return m, nil
		}
		// Exit scroll mode if active
		if m.tabbedWindow.IsPreviewInScrollMode() {
			if err := m.tabbedWindow.ResetPreviewToNormalMode(selected); err != nil {
				return m, m.handleError(err)
			}
		}
		m.state = stateInteractive
		m.menu.SetState(ui.StateInteractive)
		m.tabbedWindow.SetPreviewInteractive(true)
		return m, nil
	case keys.KeyNotes:
		selected := m.getActiveInstance()
		if selected == nil || selected.Status == session.Loading {
			return m, nil
		}
		existingContent, err := session.LoadNote(m.projectDir, selected.Title)
		if err != nil {
			return m, m.handleError(err)
		}
		m.textInputOverlay = overlay.NewTextInputOverlay("Edit Note", existingContent)
		m.state = stateNotes
		m.menu.SetState(ui.StateNotes)
		return m, tea.WindowSize()
	case keys.KeyKanban:
		m.kanbanVisible = !m.kanbanVisible
		m.menu.SetKanbanVisible(m.kanbanVisible)
		if m.kanbanVisible {
			// Initialize kanban cursor from current list selection
			if inst := m.list.GetSelectedInstance(); inst != nil {
				m.kanban.SetCursorToInstance(inst)
			}
		} else {
			// Sync list selection to kanban cursor
			if inst := m.kanban.GetCursorInstance(); inst != nil {
				m.list.SelectInstance(inst)
			}
		}
		return m, tea.Batch(m.instanceChanged(), tea.WindowSize())
	default:
		return m, nil
	}
}

// getActiveInstance returns the instance that should be acted upon.
// When kanban is visible, it returns the kanban cursor instance; otherwise, the list selection.
func (m *home) getActiveInstance() *session.Instance {
	if m.kanbanVisible {
		return m.kanban.GetCursorInstance()
	}
	return m.list.GetSelectedInstance()
}

// instanceChanged updates the preview pane, menu, and diff pane based on the selected instance. It returns an error
// Cmd if there was any error.
func (m *home) instanceChanged() tea.Cmd {
	// selected may be nil
	selected := m.getActiveInstance()

	// When kanban is visible, sync the list selection so downstream code works
	if m.kanbanVisible && selected != nil {
		m.list.SelectInstance(selected)
	}

	m.tabbedWindow.UpdateDiff(selected)
	m.tabbedWindow.SetInstance(selected)
	// Update menu with current instance
	m.menu.SetInstance(selected)
	// Update kanban board
	m.kanban.UpdateInstances(m.list.GetInstances(), selected)

	// If there's no selected instance, we don't need to update the preview.
	if err := m.tabbedWindow.UpdatePreview(selected); err != nil {
		return m.handleError(err)
	}
	if err := m.tabbedWindow.UpdateTerminal(selected); err != nil {
		return m.handleError(err)
	}
	return nil
}

type keyupMsg struct{}

// keydownCallback clears the menu option highlighting after 500ms.
func (m *home) keydownCallback(name keys.KeyName) tea.Cmd {
	m.menu.Keydown(name)
	return func() tea.Msg {
		select {
		case <-m.ctx.Done():
		case <-time.After(500 * time.Millisecond):
		}

		return keyupMsg{}
	}
}

// hideErrMsg implements tea.Msg and clears the error text from the screen.
type hideErrMsg struct{}

// previewTickMsg implements tea.Msg and triggers a preview update
type previewTickMsg struct{}

type tickUpdateMetadataMessage struct{}

type instanceChangedMsg struct{}

type attachFinishedMsg struct {
	err error
}

type instanceStartedMsg struct {
	instance        *session.Instance
	err             error
	promptAfterName bool
	selectedBranch  string
}

// branchSearchDebounceMsg fires after the debounce interval to trigger a search.
type branchSearchDebounceMsg struct {
	filter  string
	version uint64
}

// branchSearchResultMsg carries search results back to Update.
type branchSearchResultMsg struct {
	branches []string
	version  uint64
}

const branchSearchDebounce = 150 * time.Millisecond

// scheduleBranchSearch returns a debounced tea.Cmd: sleeps, then triggers a search message.
func (m *home) scheduleBranchSearch(filter string, version uint64) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(branchSearchDebounce)
		return branchSearchDebounceMsg{filter: filter, version: version}
	}
}

// runBranchSearch returns a tea.Cmd that performs the git search in the background.
func (m *home) runBranchSearch(filter string, version uint64) tea.Cmd {
	return func() tea.Msg {
		currentDir, _ := os.Getwd()
		branches, err := git.SearchBranches(currentDir, filter)
		if err != nil {
			log.WarningLog.Printf("branch search failed: %v", err)
			return nil
		}
		return branchSearchResultMsg{branches: branches, version: version}
	}
}

// tickUpdateMetadataCmd is the callback to update the metadata of the instances every 500ms. Note that we iterate
// overall the instances and capture their output. It's a pretty expensive operation. Let's do it 2x a second only.
var tickUpdateMetadataCmd = func() tea.Msg {
	time.Sleep(500 * time.Millisecond)
	return tickUpdateMetadataMessage{}
}

// handleError handles all errors which get bubbled up to the app. sets the error message. We return a callback tea.Cmd that returns a hideErrMsg message
// which clears the error message after 3 seconds.
func (m *home) handleError(err error) tea.Cmd {
	log.ErrorLog.Printf("%v", err)
	m.errBox.SetError(err)
	return func() tea.Msg {
		select {
		case <-m.ctx.Done():
		case <-time.After(3 * time.Second):
		}

		return hideErrMsg{}
	}
}

func (m *home) newPromptOverlay() *overlay.TextInputOverlay {
	return overlay.NewTextInputOverlayWithBranchPicker(
		"Enter prompt", "",
		m.appConfig.GetProfiles(),
		m.projectConfig.DefaultEffort,
		m.projectConfig.DefaultModel,
		m.projectConfig.GetSkipPermissions(),
	)
}

// cancelPromptOverlay cancels the prompt overlay, cleaning up unstarted instances.
func (m *home) cancelPromptOverlay() tea.Cmd {
	selected := m.list.GetSelectedInstance()
	if selected != nil && !selected.Started() {
		m.list.Kill()
	}
	m.textInputOverlay = nil
	m.state = stateDefault
	return tea.Sequence(
		tea.WindowSize(),
		func() tea.Msg {
			m.menu.SetState(ui.StateDefault)
			return nil
		},
	)
}

// confirmAction shows a confirmation modal and stores the action to execute on confirm
func (m *home) confirmAction(message string, action tea.Cmd) tea.Cmd {
	m.state = stateConfirm
	m.pendingConfirmAction = action

	// Create and show the confirmation overlay using ConfirmationOverlay
	m.confirmationOverlay = overlay.NewConfirmationOverlay(message)
	// Set a fixed width for consistent appearance
	m.confirmationOverlay.SetWidth(50)

	// Set callbacks for confirmation and cancellation
	m.confirmationOverlay.OnConfirm = func() {
		m.state = stateDefault
	}

	m.confirmationOverlay.OnCancel = func() {
		m.state = stateDefault
		m.pendingConfirmAction = nil
	}

	return nil
}

// handleMouseClick routes a left-click to the appropriate panel.
func (m *home) handleMouseClick(x, y int) (tea.Model, tea.Cmd) {
	if m.state != stateDefault {
		return m, nil
	}

	if m.bounds.list.Contains(x, y) {
		localY := y - m.bounds.list.Y
		if idx := m.list.IndexAtY(localY); idx >= 0 {
			m.list.SetSelectedIndex(idx)
			return m, m.instanceChanged()
		}
	} else if m.kanbanVisible && m.bounds.kanban.Contains(x, y) {
		localX := x - m.bounds.kanban.X
		localY := y - m.bounds.kanban.Y
		if inst := m.kanban.HandleClick(localX, localY); inst != nil {
			m.kanban.SetCursorToInstance(inst)
			m.list.SelectInstance(inst)
			return m, m.instanceChanged()
		}
	}

	return m, nil
}

// handleMouseWheel routes scroll wheel events to the appropriate panel.
func (m *home) handleMouseWheel(button tea.MouseButton, x, y int) (tea.Model, tea.Cmd) {
	if m.state == stateInteractive {
		return m, nil
	}
	delta := 1
	if button == tea.MouseButtonWheelUp {
		delta = -1
	}

	if m.kanbanVisible && m.bounds.kanban.Contains(x, y) {
		localX := x - m.bounds.kanban.X
		colIdx := m.kanban.ColumnAtX(localX)
		m.kanban.ScrollColumn(colIdx, delta)
		return m, nil
	}

	if m.bounds.list.Contains(x, y) {
		if delta > 0 {
			m.list.Down()
		} else {
			m.list.Up()
		}
		return m, m.instanceChanged()
	}

	// Default: scroll the preview/diff/terminal pane
	selected := m.getActiveInstance()
	if selected == nil || selected.Status == session.Paused {
		return m, nil
	}
	if button == tea.MouseButtonWheelUp {
		m.tabbedWindow.ScrollUp()
	} else {
		m.tabbedWindow.ScrollDown()
	}
	return m, nil
}

func (m *home) View() string {
	previewWithPadding := lipgloss.NewStyle().PaddingTop(1).Render(m.tabbedWindow.String())

	var listAndPreview string
	if m.kanbanVisible {
		kanbanWithPadding := lipgloss.NewStyle().PaddingTop(1).Render(m.kanban.String())
		listAndPreview = lipgloss.JoinHorizontal(lipgloss.Top, kanbanWithPadding, previewWithPadding)
	} else {
		listWithPadding := lipgloss.NewStyle().PaddingTop(1).Render(m.list.String())
		listAndPreview = lipgloss.JoinHorizontal(lipgloss.Top, listWithPadding, previewWithPadding)
	}

	mainView := lipgloss.JoinVertical(
		lipgloss.Center,
		listAndPreview,
		m.menu.String(),
		m.errBox.String(),
	)

	if m.state == statePrompt || m.state == stateNotes {
		if m.textInputOverlay == nil {
			log.ErrorLog.Printf("text input overlay is nil")
		}
		return overlay.PlaceOverlay(0, 0, m.textInputOverlay.Render(), mainView, true, true)
	} else if m.state == stateHelp {
		if m.helpOverlay != nil {
			return overlay.PlaceOverlay(0, 0, m.helpOverlay.Render(), mainView, true, true)
		}
		if m.textOverlay == nil {
			log.ErrorLog.Printf("text overlay is nil")
		}
		return overlay.PlaceOverlay(0, 0, m.textOverlay.Render(), mainView, true, true)
	} else if m.state == stateConfirm {
		if m.confirmationOverlay == nil {
			log.ErrorLog.Printf("confirmation overlay is nil")
		}
		return overlay.PlaceOverlay(0, 0, m.confirmationOverlay.Render(), mainView, true, true)
	}

	return mainView
}

// copyToClipboard copies text to the system clipboard.
func copyToClipboard(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		cmd = exec.Command("xclip", "-selection", "clipboard")
	default:
		return fmt.Errorf("clipboard not supported on %s", runtime.GOOS)
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
