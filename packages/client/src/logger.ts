import { createLogger, type Logger } from '@agate/shared';

export const logger: Logger = createLogger({
  component: 'client',
  // Disable stdout for client since it's a TUI - only log to file
  disableStdout: true,
});
