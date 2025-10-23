import pino from 'pino';
import { join } from 'path';
import { homedir } from 'os';
import { mkdirSync } from 'fs';

const AGATE_DIR = join(homedir(), '.agate');

// Ensure .agate directory exists
try {
  mkdirSync(AGATE_DIR, { recursive: true });
} catch (err) {
  // Directory already exists or can't be created
}

export type LoggerComponent = 'server' | 'client';

export interface LoggerOptions {
  component: LoggerComponent;
  isDev?: boolean;
  disableStdout?: boolean;
}

export function createLogger(options: LoggerOptions) {
  const { component, isDev = process.env.NODE_ENV === 'development', disableStdout = false } = options;

  const logFile = isDev
    ? join(AGATE_DIR, 'dev.log') // Unified log in dev
    : join(AGATE_DIR, `${component}.log`); // Separate logs in prod

  const targets: pino.TransportTargetOptions[] = [
    // File transport (always)
    {
      target: 'pino/file',
      options: {
        destination: logFile,
        mkdir: true,
      },
      level: 'debug',
    },
  ];

  // In dev mode, also log to stdout with pretty printing (unless disabled)
  if (isDev && !disableStdout) {
    targets.push({
      target: 'pino-pretty',
      options: {
        destination: 1, // stdout
        colorize: true,
        translateTime: 'HH:MM:ss',
        ignore: 'pid,hostname',
        messageFormat: `[${component}] {msg}`,
      },
      level: 'debug',
    });
  }

  return pino({
    level: isDev ? 'debug' : 'info',
    base: {
      component,
    },
    transport: {
      targets,
    },
  });
}

// Export a type for the logger
export type Logger = ReturnType<typeof createLogger>;
