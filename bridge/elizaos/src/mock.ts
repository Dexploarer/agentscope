import {
  normalizeActionCompleted,
  normalizeActionStarted,
  normalizeMessageReceived,
  normalizeRoomClosed,
  normalizeRoomOpened,
  normalizeRoomUpdated,
  normalizeStreamChunk,
} from "./normalize";
import { createTransport } from "./transport";
import type { AgentContext, RoomDescriptor } from "./types";

const target = Bun.argv[2] ?? "stdout";
const transport = await createTransport(target);

const context: AgentContext = {
  agent: "router",
  worldId: "solana-agents-prod",
  source: "mock-elizaos",
};

const intake: RoomDescriptor = {
  id: "room-intake",
  name: "intake",
  topic: "New work routing",
  members: ["router", "planner"],
  status: "open",
};

const research: RoomDescriptor = {
  id: "room-research",
  name: "research",
  topic: "Validate runtime assumptions",
  members: ["researcher", "router"],
  status: "open",
};

const events = [
  normalizeRoomOpened(context, intake, "2026-04-19T12:00:00Z"),
  normalizeRoomOpened(context, research, "2026-04-19T12:00:02Z"),
  normalizeMessageReceived(
    context,
    intake,
    { roomId: intake.id, text: "Assigned bridge implementation to planner" },
    "2026-04-19T12:00:04Z",
  ),
  normalizeActionStarted(
    context,
    research,
    { name: "context-scan" },
    "run-research-001",
    "2026-04-19T12:00:06Z",
  ),
  normalizeStreamChunk(
    context,
    research,
    "Found Bubble Tea input collision when stdin carries NDJSON.",
    "run-research-001",
    "2026-04-19T12:00:07Z",
  ),
  normalizeActionCompleted(
    context,
    research,
    { name: "context-scan", result: { files: 9, findings: 1 } },
    "run-research-001",
    "2026-04-19T12:00:08Z",
  ),
  normalizeRoomUpdated(
    context,
    intake,
    "Expanded intake channel to include executor",
    "2026-04-19T12:00:10Z",
  ),
  normalizeRoomClosed(
    context,
    research,
    "Closed research channel after handoff",
    "2026-04-19T12:00:12Z",
  ),
];

try {
  for (const event of events) {
    await transport.send(event);
  }
} finally {
  await transport.close();
}
