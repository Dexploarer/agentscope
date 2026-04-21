package main

import (
	"strings"
	"testing"
)

func TestRenderVersionIncludesBuildMetadata(t *testing.T) {
	t.Parallel()

	originalVersion := buildVersion
	originalCommit := buildCommit
	originalTime := buildTime
	defer func() {
		buildVersion = originalVersion
		buildCommit = originalCommit
		buildTime = originalTime
	}()

	buildVersion = "0.1.0"
	buildCommit = "abc1234"
	buildTime = "2026-04-21T00:00:00Z"

	rendered := renderVersion()
	for _, expected := range []string{
		"version=0.1.0",
		"commit=abc1234",
		"built=2026-04-21T00:00:00Z",
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("expected %q in %q", expected, rendered)
		}
	}
}
