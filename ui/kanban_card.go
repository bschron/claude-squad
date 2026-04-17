package ui

import (
	"claude-squad/session"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

var (
	cardDimBorder = lipgloss.AdaptiveColor{Light: "#888888", Dark: "#555555"}

	cardBranchStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"})

	cardTimeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"})

	cardStyleDim = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(cardDimBorder).
			Padding(0, 1)

	cardStyleSel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(highlightColor).
			Padding(0, 1)
)

// KanbanCardHeight is the total rendered height of a kanban card:
// 3 content lines + top border + bottom border = 5.
const KanbanCardHeight = 5

// cardKeyFor builds a cache key that captures every input renderCard reads.
func cardKeyFor(inst *session.Instance, selected bool, width int, icon string) cardCacheKey {
	k := cardCacheKey{
		status:   inst.Status,
		title:    inst.Title,
		branch:   inst.Branch,
		selected: selected,
		width:    width,
		elapsed:  formatDuration(time.Since(inst.CreatedAt)),
		icon:     icon,
	}
	if stat := inst.GetDiffStats(); stat != nil && stat.Error == nil && !stat.IsEmpty() {
		k.diffOK = true
		k.added = stat.Added
		k.removed = stat.Removed
	}
	return k
}

// renderCard renders a single kanban card for the given instance.
// icon is the pre-rendered status icon (spinner or static), supplied by the
// caller so the spinner view is computed once per render, not once per card.
func renderCard(inst *session.Instance, selected bool, width int, icon string) string {
	if inst == nil {
		return ""
	}

	innerWidth := width - 4 // 2 for border + 2 for padding
	if innerWidth < 6 {
		innerWidth = 6
	}

	style := cardStyleDim
	if selected {
		style = cardStyleSel
	}
	style = style.Width(width - 2) // account for border

	// -- Line 1: Status icon + Title --
	titleText := inst.Title
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
	branchWidth := runewidth.StringWidth(branch)
	if branchAvail > 0 && branchWidth > branchAvail {
		if branchAvail > 3 {
			branch = runewidth.Truncate(branch, branchAvail-3, "...")
			branchWidth = runewidth.StringWidth(branch)
		} else {
			branch = ""
			branchWidth = 0
		}
	}

	var branchLine string
	if diffStr != "" {
		spaces := ""
		remaining := innerWidth - branchWidth - diffWidth
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

	return style.Render(strings.Join(lines, "\n"))
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
