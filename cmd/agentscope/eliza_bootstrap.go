package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var (
	connectElizaFlow = connectEliza
	runMonitorFlow   = func(args []string, stdout io.Writer, stdin io.Reader) error {
		return runConsole(args, stdout, stdin)
	}
)

func runBootstrapEliza(args []string, stdout, stderr io.Writer, stdin io.Reader) error {
	fs := flag.NewFlagSet("bootstrap-eliza", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	auth := defaultElizaAuthConfig()
	bindElizaAuthFlags(fs, &auth)

	listenTarget := "unix:///tmp/agentscope.sock"
	workspace := "eliza-live"
	source := "elizaos"
	width := 0
	height := 0
	once := false
	printOnly := false

	fs.StringVar(&listenTarget, "listen", listenTarget, "socket target used by the AgentScope monitor")
	fs.StringVar(&workspace, "workspace", workspace, "workspace name for the live monitor")
	fs.StringVar(&source, "source", source, "source label used in the AgentScope Eliza plugin")
	fs.IntVar(&width, "width", width, "monitor width; defaults to $COLUMNS or 140")
	fs.IntVar(&height, "height", height, "monitor height; defaults to $LINES or 40")
	fs.BoolVar(&once, "once", once, "quit automatically when a finite live stream ends")
	fs.BoolVar(&printOnly, "print-only", printOnly, "print plugin and monitor settings without starting the monitor")

	if err := fs.Parse(args); err != nil {
		return err
	}

	result, err := connectElizaFlow(auth, stderr, stderr, stdin)
	if err != nil {
		return err
	}

	printElizaConnectionSummary(stderr, result)
	printBootstrapSummary(stderr, result, listenTarget, workspace, source)

	if printOnly {
		return nil
	}

	return runMonitorFlow(buildBootstrapMonitorArgs(listenTarget, workspace, width, height, once), stdout, stdin)
}

func buildBootstrapMonitorArgs(listenTarget, workspace string, width, height int, once bool) []string {
	args := []string{
		"-listen", strings.TrimSpace(listenTarget),
		"-workspace", strings.TrimSpace(workspace),
	}
	if width > 0 {
		args = append(args, "-width", fmt.Sprintf("%d", width))
	}
	if height > 0 {
		args = append(args, "-height", fmt.Sprintf("%d", height))
	}
	if once {
		args = append(args, "-once")
	}
	return args
}

func printBootstrapSummary(w io.Writer, result elizaConnectionResult, listenTarget, workspace, source string) {
	bridgePath := preferredBridgePackagePath(projectRoot())

	fmt.Fprintln(w)
	fmt.Fprintln(w, "AgentScope Bootstrap")
	fmt.Fprintf(w, "Monitor Socket: %s\n", strings.TrimSpace(listenTarget))
	fmt.Fprintf(w, "Workspace: %s\n", strings.TrimSpace(workspace))
	fmt.Fprintf(w, "Plugin Source: %s\n", strings.TrimSpace(source))
	fmt.Fprintf(w, "Bridge Package: %s\n", bridgePath)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Install Command")
	fmt.Fprintf(w, "bun add %s\n", bridgePath)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Eliza Plugin Snippet")
	fmt.Fprintln(w, renderElizaPluginSnippet(strings.TrimSpace(listenTarget), strings.TrimSpace(source)))
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Monitor Command")
	fmt.Fprintf(w, "agentscope monitor %s\n", strings.Join(buildBootstrapMonitorArgs(listenTarget, workspace, 0, 0, false), " "))
}

func preferredBridgePackagePath(root string) string {
	distPath := filepath.Join(root, "dist", "elizaos-bridge")
	if info, err := statPath(filepath.Join(distPath, "package.json")); err == nil && !info.IsDir() {
		return distPath
	}

	return filepath.Join(root, "bridge", "elizaos")
}

func renderElizaPluginSnippet(target, source string) string {
	return strings.TrimSpace(fmt.Sprintf(`
import { EventType } from "@elizaos/core";
import {
  createAgentScopePlugin,
  createAgentScopePublisher,
} from "@agentscope/elizaos-bridge";

const publisher = await createAgentScopePublisher(%q);

export const agentscopePlugin = createAgentScopePlugin(
  {
    source: %q,
    eventNames: {
      roomJoined: EventType.ROOM_JOINED,
      roomLeft: EventType.ROOM_LEFT,
      roomUpdated: EventType.ROOM_UPDATED,
      messageReceived: EventType.MESSAGE_RECEIVED,
      messageSent: EventType.MESSAGE_SENT,
      actionStarted: EventType.ACTION_STARTED,
      actionCompleted: EventType.ACTION_COMPLETED,
      actionFailed: EventType.ACTION_FAILED,
      runStarted: EventType.RUN_STARTED,
      runCompleted: EventType.RUN_COMPLETED,
      runFailed: EventType.RUN_FAILED,
      runTimeout: EventType.RUN_TIMEOUT,
    },
  },
  publisher,
);
`, target, source))
}

func projectRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}
