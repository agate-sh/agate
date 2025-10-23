import chalk from 'chalk';
import { AGENTS, type AgentName } from '@agate/shared';
import { ensureServer } from '../bootstrap.js';
import { initClient, validateGitRepo, createAgentSession, sendToSession } from '../api.js';

/**
 * Validate that the agent name is valid
 */
function validateAgent(agent: string): agent is AgentName {
  return agent in AGENTS;
}

/**
 * Run an agent with a prompt in a new session
 */
export async function runCommand(agent: string, prompt: string): Promise<void> {
  try {
    // Validate agent name
    if (!validateAgent(agent)) {
      const validAgents = Object.keys(AGENTS).filter(name => name !== 'default').join(', ');
      console.error(chalk.red(`Error: Invalid agent "${agent}"`));
      console.error(chalk.gray(`Valid agents: ${validAgents}`));
      process.exit(1);
    }

    // Validate we're in a git repo
    const cwd = process.cwd();
    await ensureServer();
    initClient();
    await validateGitRepo(cwd);

    console.log(chalk.gray(`Creating session for ${AGENTS[agent].companyName}...`));

    // Create session with auto-generated branch name
    const session = await createAgentSession(cwd, agent);

    console.log(chalk.green('✓ Session created'));
    console.log(chalk.gray(`  Branch: ${chalk.bold(session.branchName)}`));
    console.log(chalk.gray(`  Worktree: ${session.worktreePath}`));

    // Inject the prompt
    console.log(chalk.gray('\nInjecting prompt...'));
    await sendToSession(session.sessionId, prompt);

    console.log(chalk.green('✓ Prompt injected'));

    // Print instructions for attaching
    console.log(chalk.cyan('\nTo attach to the session, run:'));
    console.log(chalk.bold(`  tmux attach-session -t ${session.tmuxName}`));

  } catch (error) {
    if (error instanceof Error) {
      console.error(chalk.red(`Error: ${error.message}`));
    } else {
      console.error(chalk.red('An unknown error occurred'));
    }
    process.exit(1);
  }
}
