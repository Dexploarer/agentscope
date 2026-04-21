import { createHash } from "node:crypto";
import { mkdir, mkdtemp, rm } from "node:fs/promises";
import { join, resolve } from "node:path";
import { tmpdir } from "node:os";

import {
  bridgeReleaseBasename,
  cliReleaseBasename,
  createDistBridgePackage,
  resolveBuildSelection,
  resolveReleaseTargets,
  type ReleaseTarget,
} from "./build-lib";

const rootDir = resolve(import.meta.dir, "..");
const distDir = join(rootDir, "dist");
const cliOutputDir = join(distDir, "bin");
const cliOutputPath = join(cliOutputDir, "agentscope");
const bridgeSourceDir = join(rootDir, "bridge", "elizaos");
const bridgeDistDir = join(distDir, "elizaos-bridge");
const releaseDir = join(distDir, "release");

async function main() {
  const args = Bun.argv.slice(2);
  const selection = resolveBuildSelection(args);
  const shouldRelease = args.includes("--release");
  const version = await resolveVersion();
  const commit = await resolveCommit();
  const builtAt = new Date().toISOString();

  await mkdir(distDir, { recursive: true });

  if (selection.cli) {
    await buildCLI({
      outputPath: cliOutputPath,
      version,
      commit,
      builtAt,
    });
  }
  if (selection.bridge) {
    await buildBridge();
  }
  if (shouldRelease) {
    await buildRelease(version, commit, builtAt, args);
  }

  const builtTargets = [
    selection.cli ? `cli=${cliOutputPath}` : "",
    selection.bridge ? `bridge=${bridgeDistDir}` : "",
    shouldRelease ? `release=${releaseDir}` : "",
  ].filter(Boolean);

  console.log(`built ${builtTargets.join(" ")}`);
}

async function buildCLI(options: {
  outputPath: string;
  version: string;
  commit: string;
  builtAt: string;
  target?: ReleaseTarget;
}) {
  await mkdir(resolve(options.outputPath, ".."), { recursive: true });

  const ldflags = [
    "-X", `main.buildVersion=${options.version}`,
    "-X", `main.buildCommit=${options.commit}`,
    "-X", `main.buildTime=${options.builtAt}`,
  ];
  const env = {
    ...process.env,
    CGO_ENABLED: "0",
  } as Record<string, string | undefined>;
  if (options.target) {
    env.GOOS = options.target.goos;
    env.GOARCH = options.target.goarch;
  }

  const proc = Bun.spawn(
    ["go", "build", "-ldflags", ldflags.join(" "), "-o", options.outputPath, "./cmd/agentscope"],
    {
      cwd: rootDir,
      stdout: "inherit",
      stderr: "inherit",
      env,
    },
  );

  const exitCode = await proc.exited;
  if (exitCode !== 0) {
    throw new Error(`go build failed with exit code ${exitCode}`);
  }
}

async function buildBridge() {
  await rm(bridgeDistDir, { recursive: true, force: true });
  await mkdir(bridgeDistDir, { recursive: true });

  await buildBridgeEntrypoint(
    join(bridgeSourceDir, "src", "index.ts"),
    join(bridgeDistDir, "index.js"),
    "bridge index",
  );
  await buildBridgeEntrypoint(
    join(bridgeSourceDir, "src", "mock.ts"),
    join(bridgeDistDir, "mock.js"),
    "bridge mock",
  );

  const sourcePackage = JSON.parse(
    await Bun.file(join(bridgeSourceDir, "package.json")).text(),
  );
  const distPackage = createDistBridgePackage(sourcePackage);

  await Bun.write(
    join(bridgeDistDir, "package.json"),
    `${JSON.stringify(distPackage, null, 2)}\n`,
  );
  await Bun.write(
    join(bridgeDistDir, "README.md"),
    await Bun.file(join(bridgeSourceDir, "README.md")).text(),
  );
}

async function buildRelease(
  version: string,
  commit: string,
  builtAt: string,
  args: string[],
) {
  await rm(releaseDir, { recursive: true, force: true });
  await mkdir(releaseDir, { recursive: true });

  const checksumLines: string[] = [];
  const targets = resolveReleaseTargets(args);
  for (const target of targets) {
    const basename = cliReleaseBasename(version, target);
    const stagingDir = await mkdtemp(join(tmpdir(), `${basename}-`));
    try {
      const binaryName = target.goos === "windows" ? "agentscope.exe" : "agentscope";
      const binaryPath = join(stagingDir, binaryName);
      await buildCLI({
        outputPath: binaryPath,
        version,
        commit,
        builtAt,
        target,
      });
      await Bun.write(join(stagingDir, "README.md"), await Bun.file(join(rootDir, "README.md")).text());

      const archivePath = join(releaseDir, `${basename}.tar.gz`);
      await createTarArchive(stagingDir, archivePath);
      checksumLines.push(await checksumLine(archivePath));
    } finally {
      await rm(stagingDir, { recursive: true, force: true });
    }
  }

  const bridgeArchivePath = join(
    releaseDir,
    `${bridgeReleaseBasename(version)}.tar.gz`,
  );
  await createTarArchive(bridgeDistDir, bridgeArchivePath);
  checksumLines.push(await checksumLine(bridgeArchivePath));
  await Bun.write(join(releaseDir, "SHA256SUMS"), `${checksumLines.join("\n")}\n`);
}

async function buildBridgeEntrypoint(
	entrypoint: string,
	outfile: string,
	label: string,
) {
	const proc = Bun.spawn(
		[
			process.execPath,
			"build",
			entrypoint,
			"--target",
			"bun",
			"--format",
			"esm",
			"--outfile",
			outfile,
		],
		{
			cwd: rootDir,
			stdout: "pipe",
			stderr: "pipe",
		},
	);

	const exitCode = await proc.exited;
	if (exitCode === 0) {
		return;
	}

	const errorOutput = (await new Response(proc.stderr).text()).trim();
	const detail = errorOutput || `exit code ${exitCode}`;
	throw new Error(`${label} build failed: ${detail}`);
}

async function createTarArchive(sourceDir: string, archivePath: string) {
  await mkdir(resolve(archivePath, ".."), { recursive: true });

  const proc = Bun.spawn(
    ["tar", "-czf", archivePath, "-C", sourceDir, "."],
    {
      cwd: rootDir,
      stdout: "pipe",
      stderr: "pipe",
    },
  );

  const exitCode = await proc.exited;
  if (exitCode === 0) {
    return;
  }

  const errorOutput = (await new Response(proc.stderr).text()).trim();
  const detail = errorOutput || `exit code ${exitCode}`;
  throw new Error(`archive creation failed: ${detail}`);
}

async function checksumLine(path: string): Promise<string> {
  const hash = createHash("sha256");
  hash.update(Buffer.from(await Bun.file(path).arrayBuffer()));
  return `${hash.digest("hex")}  ${path.split("/").at(-1) ?? path}`;
}

async function resolveVersion(): Promise<string> {
  const sourcePackage = JSON.parse(
    await Bun.file(join(bridgeSourceDir, "package.json")).text(),
  ) as { version?: string };
  return sourcePackage.version?.trim() || "0.1.0";
}

async function resolveCommit(): Promise<string> {
  const proc = Bun.spawn(
    ["git", "rev-parse", "--short", "HEAD"],
    {
      cwd: rootDir,
      stdout: "pipe",
      stderr: "pipe",
    },
  );
  const exitCode = await proc.exited;
  if (exitCode !== 0) {
    return "uncommitted";
  }

  const commit = (await new Response(proc.stdout).text()).trim();
  return commit || "uncommitted";
}

await main();
