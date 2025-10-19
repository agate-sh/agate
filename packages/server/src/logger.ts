import { createLogger, type Logger } from '@agate/shared';

export const logger: Logger = createLogger({
  component: 'server',
});
