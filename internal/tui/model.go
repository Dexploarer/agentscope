package tui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"agentscope/internal/agent"
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type FocusPane int

const (
	focusChannels FocusPane = iota
	focusEvents
	focusDetail
)

type StreamEnvelope struct {
	Event agent.Event
	Err   error
}

type streamMessage struct {
	envelope StreamEnvelope
	ok       bool
}

type Options struct {
	Width           int
	Height          int
	Live            bool
	Stream          <-chan StreamEnvelope
	Connection      string
	QuitOnStreamEnd bool
}

type Model struct {
	board       *agent.Board
	keys        KeyMap
	help        help.Model
	spinner     spinner.Model
	channels    list.Model
	events      list.Model
	detail      viewport.Model
	stream      <-chan StreamEnvelope
	renderWidth int

	channelDelegate *channelDelegate
	eventDelegate   *eventDelegate

	width           int
	height          int
	selectedChannel string
	showClosed      bool
	follow          bool
	live            bool
	connection      string
	quitOnStreamEnd bool
	streamDone      bool
	streamErr       error
	focus           FocusPane
}

func New(snapshot agent.Snapshot, opts Options) (Model, error) {
	channelDelegate := &channelDelegate{}
	eventDelegate := &eventDelegate{}

	channels := list.New(nil, channelDelegate, max(opts.Width/4, 28), max(opts.Height-8, 12))
	channels.Title = "Channels"
	channels.SetShowHelp(false)
	channels.SetShowStatusBar(false)
	channels.SetShowPagination(false)
	channels.SetFilteringEnabled(true)

	events := list.New(nil, eventDelegate, max(opts.Width/3, 40), max(opts.Height-8, 12))
	events.Title = "Events"
	events.SetShowHelp(false)
	events.SetShowStatusBar(false)
	events.SetShowPagination(false)
	events.SetFilteringEnabled(true)

	detail := viewport.New(
		viewport.WithWidth(max(opts.Width/3, 44)),
		viewport.WithHeight(max(opts.Height-8, 12)),
	)
	detail.MouseWheelEnabled = true
	detail.SoftWrap = true
	detail.FillHeight = true

	m := Model{
		board:           agent.NewBoard(snapshot),
		keys:            DefaultKeyMap(),
		help:            help.New(),
		spinner:         spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithStyle(lipgloss.NewStyle().Foreground(accentColor))),
		channels:        channels,
		events:          events,
		detail:          detail,
		stream:          opts.Stream,
		renderWidth:     max(opts.Width/3, 44),
		channelDelegate: channelDelegate,
		eventDelegate:   eventDelegate,
		width:           max(opts.Width, 120),
		height:          max(opts.Height, 36),
		showClosed:      true,
		follow:          true,
		live:            opts.Live,
		connection:      strings.TrimSpace(opts.Connection),
		quitOnStreamEnd: opts.QuitOnStreamEnd,
		focus:           focusChannels,
	}

	m.help.Styles = help.DefaultStyles(true)
	m.help.ShowAll = false
	m.help.SetWidth(m.width - 4)
	m.rebuildLists("")
	m.updateFocus()
	m.layout(m.width, m.height)
	m.refreshDetail()

	return m, nil
}

func (m Model) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0, 2)
	if m.live {
		cmds = append(cmds, waitForStream(m.stream), m.spinner.Tick)
	}
	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if m.live {
		m.spinner, _ = m.spinner.Update(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.layout(msg.Width, msg.Height)
		return m, nil
	case streamMessage:
		if !msg.ok {
			m.streamDone = true
			m.live = false
			if m.quitOnStreamEnd {
				return m, tea.Quit
			}
			return m, nil
		}
		if msg.envelope.Err != nil {
			m.streamErr = msg.envelope.Err
			m.streamDone = true
			m.live = false
			m.refreshDetail()
			if m.quitOnStreamEnd {
				return m, tea.Quit
			}
			return m, nil
		}

		previousChannel := m.selectedChannel
		if err := m.board.ApplyEvent(msg.envelope.Event); err != nil {
			m.streamErr = err
			m.streamDone = true
			m.live = false
			m.refreshDetail()
			if m.quitOnStreamEnd {
				return m, tea.Quit
			}
			return m, nil
		}

		nextChannel := previousChannel
		if m.follow && msg.envelope.Event.Channel != "" {
			nextChannel = msg.envelope.Event.Channel
		}

		m.rebuildLists(nextChannel)
		m.refreshDetail()
		if m.stream != nil {
			cmds = append(cmds, waitForStream(m.stream))
		}
		return m, tea.Batch(cmds...)
	case tea.KeyPressMsg:
		if m.activeListFiltering() {
			return m.updateFocusedPane(msg)
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.ToggleHelp):
			m.help.ShowAll = !m.help.ShowAll
			return m, nil
		case key.Matches(msg, m.keys.NextPane):
			m.focus = (m.focus + 1) % 3
			m.updateFocus()
			return m, nil
		case key.Matches(msg, m.keys.PrevPane):
			m.focus = (m.focus + 2) % 3
			m.updateFocus()
			return m, nil
		case key.Matches(msg, m.keys.ToggleLive):
			m.follow = !m.follow
			m.refreshDetail()
			return m, nil
		case key.Matches(msg, m.keys.ToggleDone):
			m.showClosed = !m.showClosed
			m.rebuildLists(m.selectedChannel)
			m.refreshDetail()
			return m, nil
		case key.Matches(msg, m.keys.SelectPane):
			if m.focus == focusChannels {
				m.focus = focusEvents
				m.updateFocus()
				return m, nil
			}
		}
	}

	return m.updateFocusedPane(msg)
}

func (m Model) View() tea.View {
	content := appStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderHeader(),
			m.renderBody(),
			m.renderFooter(),
		),
	)

	view := tea.NewView(content)
	view.AltScreen = true
	view.ReportFocus = true
	view.MouseMode = tea.MouseModeCellMotion
	view.WindowTitle = "AgentScope Console"
	return view
}

func (m Model) ShortHelp() []key.Binding {
	return []key.Binding{
		m.keys.NextPane,
		m.keys.Filter,
		m.keys.ToggleLive,
		m.keys.ToggleDone,
		m.keys.ToggleHelp,
		m.keys.Quit,
	}
}

func (m Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{m.keys.NextPane, m.keys.PrevPane, m.keys.SelectPane},
		{m.keys.Filter, m.keys.ToggleLive, m.keys.ToggleDone},
		{m.keys.ToggleHelp, m.keys.Quit},
	}
}

func (m *Model) updateFocusedPane(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.focus {
	case focusChannels:
		previous := m.selectedChannel
		m.channels, cmd = m.channels.Update(msg)
		if selected := selectedChannelName(m.channels); selected != previous {
			m.selectedChannel = selected
			m.rebuildEvents()
			m.refreshDetail()
		}
	case focusEvents:
		m.events, cmd = m.events.Update(msg)
		m.refreshDetail()
	case focusDetail:
		m.detail, cmd = m.detail.Update(msg)
	}

	return *m, cmd
}

func (m *Model) activeListFiltering() bool {
	switch m.focus {
	case focusChannels:
		return m.channels.SettingFilter()
	case focusEvents:
		return m.events.SettingFilter()
	default:
		return false
	}
}

func (m *Model) updateFocus() {
	m.channelDelegate.focused = m.focus == focusChannels
	m.eventDelegate.focused = m.focus == focusEvents
}

func (m *Model) layout(width, height int) {
	if width <= 0 {
		width = m.width
	}
	if height <= 0 {
		height = m.height
	}

	m.width = max(width, 120)
	m.height = max(height, 30)
	m.help.SetWidth(m.width - 4)

	headerHeight := 4
	footerHeight := 2
	helpHeight := lipgloss.Height(m.help.View(m))
	bodyHeight := max(m.height-headerHeight-footerHeight-helpHeight-1, 16)

	channelWidth := max(min(32, m.width/4), 26)
	detailWidth := max(min(54, m.width/3), 38)
	eventWidth := max(m.width-channelWidth-detailWidth-8, 42)

	m.channels.SetSize(channelWidth, bodyHeight)
	m.events.SetSize(eventWidth, bodyHeight)
	m.detail.SetWidth(detailWidth)
	m.detail.SetHeight(bodyHeight - 2)
	m.renderWidth = detailWidth
	m.refreshDetail()
}

func (m *Model) rebuildLists(preferredChannel string) {
	snapshot := m.board.Snapshot()
	channelViews := snapshot.SortedChannelViews(6)

	items := make([]list.Item, 0, len(channelViews))
	for _, current := range channelViews {
		if !m.showClosed && current.Status == "closed" {
			continue
		}
		items = append(items, channelItem{view: current})
	}

	m.channels.SetItems(items)

	switch {
	case preferredChannel != "":
		selectChannelItem(&m.channels, preferredChannel)
	case m.selectedChannel != "":
		selectChannelItem(&m.channels, m.selectedChannel)
	case len(items) > 0:
		m.channels.Select(0)
	}

	m.selectedChannel = selectedChannelName(m.channels)
	m.rebuildEvents()
}

func (m *Model) rebuildEvents() {
	events := m.board.Snapshot().EventsForChannel(m.selectedChannel)
	items := make([]list.Item, 0, len(events))
	for _, current := range events {
		items = append(items, eventItem{event: current})
	}

	m.events.SetItems(items)

	if len(items) > 0 {
		if m.follow {
			m.events.Select(len(items) - 1)
		} else if m.events.Index() >= len(items) {
			m.events.Select(len(items) - 1)
		}
	}
}

func (m *Model) refreshDetail() {
	channel, ok := m.board.Snapshot().ChannelByName(m.selectedChannel)
	if !ok {
		m.detail.SetContent("Select a channel to inspect events.")
		return
	}

	content := channelDetail(channel)
	if selected := selectedEvent(m.events); selected != nil {
		content = eventDetail(channel, *selected)
	}
	if m.streamErr != nil {
		content += "\n\nStream Error\n" + strings.Repeat("─", 24) + "\n" + m.streamErr.Error()
	}

	m.detail.SetContent(renderDetail(content, m.renderWidth))
	if m.follow {
		m.detail.GotoTop()
	}
}

func (m Model) renderHeader() string {
	snapshot := m.board.Snapshot()
	summary := snapshot.Summary()

	left := lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render("AgentScope Control Room"),
		subtitleStyle.Render(snapshot.Workspace),
	)

	rightParts := []string{
		statusBadge(fmt.Sprintf("%d agents", summary.TotalAgents)),
		statusBadge(fmt.Sprintf("%d channels", summary.TotalChannels)),
		statusBadge(connectionLabel(m)),
	}

	meta := []string{
		fmt.Sprintf("follow=%t", m.follow),
		fmt.Sprintf("closed=%t", m.showClosed),
	}
	if m.connection != "" {
		meta = append(meta, "source="+truncate(m.connection, 40))
	}
	if !snapshot.UpdatedAt.IsZero() {
		meta = append(meta, snapshot.UpdatedAt.UTC().Format(time.RFC3339))
	}

	right := lipgloss.JoinVertical(
		lipgloss.Right,
		lipgloss.JoinHorizontal(lipgloss.Left, joinWithSpaces(rightParts)...),
		metaStyle.Render(strings.Join(meta, "  ")),
	)

	row := lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.NewStyle().Width(max(m.width-lipgloss.Width(right)-6, 40)).Render(left),
		right,
	)

	return headerStyle.Width(m.width).Render(row)
}

func (m Model) renderBody() string {
	channelPanel := panel("Channels", m.focus == focusChannels, m.channels.View(), m.channels.Width()+4)
	eventPanel := panel(panelTitle(m.selectedChannel, selectedEventCount(m.events)), m.focus == focusEvents, m.events.View(), m.events.Width()+4)
	detailPanel := panel("Detail", m.focus == focusDetail, m.detail.View(), m.detail.Width()+4)

	return lipgloss.JoinHorizontal(lipgloss.Top, channelPanel, eventPanel, detailPanel)
}

func (m Model) renderFooter() string {
	helpView := m.help.View(m)
	statusBits := []string{"arrows/jk move"}
	if m.live && !m.streamDone {
		statusBits = append(statusBits, m.spinner.View()+" live stream")
	}
	if m.streamDone {
		statusBits = append(statusBits, "stream ended")
	}
	if m.activeListFiltering() {
		statusBits = append(statusBits, "filtering")
	}
	if m.quitOnStreamEnd {
		statusBits = append(statusBits, "once")
	}

	line := lipgloss.JoinHorizontal(
		lipgloss.Left,
		lipgloss.NewStyle().Width(max(m.width-lipgloss.Width(helpView)-8, 30)).Render(strings.Join(statusBits, "  ")),
		helpView,
	)

	return footerStyle.Width(m.width).Render(line)
}

func panel(title string, focused bool, content string, width int) string {
	style := panelStyle.Copy().Width(width)
	if focused {
		style = focusedPanelStyle.Copy().Width(width)
	}

	return style.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			panelTitleStyle.Render(title),
			content,
		),
	)
}

func panelTitle(channel string, count int) string {
	if channel == "" {
		return "Events"
	}
	return fmt.Sprintf("#%s · %d events", channel, count)
}

func selectedEventCount(events list.Model) int {
	return len(events.Items())
}

func selectedChannelName(channels list.Model) string {
	selected, ok := channels.SelectedItem().(channelItem)
	if !ok {
		return ""
	}
	return selected.view.Name
}

func selectedEvent(events list.Model) *agent.Event {
	selected, ok := events.SelectedItem().(eventItem)
	if !ok {
		return nil
	}
	event := selected.event
	return &event
}

func selectChannelItem(channels *list.Model, name string) {
	for index, current := range channels.Items() {
		item, ok := current.(channelItem)
		if ok && item.view.Name == name {
			channels.Select(index)
			return
		}
	}
}

func waitForStream(stream <-chan StreamEnvelope) tea.Cmd {
	if stream == nil {
		return nil
	}
	return func() tea.Msg {
		envelope, ok := <-stream
		return streamMessage{envelope: envelope, ok: ok}
	}
}

func connectionLabel(m Model) string {
	switch {
	case m.streamErr != nil:
		return "error"
	case m.live && !m.streamDone:
		return "live"
	case m.streamDone:
		return "ended"
	default:
		return "snapshot"
	}
}

func channelDetail(channel agent.ChannelView) string {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("#%s\n\n", channel.Name))
	if channel.Topic != "" {
		builder.WriteString(channel.Topic + "\n\n")
	}
	builder.WriteString(fmt.Sprintf("Status: %s\n", channel.Status))
	if !channel.UpdatedAt.IsZero() {
		builder.WriteString(fmt.Sprintf("Updated: %s\n", channel.UpdatedAt.UTC().Format(time.RFC3339)))
	}
	if len(channel.Members) > 0 {
		builder.WriteString(fmt.Sprintf("Members: %s\n", strings.Join(channel.Members, ", ")))
	}
	if channel.LastEvent != "" {
		builder.WriteString("\nLast Event\n")
		builder.WriteString(strings.Repeat("─", 24) + "\n")
		builder.WriteString(channel.LastEvent + "\n")
	}
	if len(channel.Events) > 0 {
		builder.WriteString("\nRecent Activity\n")
		builder.WriteString(strings.Repeat("─", 24) + "\n")
		for _, current := range channel.Events {
			builder.WriteString(fmt.Sprintf("• %s  %s  %s  %s\n", current.Time.Format("15:04:05"), strings.ToUpper(current.Kind), current.Agent, current.Message))
		}
	}
	return builder.String()
}

func eventDetail(channel agent.ChannelView, event agent.Event) string {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("%s\n\n", strings.ToUpper(event.Kind)))
	builder.WriteString(fmt.Sprintf("Channel: #%s\n", channel.Name))
	builder.WriteString(fmt.Sprintf("Agent: %s\n", event.Agent))
	builder.WriteString(fmt.Sprintf("Time: %s\n", event.Time.UTC().Format(time.RFC3339)))
	if event.Source != "" {
		builder.WriteString(fmt.Sprintf("Source: %s\n", event.Source))
	}
	if event.RunID != "" {
		builder.WriteString(fmt.Sprintf("Run ID: %s\n", event.RunID))
	}
	if event.RoomID != "" {
		builder.WriteString(fmt.Sprintf("Room ID: %s\n", event.RoomID))
	}
	if event.WorldID != "" {
		builder.WriteString(fmt.Sprintf("World ID: %s\n", event.WorldID))
	}
	if event.Status != "" {
		builder.WriteString(fmt.Sprintf("Status: %s\n", event.Status))
	}

	builder.WriteString("\nMessage\n")
	builder.WriteString(strings.Repeat("─", 24) + "\n")
	builder.WriteString(event.Message + "\n")

	if event.Topic != "" {
		builder.WriteString("\nTopic\n")
		builder.WriteString(strings.Repeat("─", 24) + "\n")
		builder.WriteString(event.Topic + "\n")
	}
	if len(event.Members) > 0 {
		builder.WriteString("\nMembers\n")
		builder.WriteString(strings.Repeat("─", 24) + "\n")
		for _, current := range event.Members {
			builder.WriteString("• " + current + "\n")
		}
	}
	if len(event.Data) > 0 {
		encoded, err := json.MarshalIndent(event.Data, "", "  ")
		if err == nil {
			builder.WriteString("\nPayload\n")
			builder.WriteString(strings.Repeat("─", 24) + "\n")
			builder.Write(encoded)
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

func renderDetail(content string, width int) string {
	sections := strings.Split(content, "\n\n")
	rendered := make([]string, 0, len(sections))

	for index, section := range sections {
		if section == "" {
			continue
		}

		lines := strings.Split(section, "\n")
		if len(lines) == 0 {
			continue
		}

		switch index {
		case 0:
			rendered = append(rendered, titleStyle.Render(lines[0]))
			if len(lines) > 1 {
				rendered = append(rendered, lipgloss.NewStyle().Foreground(subtleColor).Width(max(width-6, 20)).Render(strings.Join(lines[1:], "\n")))
			}
		default:
			rendered = append(rendered, lipgloss.NewStyle().Foreground(subtleColor).Width(max(width-6, 20)).Render(strings.Join(lines, "\n")))
		}
	}

	return strings.Join(rendered, "\n\n")
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
