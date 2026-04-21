package ui

import "charm.land/lipgloss/v2"

var (
	panelBorderColor = lipgloss.Color("#3F3F46")
	mutedColor       = lipgloss.Color("#94A3B8")
	accentColor      = lipgloss.Color("#7C3AED")
	infoColor        = lipgloss.Color("#2563EB")
	successColor     = lipgloss.Color("#059669")
	warnColor        = lipgloss.Color("#D97706")
	dangerColor      = lipgloss.Color("#DC2626")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(accentColor).
			Padding(0, 1)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(panelBorderColor).
			Padding(0, 1)

	channelCardStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(panelBorderColor).
				Padding(0, 1)

	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA"))

	labelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(mutedColor)

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5E7EB"))

	footerStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	eventMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#E5E7EB")).
				PaddingLeft(2)

	tableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#E5E7EB"))

	tableEvenRowStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#E5E7EB"))

	tableOddRowStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#CBD5E1"))
)

var statusStyles = map[string]lipgloss.Style{
	"ready":   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(successColor).Padding(0, 1),
	"running": lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(infoColor).Padding(0, 1),
	"blocked": lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#111827")).Background(warnColor).Padding(0, 1),
	"failed":  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(dangerColor).Padding(0, 1),
}

var channelStatusStyles = map[string]lipgloss.Style{
	"open":    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(successColor).Padding(0, 1),
	"closed":  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(mutedColor).Padding(0, 1),
	"blocked": lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#111827")).Background(warnColor).Padding(0, 1),
}

var metricStyles = map[string]lipgloss.Style{
	"agents":  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(accentColor).Padding(0, 1),
	"running": lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(infoColor).Padding(0, 1),
	"queue":   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(successColor).Padding(0, 1),
	"failed":  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(dangerColor).Padding(0, 1),
}

var eventKindStyles = map[string]lipgloss.Style{
	"dispatch":       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(accentColor).Padding(0, 1),
	"plan":           lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(infoColor).Padding(0, 1),
	"update":         lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(successColor).Padding(0, 1),
	"message":        lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(infoColor).Padding(0, 1),
	"chunk":          lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#111827")).Background(lipgloss.Color("#93C5FD")).Padding(0, 1),
	"action_started": lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(accentColor).Padding(0, 1),
	"action_completed": lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(successColor).
		Padding(0, 1),
	"blocked":        lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#111827")).Background(warnColor).Padding(0, 1),
	"error":          lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(dangerColor).Padding(0, 1),
	"channel_open":   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(successColor).Padding(0, 1),
	"channel_close":  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(mutedColor).Padding(0, 1),
	"channel_update": lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(infoColor).Padding(0, 1),
}
