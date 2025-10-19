import { defineConfig } from 'tsup';

export default defineConfig({
  entry: ['src/index.ts', 'src/theme.ts'],
  format: ['esm'],
  dts: true,
  clean: true,
  sourcemap: true,
  splitting: false,
  // Don't bundle dependencies - treat them as external
  noExternal: [],
  external: ['pino', 'pino-pretty'],
});
