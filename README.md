# AgentScope

AgentScope is an event-driven operator console for agent runtimes.

The Go side uses [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Bubbles](https://github.com/charmbracelet/bubbles), and [Lip Gloss](https://github.com/charmbracelet/lipgloss) to render a control room with:

- channel panes
- per-channel event history
- a focused detail viewport
- live updates from `stdin`, files, Unix sockets, or TCP sockets

The TypeScript side ships with a Bun bridge skeleton for ElizaOS-style runtimes so the agent runtime can stay the source of truth and the console can stay renderer-only.

## Quick Start

Build the host binary and packaged Eliza bridge:

```bash
bun run build
```

Build release archives and checksums:

```bash
bun run build:release
```

Static sample console:

```bash
go run ./cmd/agentscope
```

Render the embedded sample feed as plain events:

```bash
go run ./cmd/agentscope sample-events | go run ./cmd/agentscope events
```

Run the live console from a piped NDJSON stream:

```bash
go run ./cmd/agentscope sample-events | go run ./cmd/agentscope monitor -once
```

Run the live console in socket mode:

```bash
go run ./cmd/agentscope monitor -listen unix:///tmp/agentscope.sock
cd bridge/elizaos && bun run mock unix:///tmp/agentscope.sock
```

## Commands

```bash
go run ./cmd/agentscope
go run ./cmd/agentscope bootstrap-eliza -use-keychain -listen unix:///tmp/agentscope.sock
go run ./cmd/agentscope bootstrap-eliza -prompt -save-keychain -print-only
go run ./cmd/agentscope connect-eliza -server-url http://localhost:3000 -prompt
go run ./cmd/agentscope connect-eliza -server-url http://localhost:3000 -prompt -save-keychain
go run ./cmd/agentscope connect-eliza -server-url http://localhost:3000 -use-keychain
go run ./cmd/agentscope connect-eliza -cloud-login
go run ./cmd/agentscope doctor-eliza -skip-ping
go run ./cmd/agentscope doctor-eliza -server-url http://localhost:3000 -use-keychain
bun run build
bun run build:cli
bun run build:bridge
bun run build:release
go run ./cmd/agentscope dashboard -data ./snapshot.json
go run ./cmd/agentscope events -file ./events.ndjson
go run ./cmd/agentscope monitor -file ./events.ndjson -once
go run ./cmd/agentscope monitor -listen unix:///tmp/agentscope.sock
go run ./cmd/agentscope sample-events
./dist/bin/agentscope version
```

## Data Contract

`dashboard` reads a snapshot JSON document:

```json
{
  "workspace": "solana-agents-prod",
  "updatedAt": "2026-04-19T12:00:00Z",
  "queue": { "pending": 1, "running": 2, "failed": 0 },
  "agents": [],
  "channels": [],
  "events": []
}
```

Live modes consume NDJSON events:

```json
{"time":"2026-04-19T12:00:00Z","agent":"router","channel":"intake","kind":"channel_open","status":"open","topic":"New work routing","members":["router","planner"],"message":"Opened intake channel"}
```

## Event-Driven Channel Model

Channel lifecycle is part of the stream:

- `channel_open`: create or reopen a channel
- `channel_update`: mutate topic, members, or status without losing history
- `channel_close`: mark the channel closed while keeping it visible
- `message`: append a human-readable runtime message to the channel
- `chunk`: append streamed output to the event log
- `action_started` / `action_completed`: show tool or workflow execution
- `blocked` / `error`: surface operator attention states directly in the channel

## TypeScript Bridge

The Bun package under [bridge/elizaos/package.json](/Users/home/Documents/Codex/2026-04-19-https-github-com-charmbracelet-lipgloss-id/bridge/elizaos/package.json) exports:

- `createAgentScopePublisher(target)`
- `createElizaOSBridge(options, publisher)`
- `createAgentScopePlugin(options, publisher)`
- normalization helpers for room lifecycle, messages, chunks, actions, blocked states, and errors

That gives you a thin boundary where ElizaOS runtime callbacks can be translated once into AgentScope events.

Local install from an Eliza project:

```bash
bun run build:bridge
bun add /Users/home/Documents/Codex/2026-04-19-https-github-com-charmbracelet-lipgloss-id/dist/elizaos-bridge
```

Release archives land under `dist/release`:

- `agentscope_0.1.0_darwin_arm64.tar.gz`
- `agentscope_0.1.0_darwin_amd64.tar.gz`
- `agentscope_0.1.0_linux_arm64.tar.gz`
- `agentscope_0.1.0_linux_amd64.tar.gz`
- `agentscope-elizaos-bridge_0.1.0.tar.gz`
- `SHA256SUMS`

GitHub Actions now mirrors the local flow:

- `.github/workflows/ci.yml` runs `go test ./...`, `bun test`, and `bun run build` on pushes to `main` and on pull requests
- `.github/workflows/release.yml` runs on `v*` tags or manual dispatch, verifies the tag matches `bridge/elizaos/package.json`, builds `dist/release`, uploads the artifacts, and publishes them to a GitHub release

Minimal plugin wiring:

```ts
import { EventType } from "@elizaos/core";
import {
  createAgentScopePlugin,
  createAgentScopePublisher,
} from "@agentscope/elizaos-bridge";

const publisher = await createAgentScopePublisher("unix:///tmp/agentscope.sock");

export const agentscopePlugin = createAgentScopePlugin(
  {
    source: "elizaos",
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
```

See [ARCHITECTURE.md](/Users/home/Documents/Codex/2026-04-19-https-github-com-charmbracelet-lipgloss-id/ARCHITECTURE.md) for the transport and reducer layout.

## Eliza Auth

Safe supported flows:

- `agentscope doctor-eliza ...` to verify the local prerequisites before you bootstrap: `elizaos` CLI, bridge package, socket target, and any noninteractive auth source you selected
- `agentscope bootstrap-eliza ...` to validate Eliza auth, print the exact plugin/socket config, and launch `monitor -listen ...`
- `bun run build` to produce `./dist/bin/agentscope` and `./dist/elizaos-bridge`; `bootstrap-eliza` prefers the built bridge package automatically when it exists
- `bun run build:release` to produce cross-platform CLI tarballs, a versioned bridge archive, and `./dist/release/SHA256SUMS`
- `agentscope connect-eliza -prompt` to enter the local server `X-API-KEY` manually
- `agentscope connect-eliza -prompt -save-keychain` to save that key into macOS Keychain for this local user
- `agentscope connect-eliza -use-keychain` to reuse the saved key without retyping it
- `agentscope connect-eliza -api-key ...` to pass an explicit key
- `agentscope connect-eliza -cloud-login` to run the official `elizaos login` browser flow before validating connectivity
- environment variables such as `ELIZA_SERVER_URL`, `ELIZA_API_KEY`, `ELIZA_SERVER_AUTH_TOKEN`, `ELIZAOS_API_KEY`, and `ELIZAOS_CLOUD_API_KEY`

Not supported:

- extracting browser cookies
- scraping session tokens from local apps
- pulling API keys out of other tools without explicit user input or the provider's official auth flow
