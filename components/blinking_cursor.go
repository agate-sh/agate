package components

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
)

// BlinkingCursor is a simple on/off cursor spinner
var BlinkingCursor = spinner.Spinner{
	Frames: []string{"█", "░"},
	FPS:    time.Millisecond * 500, // Blink every 500ms
}