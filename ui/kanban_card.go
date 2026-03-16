package ui

import (
	"claude-squad/session"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

var (
	cardDimBorder = lipgloss.AdaptiveColor{Light: "#888888", Dark: "#555555"}

	cardBranchStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"})

	cardTimeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"})

)

// renderCard renders a single kanban card for the given instance.
func renderCard(inst *session.Instance, selected bool, width int, sp *spinner.Model) string {
	if inst == nil {
		return ""
	}

	borderColor := cardDimBorder
	if selected {
		borderColor = highlightColor
	}

	innerWidth := width - 4 // 2 for border + 2 for padding
	if innerWidth < 6 {
		innerWidth = 6
	}

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(width - 2). // account for border
		Padding(0, 1)

	// -- Line 1: Status icon + Title --
	var icon string
	switch inst.Status {
	case session.Running, session.Loading:
		if sp != nil {
			icon = readyStyle.Render(sp.View()) + " "
		} else {
			icon = readyStyle.Render(readyIcon)
		}
	case session.Ready:
		icon = readyStyle.Render(readyIcon)
	case session.Paused:
		icon = pausedStyle.Render(pausedIcon)
	}

	titleText := inst.Title
	// Truncate title if too long (leave room for icon ~2 chars)
	maxTitle := innerWidth - 3
	if maxTitle > 0 && runewidth.StringWidth(titleText) > maxTitle {
		titleText = runewidth.Truncate(titleText, maxTitle-3, "...")
	}
	titleLine := fmt.Sprintf("%s%s", icon, titleText)

	// -- Line 2: Branch + diff stats --
	branch := inst.Branch
	if branch == "" {
		branch = "-"
	}

	var diffStr string
	stat := inst.GetDiffStats()
	if stat != nil && stat.Error == nil && !stat.IsEmpty() {
		diffStr = fmt.Sprintf("+%d/-%d", stat.Added, stat.Removed)
	}

	diffWidth := runewidth.StringWidth(diffStr)
	branchAvail := innerWidth - diffWidth
	if diffWidth > 0 {
		branchAvail -= 1 // space between branch and diff
	}
	if branchAvail > 0 && runewidth.StringWidth(branch) > branchAvail {
		if branchAvail > 3 {
			branch = runewidth.Truncate(branch, branchAvail-3, "...")
		} else {
			branch = ""
		}
	}

	var branchLine string
	if diffStr != "" {
		spaces := ""
		remaining := innerWidth - runewidth.StringWidth(branch) - diffWidth
		if remaining > 0 {
			spaces = strings.Repeat(" ", remaining)
		}
		branchLine = cardBranchStyle.Render(branch) + spaces +
			addedLinesStyle.Render(fmt.Sprintf("+%d", stat.Added)) +
			cardBranchStyle.Render("/") +
			removedLinesStyle.Render(fmt.Sprintf("-%d", stat.Removed))
	} else {
		branchLine = cardBranchStyle.Render(branch)
	}

	// -- Line 3: Elapsed time --
	elapsed := time.Since(inst.CreatedAt)
	timeLine := cardTimeStyle.Render(fmt.Sprintf("%s %s", "\u23f1", formatDuration(elapsed)))

	lines := []string{titleLine, branchLine, timeLine}

	return cardStyle.Render(strings.Join(lines, "\n"))
}

// formatDuration returns a human-readable short duration string.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h >= 24 {
		days := h / 24
		h = h % 24
		return fmt.Sprintf("%dd %dh", days, h)
	}
	return fmt.Sprintf("%dh %dm", h, m)
}
