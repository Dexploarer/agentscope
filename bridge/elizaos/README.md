# AgentScope ElizaOS Bridge

Local bridge package for streaming ElizaOS runtime events into AgentScope.

## Install In An Eliza Project

Build the package from the AgentScope repo root:

```bash
bun run build:bridge
```

Then, from your Eliza project:

```bash
bun add /Users/home/Documents/Codex/2026-04-19-https-github-com-charmbracelet-lipgloss-id/dist/elizaos-bridge
```

If you need distributable archives instead of a local directory install, run:

```bash
bun run build:release
```

That emits `/Users/home/Documents/Codex/2026-04-19-https-github-com-charmbracelet-lipgloss-id/dist/release/agentscope-elizaos-bridge_0.1.0.tar.gz`.

## Minimal Usage

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

## Safe Auth Model

Use the AgentScope CLI for local-user auth bootstrapping:

- `agentscope connect-eliza -prompt`
- `agentscope connect-eliza -api-key ...`
- `agentscope connect-eliza -use-keychain`
- `agentscope connect-eliza -cloud-login`

This package does not extract cookies, scrape sessions, or harvest secrets from browsers or apps.
