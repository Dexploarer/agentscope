export interface BuildSelection {
  cli: boolean;
  bridge: boolean;
}

export interface ReleaseTarget {
  goos: string;
  goarch: string;
}

export interface SourceBridgePackage {
  name: string;
  version: string;
  description?: string;
  private?: boolean;
}

export interface DistBridgePackage {
  name: string;
  version: string;
  description?: string;
  private?: boolean;
  type: "module";
  main: "./index.js";
  exports: {
    ".": "./index.js";
    "./mock": "./mock.js";
  };
  files: string[];
}

export function resolveBuildSelection(args: string[]): BuildSelection {
  const cli = args.includes("--cli");
  const bridge = args.includes("--bridge");

  if (!cli && !bridge) {
    return { cli: true, bridge: true };
  }

  return { cli, bridge };
}

export function supportedReleaseTargets(): ReleaseTarget[] {
  return [
    { goos: "darwin", goarch: "arm64" },
    { goos: "darwin", goarch: "amd64" },
    { goos: "linux", goarch: "arm64" },
    { goos: "linux", goarch: "amd64" },
  ];
}

export function resolveReleaseTargets(args: string[]): ReleaseTarget[] {
  const requested = collectFlagValues(args, "--target");
  if (requested.length === 0) {
    return supportedReleaseTargets();
  }

  return requested.map(parseReleaseTarget);
}

export function createDistBridgePackage(
  source: SourceBridgePackage,
): DistBridgePackage {
  return {
    name: source.name,
    version: source.version,
    description: source.description,
    private: source.private,
    type: "module",
    main: "./index.js",
    exports: {
      ".": "./index.js",
      "./mock": "./mock.js",
    },
    files: ["index.js", "mock.js", "README.md"],
  };
}

export function parseReleaseTarget(value: string): ReleaseTarget {
  const [goos, goarch] = value.split("-", 2);
  if (!goos || !goarch) {
    throw new Error(`invalid release target ${JSON.stringify(value)}`);
  }
  return { goos, goarch };
}

export function cliReleaseBasename(
  version: string,
  target: ReleaseTarget,
): string {
  return `agentscope_${version}_${target.goos}_${target.goarch}`;
}

export function bridgeReleaseBasename(version: string): string {
  return `agentscope-elizaos-bridge_${version}`;
}

function collectFlagValues(args: string[], flagName: string): string[] {
  const values: string[] = [];

  for (let index = 0; index < args.length; index += 1) {
    const current = args[index];
    if (current === flagName && args[index+1]) {
      values.push(args[index+1]);
      index += 1;
      continue;
    }
    if (current.startsWith(`${flagName}=`)) {
      values.push(current.slice(flagName.length + 1));
    }
  }

  return values;
}
