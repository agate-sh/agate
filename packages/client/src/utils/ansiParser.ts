import { Terminal, type IBufferCell } from '@xterm/headless';
import { RGBA, type TextChunk } from '@opentui/core';

/**
 * Represents a parsed line of terminal output with ANSI styling
 */
export interface AnsiLine {
  chunks: TextChunk[];
}

/**
 * Color mode for terminal cells
 */
enum ColorMode {
  DEFAULT = 0,
  PALETTE = 1,
  RGB = 2,
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
 * Convert color value to hex based on color mode
 */
function convertColorToHex(color: number, colorMode: ColorMode): string | undefined {
  if (colorMode === ColorMode.RGB) {
    // Extract RGB components from packed integer
    const r = (color >> 16) & 255;
    const g = (color >> 8) & 255;
    const b = color & 255;
    return `#${r.toString(16).padStart(2, '0')}${g.toString(16).padStart(2, '0')}${b.toString(16).padStart(2, '0')}`;
  }

  if (colorMode === ColorMode.PALETTE) {
    return ansi256ToHex(color);
  }

  // ColorMode.DEFAULT - return undefined to use theme default
  return undefined;
}

/**
 * Darken a hex color by a factor (for DIM attribute)
 * @param hexColor - Hex color string (e.g., "#b1b9f9")
 * @param factor - Brightness factor (0.6 = 60% brightness)
 * @returns Darkened hex color
 */
function darkenColor(hexColor: string, factor: number = 0.6): string {
  // Parse hex color
  const r = parseInt(hexColor.slice(1, 3), 16);
  const g = parseInt(hexColor.slice(3, 5), 16);
  const b = parseInt(hexColor.slice(5, 7), 16);

  // Apply dimming factor
  const dimR = Math.floor(r * factor);
  const dimG = Math.floor(g * factor);
  const dimB = Math.floor(b * factor);

  const result = `#${dimR.toString(16).padStart(2, '0')}${dimG.toString(16).padStart(2, '0')}${dimB.toString(16).padStart(2, '0')}`;

  return result;
}

/**
 * Creates a TextChunk from xterm.js cell data
 */
function cellToTextChunk(
  text: string,
  cell: IBufferCell | null,
  attrs: number
): TextChunk {
  const chunk: TextChunk = {
    __isChunk: true,
    text,
  };

  if (cell) {
    // Check if DIM and INVERSE attributes are set
    const isDim = (attrs & 0x08) !== 0;
    const isInverse = (attrs & 0x10) !== 0;

    // Determine foreground color mode and extract color
    let fgColorMode = ColorMode.DEFAULT;
    if (cell.isFgRGB()) {
      fgColorMode = ColorMode.RGB;
    } else if (cell.isFgPalette()) {
      fgColorMode = ColorMode.PALETTE;
    }

    // Handle inverse video by swapping foreground and background
    if (isInverse) {
      // Reverse video: swap fg and bg colors
      // Background becomes white (default foreground)
      // Foreground becomes dark (theme background)
      chunk.bg = RGBA.fromHex('#ffffff'); // White background block (default foreground)
      chunk.fg = RGBA.fromHex('#030620'); // Dark text (theme background color)
    } else if (fgColorMode !== ColorMode.DEFAULT) {
      let hexColor = convertColorToHex(cell.getFgColor(), fgColorMode);
      if (hexColor) {
        // If DIM attribute is set, darken the color manually
        // This ensures consistent dimming across terminals
        if (isDim) {
          hexColor = darkenColor(hexColor, 0.6); // 60% brightness
        }

        chunk.fg = RGBA.fromHex(hexColor);
      }
    } else if (isDim) {
      // DIM with default foreground color
      // Use a standard dim grey color since we can't rely on OpenTUI to apply dimming
      chunk.fg = RGBA.fromHex('#808080'); // Standard dim grey
    }

    // Don't set background - we want OpenTUI to use our theme background
    // Setting explicit bg here causes white backgrounds from shell prompts
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
  // Dim: 0x08 - DON'T pass this to OpenTUI, we handle it manually by setting color
  // if (attrs & 0x08) attributes |= 16; // TextAttributes.DIM

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
      theme: {
        background: '#030620', // theme.base - dark blue background
        foreground: '#ffffff', // theme.textPrimary - white text
        cursor: '#000000',
        cursorAccent: '#ffffff',
        // Standard ANSI colors
        black: '#000000',
        red: '#ff5555',
        green: '#50fa7b',
        yellow: '#f1fa8c',
        blue: '#8be9fd',
        magenta: '#ff79c6',
        cyan: '#8be9fd',
        white: '#ffffff',
        // Bright colors
        brightBlack: '#4a4a4a',
        brightRed: '#ff6e6e',
        brightGreen: '#69ff94',
        brightYellow: '#ffffa5',
        brightBlue: '#a4ffff',
        brightMagenta: '#ff92df',
        brightCyan: '#a4ffff',
        brightWhite: '#ffffff',
      },
    });
  }

  /**
   * Write data to the terminal buffer
   */
  write(data: string): void {
    this.terminal.write(data);
  }

  /**
   * Get visible viewport lines (what's currently displayed)
   * Iterates from viewportY for terminal.rows lines
   */
  getVisibleLines(): AnsiLine[] {
    const lines: AnsiLine[] = [];
    const buffer = this.terminal.buffer.active;
    const viewportY = buffer.viewportY;

    // Iterate through visible viewport (viewportY to viewportY + rows)
    for (let i = 0; i < this.terminal.rows; i++) {
      const row = viewportY + i;
      const line = buffer.getLine(row);
      if (!line) {
        // Empty line
        lines.push({ chunks: [{ __isChunk: true, text: '' }] });
        continue;
      }

      const chunks: TextChunk[] = [];
      let currentText = '';
      let currentCell: IBufferCell | null = null;
      let currentAttrs = 0;

      for (let col = 0; col < line.length; col++) {
        const cell = line.getCell(col);
        if (!cell) continue;

        const char = cell.getChars();
        const attrs = cell.isAttributeDefault() ? 0 : (
          (cell.isBold() ? 0x01 : 0) |
          (cell.isUnderline() ? 0x02 : 0) |
          (cell.isItalic() ? 0x04 : 0) |
          (cell.isDim() ? 0x08 : 0) |
          (cell.isInverse() ? 0x10 : 0) // Reverse video
        );

        // Check if style changed
        const fg = cell.getFgColor();
        const bg = cell.getBgColor();
        const prevFg = currentCell?.getFgColor();
        const prevBg = currentCell?.getBgColor();

        const styleChanged =
          fg !== prevFg ||
          bg !== prevBg ||
          attrs !== currentAttrs;

        if (styleChanged && currentText) {
          // Flush current chunk
          chunks.push(cellToTextChunk(currentText, currentCell, currentAttrs));
          currentText = '';
        }

        currentCell = cell;
        currentAttrs = attrs;
        currentText += char || ' ';
      }

      // Flush final chunk
      if (currentText) {
        chunks.push(cellToTextChunk(currentText, currentCell, currentAttrs));
      }

      lines.push({ chunks: chunks.length > 0 ? chunks : [{ __isChunk: true, text: '' }] });
    }

    return lines;
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
