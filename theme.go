package main

// Theme defines all colors used throughout the application with semantic naming.
// All ANSI color codes have been converted to hex equivalents for consistency.
var (
	// Brand colors
	agateColor = "#9d87ae" // Agate purple for branding and active elements

	// Text colors
	textPrimary     = "#ffffff" // 255 - white text for focused/active elements
	textDescription = "#c9c9c9" // 250 - light gray for descriptions and help text
	textMuted       = "#7a7a7a" // 240 - dark gray for very subtle text like file paths

	// Border colors
	borderActive = "#c9c9c9" // 250 - brighter gray for active non-tmux pane borders
	borderMuted  = "#7a7a7a" // 240 - standard border color

	// Status/semantic colors (reused for borders when needed)
	successStatus = "#50fa7b" // 83 - green for clean status, success states
	warningStatus = "#ffb86c" // 214 - orange for warnings, dirty status
	errorStatus   = "#ff5555" // 196 - red for errors, dangerous actions
	infoStatus    = "#8be9fd" // 86 - cyan for info, buttons, repo headers

	// UI colors
	highlightBg    = "#282a36" // 16 - black for highlighted text backgrounds
	separatorColor = "#4a4a4a" // 238 - very dark gray for separators
	warningYellow  = "#f1fa8c" // 220 - yellow for warnings/highlights
	white          = "#ffffff" // 7 - white for debug overlay and other UI elements
	selection      = "#dfcfae" // warm beige for selected items
)