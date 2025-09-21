package icons

import (
	"os"
	"strings"
)

// Icon represents an icon with Nerd Font and fallback options
type Icon struct {
	NerdFont string
	Fallback string
}

// Icons we actually need for the worktree list
var (
	// Repository/branch icon (main one we need)
	GitRepo = Icon{
		NerdFont: "\ue0a0", // Nerd Font git branch icon
		Fallback: "Ꮧ",      // Claude Squad's branch icon
	}

	// Status indicators (for future use)
	Ready = Icon{
		NerdFont: "\uf00c", // Nerd Font check mark
		Fallback: "●",      // Claude Squad's ready icon
	}

	// Navigation (already using Unicode, but providing Nerd Font versions)
	Selected = Icon{
		NerdFont: "\ue0b0", // Nerd Font right arrow
		Fallback: "▶",      // Current Unicode arrow
	}

	Current = Icon{
		NerdFont: "\uf0a4", // Nerd Font current indicator
		Fallback: "→",      // Current Unicode arrow
	}

	// Home icon for main repository entries
	Home = Icon{
		NerdFont: "\uf015", // Nerd Font home icon
		Fallback: "🏠",      // Unicode home emoji
	}

	// Folder icon for main repository entries
	Folder = Icon{
		NerdFont: "\uf07b", // Nerd Font folder icon
		Fallback: "📁",      // Unicode folder emoji
	}
)

var useNerdFonts *bool

// hasNerdFonts detects if Nerd Fonts are likely available
func hasNerdFonts() bool {
	if useNerdFonts != nil {
		return *useNerdFonts
	}

	// Check common environment variables that indicate Nerd Font usage
	termProgram := strings.ToLower(os.Getenv("TERM_PROGRAM"))
	term := strings.ToLower(os.Getenv("TERM"))

	// Common terminals/configs that often use Nerd Fonts
	nerdFontTerms := []string{
		"alacritty", "kitty", "wezterm", "iterm", "hyper", "ghostty",
		"tmux-256color", "xterm-256color", "xterm-ghostty",
	}

	result := false
	for _, nfTerm := range nerdFontTerms {
		if strings.Contains(termProgram, nfTerm) || strings.Contains(term, nfTerm) {
			result = true
			break
		}
	}

	// Cache the result
	useNerdFonts = &result
	return result
}

// Get returns the appropriate icon string based on Nerd Font availability
func (i Icon) Get() string {
	if hasNerdFonts() {
		return i.NerdFont
	}
	return i.Fallback
}

// SetNerdFonts manually overrides Nerd Font detection
func SetNerdFonts(enabled bool) {
	useNerdFonts = &enabled
}

// Convenience functions for the icons we actually use
func GetGitRepo() string {
	return GitRepo.Get()
}

func GetHome() string {
	return Home.Get()
}

func GetFolder() string {
	return Folder.Get()
}