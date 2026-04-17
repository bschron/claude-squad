package kanban_test

import (
	"fmt"
	"testing"
	"time"

	"claude-squad/ui"

	"github.com/charmbracelet/bubbles/spinner"
)

// TestRegimeCPU simulates ~20Hz render of the Kanban in steady-state and
// reports CPU% on one core. This is descriptive, not pass/fail.
func TestRegimeCPU(t *testing.T) {
	scenarios := []struct {
		name       string
		n, w, h    int
		durationMs int
		fps        int
	}{
		{"10inst_120x40_22fps", 10, 120, 40, 3000, 22},
		{"30inst_120x40_22fps", 30, 120, 40, 3000, 22},
		{"60inst_200x60_22fps", 60, 200, 60, 3000, 22},
		{"60inst_200x60_60fps", 60, 200, 60, 3000, 60},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			instances := buildInstances(sc.n)
			sp := spinner.New(spinner.WithSpinner(spinner.MiniDot))
			kb := ui.NewKanbanBoard(&sp)
			kb.SetSize(sc.w, sc.h)
			kb.UpdateInstances(instances, instances[0])

			frameInterval := time.Second / time.Duration(sc.fps)
			deadline := time.Now().Add(time.Duration(sc.durationMs) * time.Millisecond)

			var renders int
			var totalRenderTime time.Duration
			wallStart := time.Now()
			for time.Now().Before(deadline) {
				frameStart := time.Now()
				_ = kb.String()
				renderElapsed := time.Since(frameStart)
				totalRenderTime += renderElapsed
				renders++

				// Sleep to match fps
				remaining := frameInterval - renderElapsed
				if remaining > 0 {
					time.Sleep(remaining)
				}
			}
			wall := time.Since(wallStart)
			cpuPct := float64(totalRenderTime) / float64(wall) * 100

			fmt.Printf("  %s: %d renders in %v, avg render=%v, CPU~%.1f%% of 1 core\n",
				sc.name, renders, wall.Round(time.Millisecond),
				(totalRenderTime / time.Duration(renders)).Round(time.Microsecond),
				cpuPct)
		})
	}
}
