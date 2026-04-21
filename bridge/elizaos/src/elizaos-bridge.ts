import {
  normalizeActionCompleted,
  normalizeActionStarted,
  normalizeBlocked,
  normalizeError,
  normalizeMessageReceived,
  normalizeRoomClosed,
  normalizeRoomOpened,
  normalizeRoomUpdated,
  normalizeStreamChunk,
} from "./normalize";
import { type EventTransport, createTransport } from "./transport";
import type {
  ActionDescriptor,
  AgentContext,
  MessageDescriptor,
  RoomDescriptor,
  UiEvent,
} from "./types";

export interface AgentScopePublisher {
  publish(event: UiEvent): Promise<void>;
  close(): Promise<void>;
}

export interface BridgeOptions extends AgentContext {
  target?: string;
}

export async function createAgentScopePublisher(
  target = "stdout",
): Promise<AgentScopePublisher> {
  const transport = await createTransport(target);
  return publisherFromTransport(transport);
}

export function publisherFromTransport(transport: EventTransport): AgentScopePublisher {
  return {
    publish(event: UiEvent) {
      return transport.send(event);
    },
    close() {
      return transport.close();
    },
  };
}

export function createElizaOSBridge(
  options: BridgeOptions,
  publisher: AgentScopePublisher,
) {
  const context: AgentContext = {
    agent: options.agent,
    worldId: options.worldId,
    source: options.source ?? "elizaos",
  };

  return {
    roomOpened(room: RoomDescriptor, at?: string) {
      return publisher.publish(normalizeRoomOpened(context, room, at));
    },
    roomUpdated(room: RoomDescriptor, note?: string, at?: string) {
      return publisher.publish(normalizeRoomUpdated(context, room, note, at));
    },
    roomClosed(room: RoomDescriptor, note?: string, at?: string) {
      return publisher.publish(normalizeRoomClosed(context, room, note, at));
    },
    messageReceived(room: RoomDescriptor, message: MessageDescriptor, at?: string) {
      return publisher.publish(normalizeMessageReceived(context, room, message, at));
    },
    streamChunk(room: RoomDescriptor, chunk: string, runId?: string, at?: string) {
      return publisher.publish(normalizeStreamChunk(context, room, chunk, runId, at));
    },
    actionStarted(room: RoomDescriptor, action: ActionDescriptor, runId?: string, at?: string) {
      return publisher.publish(normalizeActionStarted(context, room, action, runId, at));
    },
    actionCompleted(room: RoomDescriptor, action: ActionDescriptor, runId?: string, at?: string) {
      return publisher.publish(normalizeActionCompleted(context, room, action, runId, at));
    },
    blocked(room: RoomDescriptor, reason: string, at?: string) {
      return publisher.publish(normalizeBlocked(context, room, reason, at));
    },
    error(reason: string, room?: RoomDescriptor, data?: Record<string, unknown>, at?: string) {
      return publisher.publish(normalizeError(context, room, reason, data, at));
    },
  };
}
