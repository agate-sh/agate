import yargs from 'yargs';
import { hideBin } from 'yargs/helpers';
import { tuiCommand } from './commands/tui.js';
import { runCommand } from './commands/run.js';

// Parse CLI arguments
await yargs(hideBin(process.argv))
  .scriptName('agate')
  .usage('$0 [agent] [prompt]')
  .example('$0', 'Launch the Agate TUI')
  .example('$0 claude "fix the bug in auth.ts"', 'Create a session with Claude and inject a prompt')
  .command(
    '$0 [agent] [prompt]',
    'Launch Agate TUI or run an agent with a prompt',
    (yargs) => {
      return yargs
        .positional('agent', {
          describe: 'Agent name (claude, amp, gemini, etc.)',
          type: 'string',
        })
        .positional('prompt', {
          describe: 'Prompt to inject into the agent',
          type: 'string',
        });
    },
    async (argv) => {
      // If both agent and prompt are provided, run the agent
      if (argv.agent && argv.prompt) {
        await runCommand(argv.agent, argv.prompt);
      }
      // If only agent is provided (no prompt), show error
      else if (argv.agent && !argv.prompt) {
        console.error('Error: When specifying an agent, you must also provide a prompt');
        console.error('Usage: agate <agent> "<prompt>"');
        process.exit(1);
      }
      // Otherwise launch the TUI
      else {
        await tuiCommand();
      }
    }
  )
  .help()
  .version()
  .parse();
