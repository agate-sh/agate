#!/usr/bin/env tsx
import { createCanvas } from 'canvas';
import { readFileSync, writeFileSync, mkdirSync } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

// Configuration for logo generation
const config = {
  inputPath: join(__dirname, '../packages/client/pkg/gui/overlays/ascii-art.txt'),
  outputDir: join(__dirname, '../assets'),
  fontSize: 400,
  padding: 100,
  font: 'monospace',
  agateColor: '#9d87ae', // Brand color from packages/client/pkg/gui/theme/theme.go
  themes: {
    dark: {
      background: '#000000',
      text: '#9d87ae',
      filename: 'logo-dark.png'
    },
    light: {
      background: '#FFFFFF',
      text: '#9d87ae',
      filename: 'logo-light.png'
    }
  }
};

function generateLogo(asciiArt: string, theme: { background: string; text: string }, outputPath: string): void {
  const lines = asciiArt.split('\n').filter(line => line.trim().length > 0);

  // Calculate dimensions based on the longest line
  const maxLineLength = Math.max(...lines.map(line => line.length));
  const width = (maxLineLength * config.fontSize * 0.6) + (config.padding * 2);
  const height = (lines.length * config.fontSize * 1.2) + (config.padding * 2);

  // Create canvas
  const canvas = createCanvas(width, height);
  const ctx = canvas.getContext('2d');

  // Set background
  ctx.fillStyle = theme.background;
  ctx.fillRect(0, 0, width, height);

  // Set text style
  ctx.fillStyle = theme.text;
  ctx.font = `${config.fontSize}px ${config.font}`;
  ctx.textBaseline = 'top';

  // Draw each line of ASCII art
  lines.forEach((line, index) => {
    const y = config.padding + (index * config.fontSize * 1.2);
    ctx.fillText(line, config.padding, y);
  });

  // Write to file
  const buffer = canvas.toBuffer('image/png');
  writeFileSync(outputPath, buffer);
  console.log(`✓ Generated ${outputPath}`);
}

function main(): void {
  try {
    // Read ASCII art
    const asciiArt = readFileSync(config.inputPath, 'utf-8');

    // Ensure output directory exists
    mkdirSync(config.outputDir, { recursive: true });

    // Generate logos for each theme
    Object.entries(config.themes).forEach(([themeName, theme]) => {
      const outputPath = join(config.outputDir, theme.filename);
      generateLogo(asciiArt, theme, outputPath);
    });

    console.log('\n✓ Logo generation complete!');
  } catch (error) {
    console.error('Error generating logos:', error);
    process.exit(1);
  }
}

main();
