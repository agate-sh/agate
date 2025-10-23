import chalk from "chalk";
import { execa } from "execa";
import { mkdir, readFile } from "fs/promises";
import { dirname, join } from "path";
import { fileURLToPath } from "url";
import { ensureServer, getServerUrl } from "../bootstrap.js";
import { initClient, validateGitRepo } from "../api.js";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const CLIENT_BINARY_NAME = process.platform === "win32" ? "agate-client.exe" : "agate-client";

type ClientPaths = {
  rootDir: string;
  clientDir: string;
  distDir: string;
  binaryPath: string;
};

async function findWorkspaceRoot(): Promise<string> {
  let current = __dirname;
  while (true) {
    const pkgPath = join(current, "package.json");
    try {
      const pkgRaw = await readFile(pkgPath, "utf8");
      const pkg = JSON.parse(pkgRaw) as { name?: string };
      if (pkg.name === "@agate/root") {
        return current;
      }
    } catch (error) {
      // ignore, just walk up
    }

    const parent = dirname(current);
    if (parent === current) {
      throw new Error("Workspace root not found");
    }
    current = parent;
  }
}

async function getClientPaths(): Promise<ClientPaths> {
  const rootDir =
    process.env.AGATE_ROOT_DIR && process.env.AGATE_ROOT_DIR.trim().length > 0
      ? process.env.AGATE_ROOT_DIR
      : await findWorkspaceRoot();
  const clientDir = join(rootDir, "packages", "client");
  const distDir = join(clientDir, "dist");
  const binaryPath = join(distDir, CLIENT_BINARY_NAME);
  return { rootDir, clientDir, distDir, binaryPath };
}

async function buildGoClient(): Promise<string> {
  const paths = await getClientPaths();
  await mkdir(paths.distDir, { recursive: true });
  await execa("go", ["build", "-o", paths.binaryPath, "./cmd/agate"], {
    cwd: paths.clientDir,
    stdio: "inherit",
  });
  return paths.binaryPath;
}

async function resolveClientExecutable(): Promise<string> {
  const explicitBin = process.env.AGATE_CLIENT_BIN;
  if (explicitBin && explicitBin.trim().length > 0) {
    return explicitBin.trim();
  }

  try {
    return await buildGoClient();
  } catch (error) {
    throw new Error(`Failed to build Go client: ${(error as Error).message}`);
  }
}

/**
 * Launch the Agate TUI
 */
export async function tuiCommand(): Promise<void> {
  try {
    // Validate we're in a git repo
    const cwd = process.cwd();

    // Ensure server is running
    await ensureServer();
    initClient();
    await validateGitRepo(cwd);

    const executable = await resolveClientExecutable();
    await execa(executable, [], {
      stdio: "inherit",
      env: {
        ...process.env,
        AGATE_SERVER_URL: getServerUrl(),
      },
    });
  } catch (error) {
    if (error instanceof Error) {
      console.error(chalk.red(`Error: ${error.message}`));
    } else {
      console.error(chalk.red("An unknown error occurred"));
    }
    process.exit(1);
  }
}
