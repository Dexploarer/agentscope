package agent

import "fmt"

type Board struct {
	snapshot     Snapshot
	agentIndex   map[string]int
	channelIndex map[string]int
}

func NewBoard(snapshot Snapshot) *Board {
	board := &Board{
		snapshot:     snapshot,
		agentIndex:   make(map[string]int, len(snapshot.Agents)),
		channelIndex: make(map[string]int, len(snapshot.Channels)),
	}

	for index, current := range board.snapshot.Agents {
		board.agentIndex[current.Name] = index
	}
	for index, current := range board.snapshot.Channels {
		board.channelIndex[current.Name] = index
	}

	return board
}

func (b *Board) Snapshot() Snapshot {
	return b.snapshot
}

func (b *Board) ApplyEvent(event Event) error {
	event.normalize()
	if err := event.validate(); err != nil {
		return fmt.Errorf("apply event: %w", err)
	}

	if b.snapshot.Workspace == "" {
		b.snapshot.Workspace = "live-fleet"
	}
	if b.snapshot.UpdatedAt.IsZero() || event.Time.After(b.snapshot.UpdatedAt) {
		b.snapshot.UpdatedAt = event.Time
	}
	b.snapshot.Events = append(b.snapshot.Events, event)

	currentAgent := b.ensureAgent(event.Agent)
	currentAgent.LastEvent = event.Message
	updateAgentStatus(currentAgent, event)

	if event.Channel != "" {
		currentChannel := b.ensureChannel(event.Channel)
		if event.Topic != "" {
			currentChannel.Topic = event.Topic
		}
		if len(event.Members) > 0 {
			currentChannel.Members = mergeUnique(currentChannel.Members, event.Members)
		}
		currentChannel.Members = mergeUnique(currentChannel.Members, []string{event.Agent})
		if event.Message != "" {
			currentChannel.LastEvent = event.Message
		}

		switch event.Kind {
		case "channel_close":
			currentChannel.Status = "closed"
		case "channel_open":
			currentChannel.Status = fallback(event.Status, "open")
		case "channel_update":
			currentChannel.Status = fallback(event.Status, fallback(currentChannel.Status, "open"))
		default:
			currentChannel.Status = fallback(currentChannel.Status, "open")
		}
	}

	return nil
}

func (b *Board) ensureAgent(name string) *Agent {
	index, ok := b.agentIndex[name]
	if ok {
		return &b.snapshot.Agents[index]
	}

	b.snapshot.Agents = append(b.snapshot.Agents, Agent{
		Name:   name,
		Role:   "unspecified",
		Status: "ready",
		Model:  "event-stream",
	})
	index = len(b.snapshot.Agents) - 1
	b.agentIndex[name] = index
	return &b.snapshot.Agents[index]
}

func (b *Board) ensureChannel(name string) *Channel {
	index, ok := b.channelIndex[name]
	if ok {
		return &b.snapshot.Channels[index]
	}

	b.snapshot.Channels = append(b.snapshot.Channels, Channel{
		Name:   name,
		Status: "open",
	})
	index = len(b.snapshot.Channels) - 1
	b.channelIndex[name] = index
	return &b.snapshot.Channels[index]
}

func updateAgentStatus(current *Agent, event Event) {
	switch event.Kind {
	case "blocked":
		current.Status = "blocked"
	case "error", "failed":
		current.Status = "failed"
	case "ready":
		current.Status = "ready"
	case "dispatch", "plan", "update", "channel_open", "channel_update":
		current.Status = "running"
	}
}

func mergeUnique(existing []string, values []string) []string {
	seen := make(map[string]struct{}, len(existing))
	merged := append([]string(nil), existing...)

	for _, current := range existing {
		seen[current] = struct{}{}
	}
	for _, current := range values {
		if current == "" {
			continue
		}
		if _, ok := seen[current]; ok {
			continue
		}
		seen[current] = struct{}{}
		merged = append(merged, current)
	}

	return merged
}

func appendTrimmed(events []Event, event Event, limit int) []Event {
	events = append(events, event)
	if limit > 0 && len(events) > limit {
		events = append([]Event(nil), events[len(events)-limit:]...)
	}
	return events
}
