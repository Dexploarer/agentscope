export type UiEventKind =
  | "channel_open"
  | "channel_update"
  | "channel_close"
  | "message"
  | "chunk"
  | "action_started"
  | "action_completed"
  | "blocked"
  | "error";

export interface UiEvent {
  time: string;
  agent: string;
  channel?: string;
  kind: UiEventKind;
  status?: "open" | "blocked" | "closed";
  topic?: string;
  members?: string[];
  message: string;
  source?: string;
  runId?: string;
  roomId?: string;
  worldId?: string;
  data?: Record<string, unknown>;
}

export interface AgentContext {
  agent: string;
  worldId?: string;
  source?: string;
}

export interface RoomDescriptor {
  id: string;
  name?: string;
  topic?: string;
  members?: string[];
  status?: "open" | "blocked" | "closed";
}

export interface MessageDescriptor {
  id?: string;
  roomId?: string;
  text?: string;
}

export interface ActionDescriptor {
  name: string;
  result?: unknown;
}
