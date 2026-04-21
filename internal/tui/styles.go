package tui

import "charm.land/lipgloss/v2"

var (
	backgroundColor = lipgloss.Color("#0B1020")
	panelColor      = lipgloss.Color("#1E293B")
	focusedColor    = lipgloss.Color("#38BDF8")
	accentColor     = lipgloss.Color("#F97316")
	successColor    = lipgloss.Color("#10B981")
	warnColor       = lipgloss.Color("#F59E0B")
	dangerColor     = lipgloss.Color("#EF4444")
	mutedColor      = lipgloss.Color("#94A3B8")
	textColor       = lipgloss.Color("#E2E8F0")
	subtleColor     = lipgloss.Color("#CBD5E1")

	appStyle = lipgloss.NewStyle().
			Background(backgroundColor).
			Foreground(textColor)

	headerStyle = lipgloss.NewStyle().
			Foreground(textColor).
			Background(lipgloss.Color("#111827")).
			Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F8FAFC"))

	subtitleStyle = lipgloss.NewStyle().
			Foreground(subtleColor)

	metaStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(panelColor).
			Padding(0, 1)

	focusedPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(focusedColor).
				Padding(0, 1)

	panelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F8FAFC"))

	statusOpenStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#052E16")).
			Background(successColor).
			Padding(0, 1)

	statusRunningStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#082F49")).
				Background(focusedColor).
				Padding(0, 1)

	statusBlockedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#451A03")).
				Background(warnColor).
				Padding(0, 1)

	statusClosedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#0F172A")).
				Background(mutedColor).
				Padding(0, 1)

	statusErrorStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#F8FAFC")).
				Background(dangerColor).
				Padding(0, 1)

	footerStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Padding(0, 1)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F8FAFC")).
				Background(lipgloss.Color("#172554"))

	selectedUnfocusedItemStyle = lipgloss.NewStyle().
					Foreground(textColor).
					Background(lipgloss.Color("#1F2937"))
)

func statusBadge(status string) string {
	switch status {
	case "open", "ready", "action_completed":
		return statusOpenStyle.Render(status)
	case "running", "dispatch", "plan", "update", "message", "chunk", "channel_open", "channel_update", "action_started":
		return statusRunningStyle.Render(status)
	case "blocked":
		return statusBlockedStyle.Render(status)
	case "closed", "channel_close":
		return statusClosedStyle.Render(status)
	case "failed", "error":
		return statusErrorStyle.Render(status)
	default:
		return statusClosedStyle.Render(status)
	}
}
