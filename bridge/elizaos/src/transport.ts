import { once } from "node:events";
import { createConnection, type Socket } from "node:net";
import type { UiEvent } from "./types";

export interface EventTransport {
  target: string;
  send(event: UiEvent): Promise<void>;
  close(): Promise<void>;
}

interface TargetDescriptor {
  kind: "stdout" | "unix" | "tcp";
  address?: string;
  display: string;
}

class StreamTransport implements EventTransport {
  constructor(
    public readonly target: string,
    private readonly stream: NodeJS.WritableStream,
    private readonly closeStream: boolean,
  ) {}

  async send(event: UiEvent): Promise<void> {
    const line = JSON.stringify(event) + "\n";
    if (this.stream.write(line)) {
      return;
    }
    await once(this.stream, "drain");
  }

  async close(): Promise<void> {
    if (!this.closeStream) {
      return;
    }

    const socket = this.stream as Socket;
    socket.end();
    await once(socket, "close");
  }
}

export function normalizeTarget(target = "stdout"): TargetDescriptor {
  const value = target.trim();

  if (value === "" || value === "-" || value === "stdout") {
    return { kind: "stdout", display: "stdout" };
  }
  if (value.startsWith("unix://")) {
    return { kind: "unix", address: value.slice("unix://".length), display: value };
  }
  if (value.startsWith("tcp://")) {
    return { kind: "tcp", address: value.slice("tcp://".length), display: value };
  }
  if (value.includes("/")) {
    return { kind: "unix", address: value, display: `unix://${value}` };
  }
  if (value.includes(":")) {
    return { kind: "tcp", address: value, display: `tcp://${value}` };
  }

  return { kind: "unix", address: value, display: `unix://${value}` };
}

export async function createTransport(target = "stdout"): Promise<EventTransport> {
  const normalized = normalizeTarget(target);

  switch (normalized.kind) {
    case "stdout":
      return new StreamTransport(normalized.display, process.stdout, false);
    case "unix":
      return connectSocket("unix", normalized.address!, normalized.display);
    case "tcp":
      return connectSocket("tcp", normalized.address!, normalized.display);
  }
}

async function connectSocket(
  kind: "unix" | "tcp",
  address: string,
  display: string,
): Promise<EventTransport> {
  const socket =
    kind === "unix" ? createConnection(address) : createConnection(parseTcpAddress(address));

  await once(socket, "connect");
  return new StreamTransport(display, socket, true);
}

function parseTcpAddress(address: string): { host: string; port: number } {
  const separator = address.lastIndexOf(":");
  if (separator === -1) {
    throw new Error(`invalid tcp target "${address}"`);
  }

  const host = address.slice(0, separator) || "127.0.0.1";
  const port = Number(address.slice(separator + 1));
  if (!Number.isInteger(port) || port <= 0) {
    throw new Error(`invalid tcp port in "${address}"`);
  }

  return { host, port };
}
