package keys

import (
	"github.com/charmbracelet/bubbles/key"
)

type KeyName int

const (
	KeyUp KeyName = iota
	KeyDown
	KeyEnter
	KeyNew
	KeyKill
	KeyQuit
	KeyReview
	KeyPush
	KeySubmit

	KeyTab        // Tab is a special keybinding for switching between panes.
	KeySubmitName // SubmitName is a special keybinding for submitting the name of a new instance.

	KeyCheckout
	KeyResume
	KeyPrompt // New key for entering a prompt
	KeyHelp   // Key for showing help screen

	// Diff keybindings
	KeyShiftUp
	KeyShiftDown

	KeyKanban        // Key for toggling kanban board view
	KeyLeft          // Key for moving left in kanban
	KeyRight         // Key for moving right in kanban
	KeyYank          // Key for copying tmux session name in kanban
	KeyInteractive   // Key for entering interactive mode
	KeyNotes         // Key for editing session notes
	KeyProjectPicker // Key for selecting projects to view
)

// GlobalKeyStringsMap is a global, immutable map string to keybinding.
var GlobalKeyStringsMap = map[string]KeyName{
	"up":        KeyUp,
	"down":      KeyDown,
	"ctrl+up":   KeyShiftUp,
	"ctrl+down": KeyShiftDown,
	"ctrl+t":    KeyPrompt,
	"enter":     KeyInteractive,
	"ctrl+o":    KeyEnter,
	"ctrl+n":    KeyNew,
	"ctrl+d":    KeyKill,
	"ctrl+q":    KeyQuit,
	"tab":       KeyTab,
	"ctrl+a":    KeyCheckout,
	"ctrl+r":    KeyResume,
	"ctrl+p":    KeySubmit,
	"?":         KeyHelp,
	"ctrl+k":    KeyKanban,
	"left":      KeyLeft,
	"right":     KeyRight,
	"ctrl+y":    KeyYank,
	"ctrl+e":    KeyNotes,
	"ctrl+g":    KeyProjectPicker,
}

// GlobalkeyBindings is a global, immutable map of KeyName tot keybinding.
var GlobalkeyBindings = map[KeyName]key.Binding{
	KeyUp: key.NewBinding(
		key.WithKeys("up"),
		key.WithHelp("↑", "up"),
	),
	KeyDown: key.NewBinding(
		key.WithKeys("down"),
		key.WithHelp("↓", "down"),
	),
	KeyShiftUp: key.NewBinding(
		key.WithKeys("ctrl+up"),
		key.WithHelp("ctrl+↑", "scroll"),
	),
	KeyShiftDown: key.NewBinding(
		key.WithKeys("ctrl+down"),
		key.WithHelp("ctrl+↓", "scroll"),
	),
	KeyEnter: key.NewBinding(
		key.WithKeys("ctrl+o"),
		key.WithHelp("ctrl+o", "open"),
	),
	KeyNew: key.NewBinding(
		key.WithKeys("ctrl+n"),
		key.WithHelp("ctrl+n", "new"),
	),
	KeyKill: key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("ctrl+d", "kill"),
	),
	KeyHelp: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	KeyQuit: key.NewBinding(
		key.WithKeys("ctrl+q"),
		key.WithHelp("ctrl+q", "quit"),
	),
	KeySubmit: key.NewBinding(
		key.WithKeys("ctrl+p"),
		key.WithHelp("ctrl+p", "push branch"),
	),
	KeyPrompt: key.NewBinding(
		key.WithKeys("ctrl+t"),
		key.WithHelp("ctrl+t", "new with prompt"),
	),
	KeyCheckout: key.NewBinding(
		key.WithKeys("ctrl+a"),
		key.WithHelp("ctrl+a", "checkout"),
	),
	KeyTab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch tab"),
	),
	KeyResume: key.NewBinding(
		key.WithKeys("ctrl+r"),
		key.WithHelp("ctrl+r", "resume/revive"),
	),

	KeyKanban: key.NewBinding(
		key.WithKeys("ctrl+k"),
		key.WithHelp("ctrl+k", "kanban"),
	),
	KeyLeft: key.NewBinding(
		key.WithKeys("left"),
		key.WithHelp("←", "left"),
	),
	KeyRight: key.NewBinding(
		key.WithKeys("right"),
		key.WithHelp("→", "right"),
	),
	KeyYank: key.NewBinding(
		key.WithKeys("ctrl+y"),
		key.WithHelp("ctrl+y", "copy tmux name"),
	),
	KeyInteractive: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("↵", "interact"),
	),
	KeyNotes: key.NewBinding(
		key.WithKeys("ctrl+e"),
		key.WithHelp("ctrl+e", "edit note"),
	),
	KeyProjectPicker: key.NewBinding(
		key.WithKeys("ctrl+g"),
		key.WithHelp("ctrl+g", "projects"),
	),

	// -- Special keybindings --

	KeySubmitName: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "submit name"),
	),
}
