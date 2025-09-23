# Pane Rendering Notes

This document captures the working knowledge of Agate's pane layout system and the relevant Lip Gloss behaviors we rely on.

## Layout Overview

- `Layout` (in `layout.go`) is the single source of truth for widths and heights. It is initialized with the terminal `width` and `height` and exposes helper methods for each pane's content area.
- We treat the UI as three horizontal columns:
  1. Repos/worktrees pane
  2. Tmux pane
  3. Right column (Git above Shell)
- Each column is rendered with the shared `paneBaseStyle` (`RoundedBorder` + `Padding(1,2)`), so every pane adds the same border + padding chrome around its content.
- Horizontal spacing is driven by two constants in `layout.go`:
  - `horizontalMargin` controls the outer margin between the panes block and the terminal edges.
  - `horizontalGapWidth` is the gutter inserted between the three columns. We subtract `2*horizontalMargin + 2*horizontalGapWidth` from the terminal width when sizing panes, then add the same spaces back when rendering (gaps + outer padding) so the grid occupies the full width.
- Column widths are computed from the _content_ width first, then the frame (border + padding) is added back on:
  - Left column = 25% of available content width
  - Tmux column = 50%
  - Right column = remainder (shared between Git & Shell)
- Heights are calculated by removing a set of fixed “chrome” rows from the terminal height. Constants (`topPaddingRows`, `paneTitleRows`, `footerRows`, etc.) document that subtraction explicitly. The remaining height becomes `paneHeight` for the full-height columns. The right column then splits that height between Git and Shell while keeping their titles + panes aligned with the full-height columns.

## Rendering Flow

1. **Content Wrapping** – we first apply `Width(...).MaxHeight(...)` styles to the raw content strings so wrapping happens before borders. Wrapping must be done on a fresh style because reusing `paneBaseStyle` would double-apply borders.
2. **Vertical Placement** – `lipgloss.PlaceVertical(contentHeight, Top, wrappedContent)` pads with blank lines _inside_ the pane so the rendered block reaches the desired content height. Lip Gloss returns the block unchanged if the target height is less than the content height (no shrinking happens).
3. **Border Application** – we finally render with `paneBaseStyle.Height(targetHeightWithBorders)`. Because the style includes padding above/below, we feed it `paneHeight - 2` (accounting for the two border rows) to hit the exact outer height we computed earlier.
4. **Joining** – we prepend titles, then `JoinVertical`/`JoinHorizontal` to assemble the final grid. No extra spacing is introduced between joins, so all horizontal spacing must be accounted for in the width math.

## Key Lip Gloss Behaviors

- `Style.GetHorizontalFrameSize()` returns the total characters consumed by borders + padding (both sides). This is why we subtract it when computing content widths and add it back when setting pane widths.
- `Style.Height(n)` never truncates. If the rendered block would be smaller than `n`, Lip Gloss pads with blank lines _inside_ the style before borders are drawn. If `n` is smaller than the block, the original height is kept.
- `Style.Width(n)` affects wrapping before borders are considered. For panes with padding, you must subtract the frame width to get the usable content width.
- `lipgloss.PlaceVertical` and `lipgloss.PlaceHorizontal` only pad—they never crop strings. We rely on that to top-align content without worrying about overflows.

## Practical Tips

- Always compute content dimensions first, then render borders. Mixing the two leads to off-by-one issues because padding/border rows get counted twice.
- When matching two panes’ heights, compare `lipgloss.Height(renderedPane)`, not `len(string)`, because ANSI sequences and multi-byte characters change byte counts without affecting the terminal height.
- If you need to align panes horizontally, ensure the sum of `paneWidth` values plus any intentional gutters equals the terminal width. Any leftover width shows up as blank space on the right.
- Tests can stub `NewLayout(width, height)` and assert both individual pane heights and the height of the fully joined view (see `layout_internal_test.go`).

## Open Questions / Follow Ups

- If we introduce dynamic gutters or resize handles, we’ll need to make the chrome constants configurable instead of hardcoded in `layout.go`.
