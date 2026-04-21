package agent

import (
	"strings"
	"testing"
	"time"
)

func TestLoadSnapshotSanitizesAgentFields(t *testing.T) {
	t.Parallel()

	input := `{
		"workspace":"prod-fleet",
		"updatedAt":"2026-04-19T11:20:00Z",
		"queue":{"pending":1,"running":2,"failed":0},
		"agents":[
			{
				"name":"planner",
				"role":"Plan \u001b[31mwork\u001b[0m",
				"status":"RUNNING",
				"tasks":3,
				"model":"gpt-5.4",
				"lastEvent":"Queued\nfollow-up"
			}
		],
		"channels":[
			{
				"name":"intake",
				"topic":"New \u001b[31mwork\u001b[0m",
				"status":"OPEN",
				"members":["planner","router"],
				"lastEvent":"Ready\nfor work"
			}
		],
		"events":[
			{
				"time":"2026-04-19T11:19:00Z",
				"agent":"planner",
				"channel":"intake",
				"kind":"UPDATE",
				"message":"Normalized\tmessage"
			}
		]
	}`

	snapshot, err := LoadSnapshot(strings.NewReader(input))
	if err != nil {
		t.Fatalf("LoadSnapshot returned error: %v", err)
	}

	if snapshot.Agents[0].Role != "Plan work" {
		t.Fatalf("expected sanitized role, got %q", snapshot.Agents[0].Role)
	}
	if snapshot.Agents[0].Status != "running" {
		t.Fatalf("expected lowercase status, got %q", snapshot.Agents[0].Status)
	}
	if snapshot.Agents[0].LastEvent != "Queued follow-up" {
		t.Fatalf("expected normalized last event, got %q", snapshot.Agents[0].LastEvent)
	}
	if snapshot.Events[0].Kind != "update" {
		t.Fatalf("expected lowercase kind, got %q", snapshot.Events[0].Kind)
	}
	if snapshot.Channels[0].Topic != "New work" {
		t.Fatalf("expected sanitized channel topic, got %q", snapshot.Channels[0].Topic)
	}
	if snapshot.Events[0].Channel != "intake" {
		t.Fatalf("expected normalized channel name, got %q", snapshot.Events[0].Channel)
	}
}

func TestLoadEventsParsesNDJSON(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		`{"time":"2026-04-19T11:16:03Z","agent":"router","channel":"intake","kind":"dispatch","message":"Assigned task"}`,
		`{"time":"2026-04-19T11:17:11Z","agent":"researcher","channel":"research","kind":"update","message":"Found regressions"}`,
	}, "\n")

	events, err := LoadEvents(strings.NewReader(input))
	if err != nil {
		t.Fatalf("LoadEvents returned error: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if want := time.Date(2026, time.April, 19, 11, 16, 3, 0, time.UTC); !events[0].Time.Equal(want) {
		t.Fatalf("expected first timestamp %s, got %s", want, events[0].Time)
	}
	if events[1].Message != "Found regressions" {
		t.Fatalf("unexpected second message %q", events[1].Message)
	}
	if events[1].Channel != "research" {
		t.Fatalf("expected second event channel research, got %q", events[1].Channel)
	}
}

func TestSampleSnapshotHasExpectedSummary(t *testing.T) {
	t.Parallel()

	snapshot, err := SampleSnapshot()
	if err != nil {
		t.Fatalf("SampleSnapshot returned error: %v", err)
	}

	summary := snapshot.Summary()
	if summary.TotalAgents != 4 {
		t.Fatalf("expected 4 agents, got %d", summary.TotalAgents)
	}
	if summary.RunningAgents != 2 {
		t.Fatalf("expected 2 running agents, got %d", summary.RunningAgents)
	}
	if summary.BlockedAgents != 1 {
		t.Fatalf("expected 1 blocked agent, got %d", summary.BlockedAgents)
	}
	if summary.TotalChannels != 3 {
		t.Fatalf("expected 3 channels, got %d", summary.TotalChannels)
	}
	if summary.OpenChannels != 3 {
		t.Fatalf("expected 3 open-or-active channels, got %d", summary.OpenChannels)
	}
}
