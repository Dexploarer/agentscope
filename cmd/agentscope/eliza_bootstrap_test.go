package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildBootstrapMonitorArgs(t *testing.T) {
	t.Parallel()

	args := buildBootstrapMonitorArgs("unix:///tmp/agentscope.sock", "eliza-live", 144, 44, true)
	want := []string{
		"-listen", "unix:///tmp/agentscope.sock",
		"-workspace", "eliza-live",
		"-width", "144",
		"-height", "44",
		"-once",
	}

	if strings.Join(args, " ") != strings.Join(want, " ") {
		t.Fatalf("expected %v, got %v", want, args)
	}
}

func TestRenderElizaPluginSnippetUsesCurrentEventNames(t *testing.T) {
	t.Parallel()

	snippet := renderElizaPluginSnippet("unix:///tmp/agentscope.sock", "elizaos")
	for _, expected := range []string{
		`createAgentScopePublisher("unix:///tmp/agentscope.sock")`,
		`source: "elizaos"`,
		"EventType.ROOM_UPDATED",
		"EventType.ACTION_FAILED",
		"EventType.RUN_COMPLETED",
		"EventType.RUN_FAILED",
	} {
		if !strings.Contains(snippet, expected) {
			t.Fatalf("expected snippet to contain %q, got %q", expected, snippet)
		}
	}
}

func TestPreferredBridgePackagePathPrefersDistBuild(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	distPath := filepath.Join(root, "dist", "elizaos-bridge")
	if err := os.MkdirAll(distPath, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(distPath, "package.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	if got := preferredBridgePackagePath(root); got != distPath {
		t.Fatalf("expected %q, got %q", distPath, got)
	}
}

func TestPreferredBridgePackagePathFallsBackToSourceTree(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	want := filepath.Join(root, "bridge", "elizaos")

	if got := preferredBridgePackagePath(root); got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestRunBootstrapElizaPrintOnly(t *testing.T) {
	t.Parallel()

	originalConnect := connectElizaFlow
	originalMonitor := runMonitorFlow
	defer func() {
		connectElizaFlow = originalConnect
		runMonitorFlow = originalMonitor
	}()

	connectElizaFlow = func(cfg elizaAuthConfig, stdout, stderr io.Writer, stdin io.Reader) (elizaConnectionResult, error) {
		return elizaConnectionResult{
			Config:         cfg,
			ResolvedAPIKey: "token",
			Health:         "ok",
		}, nil
	}
	runMonitorFlow = func(args []string, stdout io.Writer, stdin io.Reader) error {
		t.Fatal("runMonitorFlow should not be called in print-only mode")
		return nil
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runBootstrapEliza([]string{
		"-listen", "unix:///tmp/agentscope.sock",
		"-print-only",
	}, &stdout, &stderr, strings.NewReader(""))
	if err != nil {
		t.Fatalf("runBootstrapEliza returned error: %v", err)
	}

	output := stderr.String()
	for _, expected := range []string{
		"Eliza Connection",
		"Health: ok",
		"AgentScope Bootstrap",
		"Monitor Socket: unix:///tmp/agentscope.sock",
		"Bridge Package:",
		"bun add ",
		"EventType.ROOM_UPDATED",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected output to contain %q, got %q", expected, output)
		}
	}
}
