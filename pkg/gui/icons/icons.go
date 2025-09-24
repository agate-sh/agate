// Package icons provides consistent icon representations using Nerd Fonts
// for various file types and UI elements in the agate interface.
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

	// Git status icons for individual files
	GitModified = Icon{
		NerdFont: "\U000f1500", // Nerd Font square with dot icon (exact GitHub Desktop style)
		Fallback: "M",          // Modified
	}

	GitAdded = Icon{
		NerdFont: "\uf0fe", // Nerd Font plus square icon
		Fallback: "A",      // Added
	}

	GitDeleted = Icon{
		NerdFont: "\uf146", // Nerd Font minus square icon
		Fallback: "D",      // Deleted
	}

	GitRenamed = Icon{
		NerdFont: "\uf0ec", // Nerd Font arrow-right icon
		Fallback: "R",      // Renamed
	}

	GitUntracked = Icon{
		NerdFont: "\uf0fe", // Nerd Font plus square icon (same as added, green for new files)
		Fallback: "?",      // Untracked
	}

	GitTypeChanged = Icon{
		NerdFont: "\uf0ad", // Nerd Font wrench icon
		Fallback: "T",      // Type changed
	}

	GitConflicted = Icon{
		NerdFont: "\uf071", // Nerd Font warning icon
		Fallback: "C",      // Conflicted
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

// GetGitRepo returns the Git repository icon
func GetGitRepo() string {
	return GitRepo.Get()
}

// GetHome returns the home directory icon
func GetHome() string {
	return Home.Get()
}

// GetFolder returns the folder icon
func GetFolder() string {
	return Folder.Get()
}

// GetGitStatusIcon returns the appropriate Git status icon for a file status
func GetGitStatusIcon(status string) string {
	switch status {
	case "M", "MM", "AM": // Modified (index, working tree, or both)
		return GitModified.Get()
	case "A", "AD": // Added
		return GitAdded.Get()
	case "D", "DM": // Deleted
		return GitDeleted.Get()
	case "R", "RM": // Renamed
		return GitRenamed.Get()
	case "??": // Untracked
		return GitUntracked.Get()
	case "T": // Type changed
		return GitTypeChanged.Get()
	case "UU", "AA", "DD": // Conflicted
		return GitConflicted.Get()
	default:
		return GitModified.Get() // Default fallback
	}
}
