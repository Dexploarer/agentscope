import { expect, test } from "bun:test";

import {
  bridgeReleaseBasename,
  cliReleaseBasename,
  createDistBridgePackage,
  resolveBuildSelection,
  resolveReleaseTargets,
} from "./build-lib";

test("resolveBuildSelection builds both targets by default", () => {
  expect(resolveBuildSelection([])).toEqual({ cli: true, bridge: true });
});

test("resolveBuildSelection keeps cli-only builds narrow", () => {
  expect(resolveBuildSelection(["--cli"])).toEqual({
    cli: true,
    bridge: false,
  });
});

test("resolveBuildSelection keeps bridge-only builds narrow", () => {
  expect(resolveBuildSelection(["--bridge"])).toEqual({
    cli: false,
    bridge: true,
  });
});

test("createDistBridgePackage rewrites exports to built artifacts", () => {
  expect(
    createDistBridgePackage({
      name: "@agentscope/elizaos-bridge",
      version: "0.1.0",
      description: "AgentScope bridge for ElizaOS runtimes",
      private: true,
    }),
  ).toEqual({
    name: "@agentscope/elizaos-bridge",
    version: "0.1.0",
    description: "AgentScope bridge for ElizaOS runtimes",
    private: true,
    type: "module",
    main: "./index.js",
    exports: {
      ".": "./index.js",
      "./mock": "./mock.js",
    },
    files: ["index.js", "mock.js", "README.md"],
  });
});

test("resolveReleaseTargets defaults to the supported matrix", () => {
  expect(resolveReleaseTargets([])).toEqual([
    { goos: "darwin", goarch: "arm64" },
    { goos: "darwin", goarch: "amd64" },
    { goos: "linux", goarch: "arm64" },
    { goos: "linux", goarch: "amd64" },
  ]);
});

test("resolveReleaseTargets accepts repeated --target flags", () => {
  expect(
    resolveReleaseTargets(["--target", "darwin-arm64", "--target=linux-amd64"]),
  ).toEqual([
    { goos: "darwin", goarch: "arm64" },
    { goos: "linux", goarch: "amd64" },
  ]);
});

test("release basenames stay deterministic", () => {
  expect(cliReleaseBasename("0.1.0", { goos: "darwin", goarch: "arm64" })).toBe(
    "agentscope_0.1.0_darwin_arm64",
  );
  expect(bridgeReleaseBasename("0.1.0")).toBe(
    "agentscope-elizaos-bridge_0.1.0",
  );
});
