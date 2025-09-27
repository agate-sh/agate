package theme

// Theme defines all colors used throughout the application with semantic naming.
// All ANSI color codes have been converted to hex equivalents for consistency.
var (
	// Brand colors
	AgateColor = "#9d87ae" // Agate purple for branding and active elements

	// Text colors
	TextPrimary     = "#ffffff" // 255 - white text for focused/active elements
	TextDescription = "#c9c9c9" // 250 - light gray for descriptions and help text
	TextMuted       = "#7a7a7a" // 240 - dark gray for very subtle text like file paths

	// Border colors
	BorderActive = "#c9c9c9" // 250 - brighter gray for active non-tmux pane borders
	BorderMuted  = "#7a7a7a" // 240 - standard border color

	// Status/semantic colors (reused for borders when needed)
	SuccessStatus = "#50fa7b" // 83 - green for clean status, success states
	WarningStatus = "#ffb86c" // 214 - orange for warnings, dirty status
	ErrorStatus   = "#ff5555" // 196 - red for errors, dangerous actions
	InfoStatus    = "#8be9fd" // 86 - cyan for info, buttons, repo headers

	// UI colors
	HighlightBg    = "#282a36" // 16 - black for highlighted text backgrounds
	SeparatorColor = "#4a4a4a" // 238 - very dark gray for separators
	WarningYellow  = "#f1fa8c" // 220 - yellow for warnings/highlights
	White          = "#ffffff" // 7 - white for debug overlay and other UI elements
	RowHighlight   = "#525252" // subtle medium gray for row highlighting
)
