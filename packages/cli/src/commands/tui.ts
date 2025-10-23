import { execa } from "execa";
import chalk from "chalk";
import { createRequire } from "module";
import { ensureServer, getServerUrl } from "../bootstrap.js";
import { initClient, validateGitRepo } from "../api.js";

const require = createRequire(import.meta.url);
const CLIENT_ENTRY = require.resolve("@agate/client/dist/index.js");

async function resolveClientLauncher(): Promise<{
  command: string;
  args: string[];
}> {
  const explicitBin = process.env.AGATE_CLIENT_BIN;
  if (explicitBin && explicitBin.trim().length > 0) {
    return {
      command: explicitBin,
      args: [CLIENT_ENTRY],
    };
  }

  try {
    await execa("bun", ["--version"], { stdio: "ignore" });
    return {
      command: "bun",
      args: [CLIENT_ENTRY],
    };
  } catch {
    throw new Error(
      "Bun runtime not found. Install Bun from https://bun.sh or set AGATE_CLIENT_BIN to the executable that launches the Agate client."
    );
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

    const launcher = await resolveClientLauncher();
    await execa(launcher.command, launcher.args, {
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
