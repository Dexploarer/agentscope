package tui

import (
	"fmt"
	"io"
	"strings"

	"agentscope/internal/agent"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type channelItem struct {
	view agent.ChannelView
}

func (i channelItem) FilterValue() string {
	return strings.Join([]string{
		i.view.Name,
		i.view.Topic,
		strings.Join(i.view.Members, " "),
		i.view.LastEvent,
	}, " ")
}

func (i channelItem) Title() string {
	return "#" + i.view.Name
}

func (i channelItem) Description() string {
	return i.view.LastEvent
}

type eventItem struct {
	event agent.Event
}

func (i eventItem) FilterValue() string {
	return strings.Join([]string{
		i.event.Kind,
		i.event.Agent,
		i.event.Message,
		i.event.Channel,
		i.event.Topic,
		i.event.Source,
	}, " ")
}

func (i eventItem) Title() string {
	return fmt.Sprintf("%s %s", i.event.Time.Format("15:04:05"), strings.ToUpper(i.event.Kind))
}

func (i eventItem) Description() string {
	return i.event.Message
}

type channelDelegate struct {
	focused bool
}

func (d channelDelegate) Height() int  { return 3 }
func (d channelDelegate) Spacing() int { return 0 }
func (d channelDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}
func (d channelDelegate) ShortHelp() []key.Binding  { return nil }
func (d channelDelegate) FullHelp() [][]key.Binding { return nil }
func (d channelDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	channel, ok := item.(channelItem)
	if !ok {
		return
	}

	selected := index == m.Index() || index == m.GlobalIndex()
	container := lipgloss.NewStyle().Width(m.Width())
	if selected {
		if d.focused {
			container = selectedItemStyle.Copy().Width(m.Width())
		} else {
			container = selectedUnfocusedItemStyle.Copy().Width(m.Width())
		}
	}

	header := lipgloss.JoinHorizontal(
		lipgloss.Left,
		titleStyle.Render(channel.Title()),
		" ",
		statusBadge(channel.view.Status),
	)

	meta := metaStyle.Render(truncate(strings.Join(channel.view.Members, ", "), max(m.Width()-10, 8)))
	body := lipgloss.NewStyle().Foreground(subtleColor).Render(
		truncate(channel.view.LastEvent, max(m.Width()-4, 12)),
	)

	rendered := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		meta,
		body,
	)

	fmt.Fprint(w, container.Render(rendered))
}

type eventDelegate struct {
	focused bool
}

func (d eventDelegate) Height() int  { return 3 }
func (d eventDelegate) Spacing() int { return 0 }
func (d eventDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}
func (d eventDelegate) ShortHelp() []key.Binding  { return nil }
func (d eventDelegate) FullHelp() [][]key.Binding { return nil }
func (d eventDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	event, ok := item.(eventItem)
	if !ok {
		return
	}

	selected := index == m.Index() || index == m.GlobalIndex()
	container := lipgloss.NewStyle().Width(m.Width())
	if selected {
		if d.focused {
			container = selectedItemStyle.Copy().Width(m.Width())
		} else {
			container = selectedUnfocusedItemStyle.Copy().Width(m.Width())
		}
	}

	metaParts := []string{
		metaStyle.Render(event.event.Time.Format("15:04:05")),
		statusBadge(event.event.Kind),
		metaStyle.Render(event.event.Agent),
	}
	if event.event.Channel != "" {
		metaParts = append(metaParts, metaStyle.Render("#"+event.event.Channel))
	}

	header := lipgloss.JoinHorizontal(lipgloss.Left, joinWithSpaces(metaParts)...)
	body := lipgloss.NewStyle().Foreground(subtleColor).Render(
		truncate(event.event.Message, max(m.Width()-4, 12)),
	)

	fmt.Fprint(w, container.Render(lipgloss.JoinVertical(lipgloss.Left, header, body)))
}
