import { Terminal } from '@xterm/headless';
import { RGBA, type TextChunk } from '@opentui/core';

/**
 * Represents a parsed line of terminal output with ANSI styling
 */
export interface AnsiLine {
  chunks: TextChunk[];
}

/**
 * Converts ANSI 256-color palette index to RGB hex
 * Based on xterm.js color palette
 */
function ansi256ToHex(index: number): string {
  // Standard colors (0-15)
  const standardColors = [
    '#000000', '#800000', '#008000', '#808000', '#000080', '#800080', '#008080', '#c0c0c0',
    '#808080', '#ff0000', '#00ff00', '#ffff00', '#0000ff', '#ff00ff', '#00ffff', '#ffffff',
  ];

  if (index < 16) {
    return standardColors[index] ?? '#000000';
  }

  // 216 color cube (16-231)
  if (index < 232) {
    const i = index - 16;
    const r = Math.floor(i / 36);
    const g = Math.floor((i % 36) / 6);
    const b = i % 6;

    const toHex = (v: number) => {
      const val = v === 0 ? 0 : 55 + v * 40;
      return val.toString(16).padStart(2, '0');
    };

    return `#${toHex(r)}${toHex(g)}${toHex(b)}`;
  }

  // Grayscale (232-255)
  const gray = 8 + (index - 232) * 10;
  const hex = gray.toString(16).padStart(2, '0');
  return `#${hex}${hex}${hex}`;
}

/**
 * Creates a TextChunk from xterm.js cell data
 */
function cellToTextChunk(
  text: string,
  fg: number | undefined,
  bg: number | undefined,
  attrs: number
): TextChunk {
  const chunk: TextChunk = {
    __isChunk: true,
    text,
  };

  // Extract color information
  if (fg !== undefined) {
    // fg is RGBA packed as 32-bit integer
    // For now, use xterm's default color mapping
    const hexColor = ansi256ToHex(fg & 0xFF);
    chunk.fg = RGBA.fromHex(hexColor);
  }

  if (bg !== undefined) {
    const hexColor = ansi256ToHex(bg & 0xFF);
    chunk.bg = RGBA.fromHex(hexColor);
  }

  // Extract text attributes from flags
  // These are bit flags from xterm.js
  let attributes = 0;

  // Map xterm.js attribute flags to OpenTUI TextAttributes
  // Bold: 0x01
  if (attrs & 0x01) attributes |= 1; // TextAttributes.BOLD
  // Italic: 0x04
  if (attrs & 0x04) attributes |= 2; // TextAttributes.ITALIC
  // Underline: 0x02
  if (attrs & 0x02) attributes |= 4; // TextAttributes.UNDERLINE
  // Dim: 0x08
  if (attrs & 0x08) attributes |= 16; // TextAttributes.DIM

  if (attributes > 0) {
    chunk.attributes = attributes;
  }

  return chunk;
}

/**
 * Parses raw terminal output with ANSI codes into structured lines
 * Uses a headless xterm.js terminal for proper ANSI parsing
 */
export class AnsiParser {
  private terminal: Terminal;

  constructor(cols: number = 120, rows: number = 50) {
    this.terminal = new Terminal({
      cols,
      rows,
      allowProposedApi: true,
    });
  }

  /**
   * Write data to the terminal buffer
   */
  write(data: string): void {
    this.terminal.write(data);
  }

  /**
   * Get the current terminal buffer as structured lines with styling
   */
  getLines(): AnsiLine[] {
    const lines: AnsiLine[] = [];
    const buffer = this.terminal.buffer.active;

    for (let row = 0; row < buffer.length; row++) {
      const line = buffer.getLine(row);
      if (!line) continue;

      const chunks: TextChunk[] = [];
      let currentText = '';
      let currentFg: number | undefined;
      let currentBg: number | undefined;
      let currentAttrs = 0;

      for (let col = 0; col < line.length; col++) {
        const cell = line.getCell(col);
        if (!cell) continue;

        const char = cell.getChars();
        const fg = cell.getFgColor();
        const bg = cell.getBgColor();
        const attrs = cell.isAttributeDefault() ? 0 : (
          (cell.isBold() ? 0x01 : 0) |
          (cell.isUnderline() ? 0x02 : 0) |
          (cell.isItalic() ? 0x04 : 0) |
          (cell.isDim() ? 0x08 : 0)
        );

        // Check if style changed
        const styleChanged =
          fg !== currentFg ||
          bg !== currentBg ||
          attrs !== currentAttrs;

        if (styleChanged && currentText) {
          // Flush current chunk
          chunks.push(cellToTextChunk(currentText, currentFg, currentBg, currentAttrs));
          currentText = '';
        }

        currentFg = fg;
        currentBg = bg;
        currentAttrs = attrs;
        currentText += char || ' ';
      }

      // Flush final chunk
      if (currentText) {
        chunks.push(cellToTextChunk(currentText, currentFg, currentBg, currentAttrs));
      }

      lines.push({ chunks: chunks.length > 0 ? chunks : [{ __isChunk: true, text: '' }] });
    }

    return lines;
  }

  /**
   * Get only visible lines (non-empty or recent)
   */
  getVisibleLines(maxLines: number = 50): AnsiLine[] {
    const allLines = this.getLines();

    // Get the last N lines
    return allLines.slice(-maxLines);
  }

  /**
   * Clear the terminal buffer
   */
  clear(): void {
    this.terminal.clear();
  }

  /**
   * Resize the terminal
   */
  resize(cols: number, rows: number): void {
    this.terminal.resize(cols, rows);
  }
}
