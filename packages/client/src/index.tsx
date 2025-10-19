import { render } from '@opentui/react';
import { App } from './App';

async function main() {
  await render(
    <App />,
    {
      targetFps: 60,
      gatherStats: false,
      exitOnCtrlC: true,
      useKittyKeyboard: true,
    }
  );
}

main().catch((error) => {
  console.error('Fatal error:', error);
  process.exit(1);
});
