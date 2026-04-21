package tui

import (
	"strings"
	"testing"
	"time"

	"agentscope/internal/agent"
	tea "charm.land/bubbletea/v2"
)

func TestModelBuildsChannelAndEventLists(t *testing.T) {
	t.Parallel()

	snapshot, err := agent.SampleSnapshot()
	if err != nil {
		t.Fatalf("SampleSnapshot returned error: %v", err)
	}

	model, err := New(snapshot, Options{Width: 140, Height: 40})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if len(model.channels.Items()) == 0 {
		t.Fatal("expected channels to be populated")
	}
	if model.selectedChannel == "" {
		t.Fatal("expected selected channel to be set")
	}
	if len(model.events.Items()) == 0 {
		t.Fatal("expected events to be populated for selected channel")
	}
}

func TestModelAppliesIncomingStreamEvent(t *testing.T) {
	t.Parallel()

	model, err := New(agent.NewSnapshot("live"), Options{Width: 140, Height: 40, Live: true})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	msg := streamMessage{
		ok: true,
		envelope: StreamEnvelope{
			Event: agent.Event{
				Time:    time.Date(2026, time.April, 19, 12, 0, 0, 0, time.UTC),
				Agent:   "router",
				Channel: "intake",
				Kind:    "channel_open",
				Status:  "open",
				Topic:   "Live intake",
				Message: "Opened intake channel",
			},
		},
	}

	updatedModel, _ := model.Update(msg)
	updated := updatedModel.(Model)

	if updated.selectedChannel != "intake" {
		t.Fatalf("expected selected channel intake, got %q", updated.selectedChannel)
	}
	if len(updated.channels.Items()) != 1 {
		t.Fatalf("expected 1 channel item, got %d", len(updated.channels.Items()))
	}
	if len(updated.events.Items()) != 1 {
		t.Fatalf("expected 1 event item, got %d", len(updated.events.Items()))
	}
}

func TestModelResizesOnWindowSizeMessage(t *testing.T) {
	t.Parallel()

	snapshot, err := agent.SampleSnapshot()
	if err != nil {
		t.Fatalf("SampleSnapshot returned error: %v", err)
	}

	model, err := New(snapshot, Options{Width: 120, Height: 36})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	updatedModel, _ := model.Update(tea.WindowSizeMsg{Width: 160, Height: 48})
	updated := updatedModel.(Model)

	if updated.width != 160 {
		t.Fatalf("expected width 160, got %d", updated.width)
	}
	if updated.height != 48 {
		t.Fatalf("expected height 48, got %d", updated.height)
	}
	if updated.detail.Width() <= 0 || updated.detail.Height() <= 0 {
		t.Fatal("expected detail viewport size to be updated")
	}
}

func TestModelQuitsWhenConfiguredAfterStreamEnds(t *testing.T) {
	t.Parallel()

	model, err := New(agent.NewSnapshot("live"), Options{
		Width:           140,
		Height:          40,
		Live:            true,
		Connection:      "stdin",
		QuitOnStreamEnd: true,
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	updatedModel, cmd := model.Update(streamMessage{ok: false})
	updated := updatedModel.(Model)

	if !updated.streamDone {
		t.Fatal("expected streamDone to be set")
	}
	if updated.live {
		t.Fatal("expected live mode to be disabled")
	}
	if cmd == nil {
		t.Fatal("expected quit command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("expected tea.QuitMsg")
	}
}

func TestModelHeaderIncludesConnectionLabel(t *testing.T) {
	t.Parallel()

	snapshot, err := agent.SampleSnapshot()
	if err != nil {
		t.Fatalf("SampleSnapshot returned error: %v", err)
	}

	model, err := New(snapshot, Options{
		Width:      140,
		Height:     40,
		Connection: "unix:///tmp/agentscope.sock",
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	rendered := model.renderHeader()
	if !strings.Contains(rendered, "source=unix:///tmp/agentscope.sock") {
		t.Fatalf("expected rendered header to include connection label, got %q", rendered)
	}
}
