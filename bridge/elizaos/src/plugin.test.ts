import { describe, expect, it } from "bun:test";
import { createAgentScopePlugin, defaultElizaEventNames } from "./plugin";
import type { UiEvent } from "./types";

describe("createAgentScopePlugin", () => {
  it("registers handlers on the documented ElizaOS event names", () => {
    const plugin = createAgentScopePlugin(
      { agent: "router" },
      {
        async publish() {},
        async close() {},
      },
    );

    expect(Object.keys(plugin.events)).toEqual([
      defaultElizaEventNames.roomJoined,
      defaultElizaEventNames.roomLeft,
      defaultElizaEventNames.roomUpdated,
      defaultElizaEventNames.messageReceived,
      defaultElizaEventNames.messageSent,
      defaultElizaEventNames.actionStarted,
      defaultElizaEventNames.actionCompleted,
      defaultElizaEventNames.actionFailed,
      defaultElizaEventNames.runStarted,
      defaultElizaEventNames.runCompleted,
      defaultElizaEventNames.runFailed,
      defaultElizaEventNames.runTimeout,
    ]);
  });

  it("derives agent name from runtime payloads when the bridge option omits it", async () => {
    const published: UiEvent[] = [];
    const plugin = createAgentScopePlugin(
      {},
      {
        async publish(event) {
          published.push(event);
        },
        async close() {},
      },
    );

    await plugin.events["message:received"][0]({
      runtime: { character: { name: "planner" } },
      room: { id: "room-intake", name: "intake" },
      message: {
        id: "msg-1",
        roomId: "room-intake",
        content: { text: "Queued follow-up analysis" },
      },
      timestamp: "2026-04-19T12:00:01Z",
    });

    expect(published).toHaveLength(1);
    expect(published[0]).toEqual({
      time: "2026-04-19T12:00:01Z",
      agent: "planner",
      channel: "intake",
      kind: "message",
      message: "Queued follow-up analysis",
      source: "elizaos",
      roomId: "room-intake",
    });
  });

  it("converts action failures into error events", async () => {
    const published: UiEvent[] = [];
    const plugin = createAgentScopePlugin(
      { agent: "executor" },
      {
        async publish(event) {
          published.push(event);
        },
        async close() {},
      },
    );

    await plugin.events["action:completed"][0]({
      room: { id: "room-deploy", name: "deploy" },
      action: { name: "deploy-check" },
      error: new Error("staging credentials missing"),
      result: { runId: "run-42" },
      timestamp: "2026-04-19T12:00:02Z",
    });

    expect(published).toHaveLength(1);
    expect(published[0].kind).toBe("error");
    expect(published[0].channel).toBe("deploy");
    expect(published[0].message).toBe("staging credentials missing");
  });
});
