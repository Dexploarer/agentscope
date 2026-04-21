package agent

import (
	"slices"
	"strings"
	"time"
)

type Snapshot struct {
	Workspace string    `json:"workspace"`
	UpdatedAt time.Time `json:"updatedAt"`
	Queue     Queue     `json:"queue"`
	Agents    []Agent   `json:"agents"`
	Channels  []Channel `json:"channels"`
	Events    []Event   `json:"events"`
}

type Queue struct {
	Pending int `json:"pending"`
	Running int `json:"running"`
	Failed  int `json:"failed"`
}

type Agent struct {
	Name      string `json:"name"`
	Role      string `json:"role"`
	Status    string `json:"status"`
	Tasks     int    `json:"tasks"`
	Model     string `json:"model"`
	LastEvent string `json:"lastEvent"`
}

type Channel struct {
	Name      string   `json:"name"`
	Topic     string   `json:"topic"`
	Status    string   `json:"status"`
	Members   []string `json:"members"`
	LastEvent string   `json:"lastEvent"`
}

type Event struct {
	Time    time.Time      `json:"time"`
	Agent   string         `json:"agent"`
	Channel string         `json:"channel,omitempty"`
	Kind    string         `json:"kind"`
	Status  string         `json:"status,omitempty"`
	Topic   string         `json:"topic,omitempty"`
	Source  string         `json:"source,omitempty"`
	RunID   string         `json:"runId,omitempty"`
	RoomID  string         `json:"roomId,omitempty"`
	WorldID string         `json:"worldId,omitempty"`
	Members []string       `json:"members,omitempty"`
	Data    map[string]any `json:"data,omitempty"`
	Message string         `json:"message"`
}

type ChannelView struct {
	Name      string
	Topic     string
	Status    string
	Members   []string
	LastEvent string
	Events    []Event
	UpdatedAt time.Time
}

type Summary struct {
	TotalAgents    int
	RunningAgents  int
	ReadyAgents    int
	BlockedAgents  int
	FailedAgents   int
	PendingQueue   int
	RunningQueue   int
	FailedQueue    int
	TotalChannels  int
	OpenChannels   int
	ClosedChannels int
}

func (s Snapshot) Summary() Summary {
	channels := s.ChannelViews(0)

	summary := Summary{
		TotalAgents:   len(s.Agents),
		PendingQueue:  s.Queue.Pending,
		RunningQueue:  s.Queue.Running,
		FailedQueue:   s.Queue.Failed,
		TotalChannels: len(channels),
	}

	for _, current := range s.Agents {
		switch current.Status {
		case "running":
			summary.RunningAgents++
		case "ready":
			summary.ReadyAgents++
		case "blocked":
			summary.BlockedAgents++
		case "failed":
			summary.FailedAgents++
		}
	}

	for _, current := range channels {
		switch current.Status {
		case "closed":
			summary.ClosedChannels++
		default:
			summary.OpenChannels++
		}
	}

	return summary
}

func (s Snapshot) RecentEvents(limit int) []Event {
	if limit <= 0 || len(s.Events) <= limit {
		return append([]Event(nil), s.Events...)
	}

	return append([]Event(nil), s.Events[len(s.Events)-limit:]...)
}

func (s Snapshot) ChannelViews(limit int) []ChannelView {
	views := make([]ChannelView, 0, len(s.Channels))
	indexByName := make(map[string]int, len(s.Channels))

	for _, channel := range s.Channels {
		normalized := ChannelView{
			Name:      channel.Name,
			Topic:     channel.Topic,
			Status:    fallback(channel.Status, "open"),
			Members:   append([]string(nil), channel.Members...),
			LastEvent: channel.LastEvent,
		}
		indexByName[channel.Name] = len(views)
		views = append(views, normalized)
	}

	for _, current := range s.Events {
		if current.Channel == "" {
			continue
		}

		index, ok := indexByName[current.Channel]
		if !ok {
			index = len(views)
			indexByName[current.Channel] = index
			views = append(views, ChannelView{
				Name:   current.Channel,
				Status: "open",
			})
		}

		view := &views[index]
		if current.Topic != "" {
			view.Topic = current.Topic
		}
		if len(current.Members) > 0 {
			view.Members = mergeUnique(view.Members, current.Members)
		}
		if current.Agent != "" {
			view.Members = mergeUnique(view.Members, []string{current.Agent})
		}
		if current.Status != "" {
			view.Status = current.Status
		}

		switch current.Kind {
		case "channel_open":
			view.Status = fallback(view.Status, "open")
		case "channel_close":
			view.Status = "closed"
		case "channel_update":
			view.Status = fallback(view.Status, "open")
		default:
			view.Status = fallback(view.Status, "open")
		}

		if current.Message != "" {
			view.LastEvent = current.Message
		}
		if view.UpdatedAt.IsZero() || current.Time.After(view.UpdatedAt) {
			view.UpdatedAt = current.Time
		}

		if limit != 0 {
			view.Events = appendTrimmed(view.Events, current, limit)
		} else {
			view.Events = append(view.Events, current)
		}
	}

	return views
}

func (s Snapshot) SortedChannelViews(limit int) []ChannelView {
	views := s.ChannelViews(limit)
	slices.SortFunc(views, func(a, b ChannelView) int {
		switch {
		case a.UpdatedAt.After(b.UpdatedAt):
			return -1
		case a.UpdatedAt.Before(b.UpdatedAt):
			return 1
		default:
			return strings.Compare(a.Name, b.Name)
		}
	})
	return views
}

func (s Snapshot) EventsForChannel(name string) []Event {
	if name == "" {
		return nil
	}

	events := make([]Event, 0)
	for _, current := range s.Events {
		if current.Channel == name {
			events = append(events, current)
		}
	}
	return events
}

func (s Snapshot) ChannelByName(name string) (ChannelView, bool) {
	for _, current := range s.ChannelViews(0) {
		if current.Name == name {
			return current, true
		}
	}
	return ChannelView{}, false
}

func NewSnapshot(workspace string) Snapshot {
	return Snapshot{
		Workspace: fallback(workspace, "live-fleet"),
	}
}

func fallback(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}
