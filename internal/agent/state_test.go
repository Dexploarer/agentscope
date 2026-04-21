package agent

import (
	"testing"
	"time"
)

func TestBoardApplyEventBuildsChannelsFromStream(t *testing.T) {
	t.Parallel()

	board := NewBoard(NewSnapshot("live-fleet"))

	events := []Event{
		{
			Time:    time.Date(2026, time.April, 19, 11, 15, 20, 0, time.UTC),
			Agent:   "router",
			Channel: "intake",
			Kind:    "channel_open",
			Status:  "open",
			Topic:   "New work routing",
			Members: []string{"planner"},
			Message: "Opened intake channel",
		},
		{
			Time:    time.Date(2026, time.April, 19, 11, 16, 3, 0, time.UTC),
			Agent:   "router",
			Channel: "intake",
			Kind:    "dispatch",
			Message: "Assigned token-list sync",
		},
		{
			Time:    time.Date(2026, time.April, 19, 11, 19, 58, 0, time.UTC),
			Agent:   "executor",
			Channel: "deploy",
			Kind:    "channel_close",
			Message: "Closed deploy channel after verification",
		},
	}

	for _, current := range events {
		if err := board.ApplyEvent(current); err != nil {
			t.Fatalf("ApplyEvent returned error: %v", err)
		}
	}

	snapshot := board.Snapshot()
	if snapshot.UpdatedAt.IsZero() {
		t.Fatal("expected updated timestamp to be set")
	}
	if len(snapshot.Agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(snapshot.Agents))
	}
	if len(snapshot.Channels) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(snapshot.Channels))
	}

	views := snapshot.ChannelViews(3)
	intake := findChannelView(t, views, "intake")
	if intake.Topic != "New work routing" {
		t.Fatalf("expected intake topic to be set, got %q", intake.Topic)
	}
	if len(intake.Members) != 2 {
		t.Fatalf("expected intake members to be merged, got %v", intake.Members)
	}
	if intake.LastEvent != "Assigned token-list sync" {
		t.Fatalf("expected intake last event to update, got %q", intake.LastEvent)
	}

	deploy := findChannelView(t, views, "deploy")
	if deploy.Status != "closed" {
		t.Fatalf("expected deploy channel to be closed, got %q", deploy.Status)
	}
}

func findChannelView(t *testing.T, views []ChannelView, name string) ChannelView {
	t.Helper()

	for _, current := range views {
		if current.Name == name {
			return current
		}
	}

	t.Fatalf("channel %q not found", name)
	return ChannelView{}
}
