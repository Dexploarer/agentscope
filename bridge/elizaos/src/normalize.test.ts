import { describe, expect, it } from "bun:test";
import {
  normalizeActionCompleted,
  normalizeMessageReceived,
  normalizeRoomOpened,
} from "./normalize";
import { normalizeTarget } from "./transport";

describe("normalizeRoomOpened", () => {
  it("maps room lifecycle into a channel_open event", () => {
    const event = normalizeRoomOpened(
      { agent: "router", worldId: "world-1", source: "elizaos" },
      {
        id: "room-intake",
        name: "intake",
        topic: "New work routing",
        members: ["router", "planner"],
        status: "open",
      },
      "2026-04-19T12:00:00Z",
    );

    expect(event).toEqual({
      time: "2026-04-19T12:00:00Z",
      agent: "router",
      channel: "intake",
      kind: "channel_open",
      status: "open",
      topic: "New work routing",
      members: ["router", "planner"],
      message: "Opened intake channel",
      source: "elizaos",
      roomId: "room-intake",
      worldId: "world-1",
    });
  });
});

describe("normalizeMessageReceived", () => {
  it("keeps room identity stable while surfacing readable text", () => {
    const event = normalizeMessageReceived(
      { agent: "planner", source: "elizaos" },
      { id: "room-intake", name: "intake" },
      { roomId: "room-intake", text: "Queued follow-up analysis" },
      "2026-04-19T12:00:01Z",
    );

    expect(event.kind).toBe("message");
    expect(event.channel).toBe("intake");
    expect(event.roomId).toBe("room-intake");
    expect(event.message).toBe("Queued follow-up analysis");
  });
});

describe("normalizeActionCompleted", () => {
  it("attaches structured payloads to completed actions", () => {
    const event = normalizeActionCompleted(
      { agent: "executor" },
      { id: "room-deploy", name: "deploy" },
      { name: "deploy-check", result: { ok: true } },
      "run-42",
      "2026-04-19T12:00:02Z",
    );

    expect(event.kind).toBe("action_completed");
    expect(event.runId).toBe("run-42");
    expect(event.data).toEqual({ ok: true });
  });
});

describe("normalizeTarget", () => {
  it("supports stdout, unix sockets, and tcp addresses", () => {
    expect(normalizeTarget("stdout")).toEqual({ kind: "stdout", display: "stdout" });
    expect(normalizeTarget("/tmp/agentscope.sock")).toEqual({
      kind: "unix",
      address: "/tmp/agentscope.sock",
      display: "unix:///tmp/agentscope.sock",
    });
    expect(normalizeTarget("127.0.0.1:7777")).toEqual({
      kind: "tcp",
      address: "127.0.0.1:7777",
      display: "tcp://127.0.0.1:7777",
    });
  });
});
