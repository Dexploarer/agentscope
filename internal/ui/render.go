package ui

import (
	"fmt"
	"strings"

	"agentscope/internal/agent"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

func RenderDashboard(snapshot agent.Snapshot, width int) string {
	if width < 96 {
		width = 96
	}

	header := renderHeader(snapshot)
	overview := renderOverview(snapshot)
	eventsPanel := renderEventsPanel(snapshot.RecentEvents(8), width)

	if width < 132 {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			overview,
			renderAgentsPanel(snapshot, width),
			renderChannelsPanel(snapshot, width),
			eventsPanel,
			footerStyle.Render(`Event-driven mode: pipe NDJSON into "agentscope monitor"`),
		)
	}

	leftWidth := width/2 - 1
	rightWidth := width - leftWidth - 1

	body := lipgloss.JoinHorizontal(
		lipgloss.Top,
		renderAgentsPanel(snapshot, leftWidth),
		renderChannelsPanel(snapshot, rightWidth),
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		overview,
		body,
		eventsPanel,
		footerStyle.Render(`Event-driven mode: pipe NDJSON into "agentscope monitor"`),
	)
}

func RenderEvent(event agent.Event) string {
	headerParts := []string{
		labelStyle.Render(event.Time.Format("15:04:05")),
		eventBadge(event.Kind),
		sectionStyle.Render(event.Agent),
	}
	if event.Channel != "" {
		headerParts = append(headerParts, footerStyle.Render("#"+event.Channel))
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Left, joinWithSpaces(headerParts)...),
		eventMessageStyle.Render(event.Message),
	)
}

func renderHeader(snapshot agent.Snapshot) string {
	updated := "waiting for events"
	if !snapshot.UpdatedAt.IsZero() {
		updated = snapshot.UpdatedAt.UTC().Format("2006-01-02 15:04:05 MST")
	}

	lines := []string{
		titleStyle.Render("AgentScope"),
		labelStyle.Render("Workspace: ") + valueStyle.Render(snapshot.Workspace),
		labelStyle.Render("Updated: ") + valueStyle.Render(updated),
	}

	return panelStyle.Render(strings.Join(lines, "\n"))
}

func renderOverview(snapshot agent.Snapshot) string {
	summary := snapshot.Summary()

	metrics := lipgloss.JoinHorizontal(
		lipgloss.Left,
		metricBadge("agents", fmt.Sprintf("%d agents", summary.TotalAgents)),
		" ",
		metricBadge("running", fmt.Sprintf("%d active", summary.RunningAgents)),
		" ",
		metricBadge("queue", fmt.Sprintf("%d open channels", summary.OpenChannels)),
		" ",
		metricBadge("failed", fmt.Sprintf("%d failed", summary.FailedQueue+summary.FailedAgents)),
	)

	queueLine := fmt.Sprintf(
		"%s%s  %s%s  %s%s  %s%s",
		labelStyle.Render("Ready: "),
		valueStyle.Render(fmt.Sprintf("%d", summary.ReadyAgents)),
		labelStyle.Render("Blocked: "),
		valueStyle.Render(fmt.Sprintf("%d", summary.BlockedAgents)),
		labelStyle.Render("Closed channels: "),
		valueStyle.Render(fmt.Sprintf("%d", summary.ClosedChannels)),
		labelStyle.Render("Running queue: "),
		valueStyle.Render(fmt.Sprintf("%d", summary.RunningQueue)),
	)

	return panelStyle.Render(lipgloss.JoinVertical(lipgloss.Left, metrics, queueLine))
}

func renderAgentsPanel(snapshot agent.Snapshot, width int) string {
	nameWidth := 12
	roleWidth := 18
	modelWidth := 12
	eventWidth := 22

	switch {
	case width >= 140:
		roleWidth = 24
		modelWidth = 14
		eventWidth = 36
	case width >= 120:
		roleWidth = 22
		modelWidth = 14
		eventWidth = 30
	}

	rows := make([][]string, 0, len(snapshot.Agents))
	for _, current := range snapshot.Agents {
		rows = append(rows, []string{
			truncate(current.Name, nameWidth),
			truncate(current.Role, roleWidth),
			statusBadge(current.Status),
			fmt.Sprintf("%d", current.Tasks),
			truncate(current.Model, modelWidth),
			truncate(current.LastEvent, eventWidth),
		})
	}

	renderedTable := table.New().
		Border(lipgloss.HiddenBorder()).
		BorderHeader(true).
		Headers("AGENT", "ROLE", "STATUS", "TASKS", "MODEL", "LAST EVENT").
		Rows(rows...).
		StyleFunc(func(row, _ int) lipgloss.Style {
			switch {
			case row == table.HeaderRow:
				return tableHeaderStyle
			case row%2 == 0:
				return tableEvenRowStyle
			default:
				return tableOddRowStyle
			}
		}).
		Width(max(width-6, 60)).
		String()

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		sectionStyle.Render("Agents"),
		renderedTable,
	)

	return panelStyle.Width(max(width-4, 56)).Render(content)
}

func renderChannelsPanel(snapshot agent.Snapshot, width int) string {
	channelViews := snapshot.ChannelViews(3)
	lines := []string{sectionStyle.Render("Channels")}

	if len(channelViews) == 0 {
		lines = append(lines, footerStyle.Render("No channels opened yet"))
		return panelStyle.Width(max(width-4, 40)).Render(strings.Join(lines, "\n"))
	}

	columns := 1
	switch {
	case width >= 150:
		columns = 3
	case width >= 92:
		columns = 2
	}

	availableWidth := max(width-8, 32)
	cardWidth := max((availableWidth-(columns-1))/columns, 28)
	rows := make([]string, 0, (len(channelViews)+columns-1)/columns)

	for start := 0; start < len(channelViews); start += columns {
		end := min(start+columns, len(channelViews))
		cards := make([]string, 0, end-start)
		rowHeight := 0

		for _, current := range channelViews[start:end] {
			card := renderChannelCard(current, cardWidth)
			rowHeight = max(rowHeight, lipgloss.Height(card))
			cards = append(cards, card)
		}

		for index := range cards {
			cards[index] = lipgloss.NewStyle().Height(rowHeight).Render(cards[index])
		}

		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, joinWithSpaces(cards)...))
	}

	lines = append(lines, rows...)
	return panelStyle.Width(max(width-4, 40)).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func renderChannelCard(view agent.ChannelView, width int) string {
	header := lipgloss.JoinHorizontal(
		lipgloss.Left,
		sectionStyle.Render("#"+view.Name),
		" ",
		channelStatusBadge(view.Status),
	)

	lines := []string{header}
	if view.Topic != "" {
		lines = append(lines, valueStyle.Render(truncate(view.Topic, width-4)))
	}
	if len(view.Members) > 0 {
		lines = append(lines, labelStyle.Render("Members: ")+valueStyle.Render(truncate(strings.Join(view.Members, ", "), width-13)))
	}

	if len(view.Events) == 0 {
		if view.LastEvent != "" {
			lines = append(lines, eventMessageStyle.Render(truncate(view.LastEvent, width-4)))
		} else {
			lines = append(lines, footerStyle.Render("No channel traffic yet"))
		}
	} else {
		for _, current := range view.Events {
			lines = append(lines, renderCompactEvent(current, width-4))
		}
	}

	return channelCardStyle.Width(max(width-2, 26)).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func renderCompactEvent(event agent.Event, width int) string {
	headerParts := []string{
		labelStyle.Render(event.Time.Format("15:04")),
		eventBadge(event.Kind),
		valueStyle.Render(truncate(event.Agent, 14)),
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Left, joinWithSpaces(headerParts)...),
		footerStyle.Render(truncate(event.Message, width)),
	)
}

func renderEventsPanel(events []agent.Event, width int) string {
	lines := []string{sectionStyle.Render("Global Feed")}

	if len(events) == 0 {
		lines = append(lines, footerStyle.Render("No recent events"))
		return panelStyle.Width(max(width-4, 40)).Render(strings.Join(lines, "\n"))
	}

	for _, current := range events {
		lines = append(lines, RenderEvent(current))
	}

	return panelStyle.Width(max(width-4, 40)).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func statusBadge(status string) string {
	style, ok := statusStyles[status]
	if !ok {
		style = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#111827")).
			Background(mutedColor).
			Padding(0, 1)
	}
	return style.Render(strings.ToUpper(status))
}

func channelStatusBadge(status string) string {
	style, ok := channelStatusStyles[status]
	if !ok {
		style = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#111827")).
			Background(mutedColor).
			Padding(0, 1)
	}
	return style.Render(strings.ToUpper(status))
}

func eventBadge(kind string) string {
	style, ok := eventKindStyles[kind]
	if !ok {
		style = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#111827")).
			Background(mutedColor).
			Padding(0, 1)
	}
	return style.Render(strings.ToUpper(kind))
}

func metricBadge(kind, value string) string {
	style, ok := metricStyles[kind]
	if !ok {
		style = titleStyle
	}
	return style.Render(value)
}

func truncate(value string, limit int) string {
	if limit <= 0 || lipgloss.Width(value) <= limit {
		return value
	}

	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	if limit <= 3 {
		return string(runes[:1])
	}

	return string(runes[:limit-3]) + "..."
}

func joinWithSpaces(parts []string) []string {
	if len(parts) == 0 {
		return nil
	}

	joined := make([]string, 0, len(parts)*2-1)
	for index, current := range parts {
		if index > 0 {
			joined = append(joined, " ")
		}
		joined = append(joined, current)
	}
	return joined
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
