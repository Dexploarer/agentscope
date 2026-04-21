import type {
  ActionDescriptor,
  AgentContext,
  MessageDescriptor,
  RoomDescriptor,
  UiEvent,
} from "./types";

function cleanText(value?: string | null): string {
  return (value ?? "").replace(/\s+/g, " ").trim();
}

function optionalText(value?: string | null): string | undefined {
  const cleaned = cleanText(value);
  return cleaned === "" ? undefined : cleaned;
}

function cleanMembers(members?: string[]): string[] | undefined {
  if (!members) {
    return undefined;
  }

  const seen = new Set<string>();
  const compacted: string[] = [];

  for (const member of members) {
    const cleaned = cleanText(member);
    if (cleaned === "" || seen.has(cleaned)) {
      continue;
    }
    seen.add(cleaned);
    compacted.push(cleaned);
  }

  return compacted.length > 0 ? compacted : undefined;
}

function channelName(room: RoomDescriptor): string {
  return cleanText(room.name) || cleanText(room.id);
}

function baseEvent(
  context: AgentContext,
  room: RoomDescriptor | undefined,
  kind: UiEvent["kind"],
  message: string,
  at: string,
): UiEvent {
  return {
    time: at,
    agent: cleanText(context.agent),
    channel: room ? optionalText(channelName(room)) : undefined,
    kind,
    message: cleanText(message),
    source: optionalText(context.source) ?? "elizaos",
    roomId: room ? optionalText(room.id) : undefined,
    worldId: optionalText(context.worldId),
  };
}

export function normalizeRoomOpened(
  context: AgentContext,
  room: RoomDescriptor,
  at = new Date().toISOString(),
): UiEvent {
  return {
    ...baseEvent(context, room, "channel_open", `Opened ${channelName(room)} channel`, at),
    status: room.status ?? "open",
    topic: optionalText(room.topic),
    members: cleanMembers(room.members),
  };
}

export function normalizeRoomUpdated(
  context: AgentContext,
  room: RoomDescriptor,
  message = `Updated ${channelName(room)} channel`,
  at = new Date().toISOString(),
): UiEvent {
  return {
    ...baseEvent(context, room, "channel_update", message, at),
    status: room.status,
    topic: optionalText(room.topic),
    members: cleanMembers(room.members),
  };
}

export function normalizeRoomClosed(
  context: AgentContext,
  room: RoomDescriptor,
  message = `Closed ${channelName(room)} channel`,
  at = new Date().toISOString(),
): UiEvent {
  return {
    ...baseEvent(context, room, "channel_close", message, at),
    status: "closed",
    topic: optionalText(room.topic),
    members: cleanMembers(room.members),
  };
}

export function normalizeMessageReceived(
  context: AgentContext,
  room: RoomDescriptor,
  message: MessageDescriptor,
  at = new Date().toISOString(),
): UiEvent {
  return {
    ...baseEvent(context, room, "message", message.text ?? "Received message", at),
    roomId: optionalText(message.roomId) ?? optionalText(room.id),
  };
}

export function normalizeStreamChunk(
  context: AgentContext,
  room: RoomDescriptor,
  chunk: string,
  runId?: string,
  at = new Date().toISOString(),
): UiEvent {
  return {
    ...baseEvent(context, room, "chunk", chunk, at),
    runId: optionalText(runId),
  };
}

export function normalizeActionStarted(
  context: AgentContext,
  room: RoomDescriptor,
  action: ActionDescriptor,
  runId?: string,
  at = new Date().toISOString(),
): UiEvent {
  return {
    ...baseEvent(context, room, "action_started", `${cleanText(action.name)} started`, at),
    runId: optionalText(runId),
  };
}

export function normalizeActionCompleted(
  context: AgentContext,
  room: RoomDescriptor,
  action: ActionDescriptor,
  runId?: string,
  at = new Date().toISOString(),
): UiEvent {
  const event: UiEvent = {
    ...baseEvent(context, room, "action_completed", `${cleanText(action.name)} completed`, at),
    runId: optionalText(runId),
  };

  if (action.result && typeof action.result === "object") {
    event.data = action.result as Record<string, unknown>;
  }

  return event;
}

export function normalizeBlocked(
  context: AgentContext,
  room: RoomDescriptor,
  reason: string,
  at = new Date().toISOString(),
): UiEvent {
  return {
    ...baseEvent(context, room, "blocked", reason, at),
    status: "blocked",
  };
}

export function normalizeError(
  context: AgentContext,
  room: RoomDescriptor | undefined,
  reason: string,
  data?: Record<string, unknown>,
  at = new Date().toISOString(),
): UiEvent {
  return {
    ...baseEvent(context, room, "error", reason, at),
    data,
  };
}
