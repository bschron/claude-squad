package kanban_test

import (
	"claude-squad/session"
	"claude-squad/ui"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
)

// makeTestInstance creates a minimal session.Instance for testing.
// Since NewInstance requires a valid path, we create via NewInstance and then
// override the exported fields we need for kanban board testing.
func makeTestInstance(title string, status session.Status, branch string) *session.Instance {
	inst, err := session.NewInstance(session.InstanceOptions{
		Title:   title,
		Path:    ".", // current directory; won't be started so path doesn't matter much
		Program: "claude",
	})
	if err != nil {
		// Fallback: we can't create via NewInstance if "." fails for abs path,
		// but this should work in practice.
		panic("makeTestInstance: " + err.Error())
	}
	inst.Status = status
	inst.Branch = branch
	inst.CreatedAt = time.Now().Add(-10 * time.Minute)
	return inst
}

// makeKanbanWithInstances creates a KanbanBoard, sets its size, and populates
// it with the given instances. The selected parameter may be nil.
func makeKanbanWithInstances(width, height int, instances []*session.Instance, selected *session.Instance) *ui.KanbanBoard {
	sp := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	kb := ui.NewKanbanBoard(&sp)
	kb.SetSize(width, height)
	kb.UpdateInstances(instances, selected)
	return kb
}
