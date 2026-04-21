package ui

import (
	"strings"
	"testing"
	"time"

	"agentscope/internal/agent"
)

func TestRenderDashboardIncludesCoreSections(t *testing.T) {
	t.Parallel()

	snapshot, err := agent.SampleSnapshot()
	if err != nil {
		t.Fatalf("SampleSnapshot returned error: %v", err)
	}

	rendered := RenderDashboard(snapshot, 120)

	for _, expected := range []string{
		"AgentScope",
		"Workspace:",
		"solana-agents-prod",
		"Agents",
		"Channels",
		"#intake",
		"#research",
		"Global Feed",
		"router",
		"researcher",
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("expected rendered dashboard to contain %q", expected)
		}
	}
}

func TestRenderEventIncludesAgentKindAndMessage(t *testing.T) {
	t.Parallel()

	event := agent.Event{
		Time:    time.Date(2026, time.April, 19, 11, 19, 21, 0, time.UTC),
		Agent:   "executor",
		Channel: "deploy",
		Kind:    "blocked",
		Message: "Waiting on staging credentials",
	}

	rendered := RenderEvent(event)

	for _, expected := range []string{"11:19:21", "executor", "#deploy", "BLOCKED", "Waiting on staging credentials"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("expected rendered event to contain %q", expected)
		}
	}
}
