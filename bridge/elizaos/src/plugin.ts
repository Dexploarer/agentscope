import {
  normalizeActionCompleted,
  normalizeActionStarted,
  normalizeBlocked,
  normalizeError,
  normalizeMessageReceived,
  normalizeRoomClosed,
  normalizeRoomOpened,
} from "./normalize";
import type { AgentScopePublisher, BridgeOptions } from "./elizaos-bridge";
import type { ActionDescriptor, AgentContext, MessageDescriptor, RoomDescriptor } from "./types";

export interface ElizaEventNames {
  roomJoined: string;
  roomLeft: string;
  roomUpdated: string;
  messageReceived: string;
  messageSent: string;
  actionStarted: string;
  actionCompleted: string;
  actionFailed: string;
  runStarted: string;
  runCompleted: string;
  runFailed: string;
  runTimeout: string;
}

export interface AgentScopePluginOptions extends BridgeOptions {
  name?: string;
  description?: string;
  eventNames?: Partial<ElizaEventNames>;
}

export interface RuntimeLike {
  agentId?: string;
  character?: {
    name?: string;
  };
}

export interface RoomLike {
  id?: string;
  name?: string;
  topic?: string;
  worldId?: string;
  status?: "open" | "blocked" | "closed";
  members?: unknown[];
  participants?: unknown[];
  metadata?: Record<string, unknown>;
}

export interface MessageLike {
  id?: string;
  roomId?: string;
  worldId?: string;
  text?: string;
  content?: {
    text?: string;
    [key: string]: unknown;
  };
}

export interface ActionLike {
  name?: string;
}

export interface ElizaEventPayloadLike {
  runtime?: RuntimeLike;
  room?: RoomLike;
  message?: MessageLike;
  action?: ActionLike;
  result?: unknown;
  error?: unknown;
  responses?: MessageLike[];
  timestamp?: string | number | Date;
}

export interface AgentScopePluginLike {
  name: string;
  description: string;
  events: Record<string, Array<(payload: ElizaEventPayloadLike) => Promise<void>>>;
}

export const defaultElizaEventNames: ElizaEventNames = {
  roomJoined: "room:joined",
  roomLeft: "room:left",
  roomUpdated: "room:updated",
  messageReceived: "message:received",
  messageSent: "message:sent",
  actionStarted: "action:started",
  actionCompleted: "action:completed",
  actionFailed: "action:failed",
  runStarted: "run:started",
  runCompleted: "run:completed",
  runFailed: "run:failed",
  runTimeout: "run:timeout",
};

export function createAgentScopePlugin(
  options: AgentScopePluginOptions,
  publisher: AgentScopePublisher,
): AgentScopePluginLike {
  return {
    name: options.name ?? "agentscope-bridge",
    description:
      options.description ??
      "Streams ElizaOS runtime events to the AgentScope operator console.",
    events: createElizaOSEventHandlers(options, publisher),
  };
}

export function createElizaOSEventHandlers(
  options: AgentScopePluginOptions,
  publisher: AgentScopePublisher,
): Record<string, Array<(payload: ElizaEventPayloadLike) => Promise<void>>> {
  const eventNames = { ...defaultElizaEventNames, ...options.eventNames };

  return {
    [eventNames.roomJoined]: [
      async (payload) => {
        const room = resolveRoom(payload);
        if (!room) {
          return;
        }
        await publisher.publish(normalizeRoomOpened(resolveContext(options, payload), room, eventTime(payload)));
      },
    ],
    [eventNames.roomLeft]: [
      async (payload) => {
        const room = resolveRoom(payload);
        if (!room) {
          return;
        }
        await publisher.publish(
          normalizeRoomClosed(resolveContext(options, payload), room, "Room closed", eventTime(payload)),
        );
      },
    ],
    [eventNames.roomUpdated]: [
      async (payload) => {
        const room = resolveRoom(payload);
        if (!room) {
          return;
        }
        await publisher.publish(
          normalizeRoomUpdated(resolveContext(options, payload), room, "Room updated", eventTime(payload)),
        );
      },
    ],
    [eventNames.messageReceived]: [
      async (payload) => {
        const room = resolveRoom(payload);
        const message = resolveMessage(payload);
        if (!room || !message) {
          return;
        }
        await publisher.publish(
          normalizeMessageReceived(resolveContext(options, payload), room, message, eventTime(payload)),
        );
      },
    ],
    [eventNames.messageSent]: [
      async (payload) => {
        const room = resolveRoom(payload);
        const message = resolveMessage(payload);
        if (!room || !message) {
          return;
        }
        await publisher.publish(
          normalizeMessageReceived(resolveContext(options, payload), room, message, eventTime(payload)),
        );
      },
    ],
    [eventNames.actionStarted]: [
      async (payload) => {
        const room = resolveRoom(payload);
        if (!room) {
          return;
        }
        await publisher.publish(
          normalizeActionStarted(
            resolveContext(options, payload),
            room,
            resolveAction(payload),
            resolveRunID(payload),
            eventTime(payload),
          ),
        );
      },
    ],
    [eventNames.actionCompleted]: [
      async (payload) => {
        const room = resolveRoom(payload);
        const context = resolveContext(options, payload);

        if (payload.error) {
          await publisher.publish(
            normalizeError(
              context,
              room,
              resolveErrorMessage(payload.error),
              payload.result && typeof payload.result === "object"
                ? (payload.result as Record<string, unknown>)
                : undefined,
              eventTime(payload),
            ),
          );
          return;
        }

        if (!room) {
          return;
        }

        await publisher.publish(
          normalizeActionCompleted(
            context,
            room,
            resolveAction(payload),
            resolveRunID(payload),
            eventTime(payload),
          ),
        );
      },
    ],
    [eventNames.actionFailed]: [
      async (payload) => {
        const room = resolveRoom(payload);
        const context = resolveContext(options, payload);
        await publisher.publish(
          normalizeError(
            context,
            room,
            resolveErrorMessage(payload.error),
            payload.result && typeof payload.result === "object"
              ? (payload.result as Record<string, unknown>)
              : undefined,
            eventTime(payload),
          ),
        );
      },
    ],
    [eventNames.runStarted]: [
      async (payload) => {
        const room = resolveRoom(payload);
        if (!room) {
          return;
        }
        await publisher.publish(
          normalizeActionStarted(
            resolveContext(options, payload),
            room,
            { name: "run" },
            resolveRunID(payload),
            eventTime(payload),
          ),
        );
      },
    ],
    [eventNames.runCompleted]: [
      async (payload) => {
        const room = resolveRoom(payload);
        if (!room) {
          return;
        }
        await publisher.publish(
          normalizeActionCompleted(
            resolveContext(options, payload),
            room,
            { name: "run", result: payload.result },
            resolveRunID(payload),
            eventTime(payload),
          ),
        );
      },
    ],
    [eventNames.runFailed]: [
      async (payload) => {
        const room = resolveRoom(payload);
        const context = resolveContext(options, payload);
        await publisher.publish(
          normalizeError(
            context,
            room,
            resolveErrorMessage(payload.error),
            payload.result && typeof payload.result === "object"
              ? (payload.result as Record<string, unknown>)
              : undefined,
            eventTime(payload),
          ),
        );
      },
    ],
    [eventNames.runTimeout]: [
      async (payload) => {
        const room = resolveRoom(payload);
        if (!room) {
          return;
        }
        await publisher.publish(
          normalizeBlocked(
            resolveContext(options, payload),
            room,
            "Run timed out",
            eventTime(payload),
          ),
        );
      },
    ],
  };
}

function resolveContext(
  options: AgentScopePluginOptions,
  payload: ElizaEventPayloadLike,
): AgentContext {
  return {
    agent:
      cleanText(options.agent) ||
      cleanText(payload.runtime?.character?.name) ||
      cleanText(payload.runtime?.agentId) ||
      "agent",
    worldId:
      cleanText(options.worldId) ||
      cleanText(payload.room?.worldId) ||
      cleanText(payload.message?.worldId),
    source: cleanText(options.source) || "elizaos",
  };
}

function resolveRoom(payload: ElizaEventPayloadLike): RoomDescriptor | undefined {
  const roomID = cleanText(payload.room?.id) || cleanText(payload.message?.roomId);
  if (roomID === "") {
    return undefined;
  }

  return {
    id: roomID,
    name: cleanText(payload.room?.name) || roomID,
    topic:
      cleanText(payload.room?.topic) ||
      cleanText(typeof payload.room?.metadata?.topic === "string" ? payload.room.metadata.topic : ""),
    members: resolveMembers(payload.room),
    status: payload.room?.status,
  };
}

function resolveMembers(room: RoomLike | undefined): string[] | undefined {
  if (!room) {
    return undefined;
  }

  const values = [...(room.members ?? []), ...(room.participants ?? [])];
  const members: string[] = [];
  const seen = new Set<string>();

  for (const current of values) {
    let name = "";
    if (typeof current === "string") {
      name = cleanText(current);
    } else if (current && typeof current === "object") {
      const member = current as { name?: unknown; id?: unknown };
      if (typeof member.name === "string") {
        name = cleanText(member.name);
      } else if (typeof member.id === "string") {
        name = cleanText(member.id);
      }
    }

    if (name === "" || seen.has(name)) {
      continue;
    }
    seen.add(name);
    members.push(name);
  }

  return members.length > 0 ? members : undefined;
}

function resolveMessage(payload: ElizaEventPayloadLike): MessageDescriptor | undefined {
  const text =
    cleanText(payload.message?.content?.text) ||
    cleanText(payload.message?.text) ||
    cleanText(payload.responses?.map((current) => current.content?.text || current.text).join(" "));
  const roomID = cleanText(payload.message?.roomId) || cleanText(payload.room?.id);

  if (text === "" && roomID === "") {
    return undefined;
  }

  return {
    id: cleanText(payload.message?.id),
    roomId: roomID,
    text: text || "Message event",
  };
}

function resolveAction(payload: ElizaEventPayloadLike): ActionDescriptor {
  return {
    name: cleanText(payload.action?.name) || "action",
    result: payload.result,
  };
}

function resolveRunID(payload: ElizaEventPayloadLike): string | undefined {
  const result = payload.result;
  if (result && typeof result === "object" && "runId" in result) {
    const record = result as { runId?: unknown };
    if (typeof record.runId === "string") {
      return cleanText(record.runId);
    }
  }
  return undefined;
}

function resolveErrorMessage(error: unknown): string {
  if (typeof error === "string") {
    return cleanText(error) || "Action failed";
  }
  if (error instanceof Error) {
    return cleanText(error.message) || "Action failed";
  }
  return "Action failed";
}

function eventTime(payload: ElizaEventPayloadLike): string {
  const timestamp = payload.timestamp;
  if (typeof timestamp === "string") {
    return timestamp;
  }
  if (typeof timestamp === "number") {
    return new Date(timestamp).toISOString();
  }
  if (timestamp instanceof Date) {
    return timestamp.toISOString();
  }
  return new Date().toISOString();
}

function cleanText(value: string | undefined): string {
  return (value ?? "").replace(/\s+/g, " ").trim();
}
