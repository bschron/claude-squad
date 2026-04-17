package kanban_test

import (
	"claude-squad/session"
	"claude-squad/ui"
	"fmt"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
)

// buildInstances creates n instances split roughly evenly across the 3 kanban statuses.
func buildInstances(n int) []*session.Instance {
	statuses := []session.Status{session.Running, session.Ready, session.Paused}
	out := make([]*session.Instance, 0, n)
	for i := 0; i < n; i++ {
		st := statuses[i%3]
		out = append(out, makeTestInstance(
			fmt.Sprintf("session-%03d-with-a-moderately-long-title", i),
			st,
			fmt.Sprintf("feature/branch-%03d-long-name", i),
		))
	}
	return out
}

// ---------- Kanban rendering ----------

func benchmarkKanbanString(b *testing.B, n, width, height int) {
	instances := buildInstances(n)
	sp := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	kb := ui.NewKanbanBoard(&sp)
	kb.SetSize(width, height)
	kb.UpdateInstances(instances, instances[0])

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = kb.String()
	}
}

func BenchmarkKanbanString_10instances_120x40(b *testing.B)  { benchmarkKanbanString(b, 10, 120, 40) }
func BenchmarkKanbanString_30instances_120x40(b *testing.B)  { benchmarkKanbanString(b, 30, 120, 40) }
func BenchmarkKanbanString_60instances_120x40(b *testing.B)  { benchmarkKanbanString(b, 60, 120, 40) }
func BenchmarkKanbanString_10instances_200x60(b *testing.B)  { benchmarkKanbanString(b, 10, 200, 60) }
func BenchmarkKanbanString_60instances_200x60(b *testing.B)  { benchmarkKanbanString(b, 60, 200, 60) }

// ---------- Kanban UpdateInstances (classification + sort) ----------

func BenchmarkKanbanUpdateInstances_60(b *testing.B) {
	instances := buildInstances(60)
	sp := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	kb := ui.NewKanbanBoard(&sp)
	kb.SetSize(120, 40)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		kb.UpdateInstances(instances, instances[0])
	}
}

// ---------- List rendering (baseline for comparison) ----------

func benchmarkListString(b *testing.B, n, width, height int) {
	instances := buildInstances(n)
	sp := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	list := ui.NewList(&sp, false)
	for _, inst := range instances {
		list.AddInstance(inst)
	}
	list.SetSize(width, height)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = list.String()
	}
}

func BenchmarkListString_10instances_120x40(b *testing.B)  { benchmarkListString(b, 10, 120, 40) }
func BenchmarkListString_30instances_120x40(b *testing.B)  { benchmarkListString(b, 30, 120, 40) }
func BenchmarkListString_60instances_120x40(b *testing.B)  { benchmarkListString(b, 60, 120, 40) }
func BenchmarkListString_10instances_200x60(b *testing.B)  { benchmarkListString(b, 10, 200, 60) }
func BenchmarkListString_60instances_200x60(b *testing.B)  { benchmarkListString(b, 60, 200, 60) }
