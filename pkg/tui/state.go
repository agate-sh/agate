package tui

import "agate/pkg/tui/layout"

// UIMode represents the different modes the TUI can be in
type UIMode int

const (
	ModeSession    UIMode = iota // Normal session view
	ModeNewSession               // Creating new session
	ModeOverlay                  // Overlay is active
)

// String returns a string representation of the UIMode
func (m UIMode) String() string {
	switch m {
	case ModeSession:
		return "Session"
	case ModeNewSession:
		return "NewSession"
	case ModeOverlay:
		return "Overlay"
	default:
		return "Unknown"
	}
}

// OverlayType represents the different types of overlays
type OverlayType int

const (
	NoOverlay            OverlayType = iota // No overlay active
	HelpOverlay                             // Help dialog
	DebugOverlay                            // Debug overlay
	MergeOverlay                            // Merge changes overlay
	SessionDeleteOverlay                    // Session deletion confirmation
	AgentSelectorOverlay                    // Agent selector modal
)

// String returns a string representation of the OverlayType
func (o OverlayType) String() string {
	switch o {
	case NoOverlay:
		return "None"
	case HelpOverlay:
		return "Help"
	case DebugOverlay:
		return "Debug"
	case MergeOverlay:
		return "Merge"
	case SessionDeleteOverlay:
		return "SessionDelete"
	case AgentSelectorOverlay:
		return "AgentSelector"
	default:
		return "Unknown"
	}
}

// UIState represents the current state of the UI
type UIState struct {
	Mode          UIMode            // Current UI mode
	ActiveOverlay OverlayType       // Currently active overlay (if any)
	Focus         layout.FocusState // Current focus state
}

// InSession returns true if in normal session view mode
func (s *UIState) InSession() bool {
	return s.Mode == ModeSession
}

// InNewSession returns true if in new session creation mode
func (s *UIState) InNewSession() bool {
	return s.Mode == ModeNewSession
}

// InOverlay returns true if an overlay is active
func (s *UIState) InOverlay() bool {
	return s.Mode == ModeOverlay && s.ActiveOverlay != NoOverlay
}

// HasOverlay returns true if a specific overlay is active
func (s *UIState) HasOverlay(overlay OverlayType) bool {
	return s.Mode == ModeOverlay && s.ActiveOverlay == overlay
}
