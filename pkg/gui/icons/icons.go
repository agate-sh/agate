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
		NerdFont: "\ue725", // Nerd Font git branch icon
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

	// Link icon for linked worktrees
	Link = Icon{
		NerdFont: "\uf0c1", // Nerd Font link icon
		Fallback: "🔗",      // Unicode link emoji
	}

	// Pin icon for pinned sessions
	Pin = Icon{
		NerdFont: "\uf08d", // Nerd Font pushpin icon
		Fallback: "📌",      // Unicode pushpin emoji
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

	// Generic dropdown chevron for selectors
	ChevronDown = Icon{
		NerdFont: "\uf078", // Font Awesome chevron-down (available in Nerd Fonts)
		Fallback: "⌄",
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
// Status format is XY where X=staged status, Y=unstaged status, space=no change
func GetGitStatusIcon(status string) string {
	switch status {
	// Modified - covers all modification cases (staged, unstaged, or both)
	case " M", "M ", "MM", "AM":
		return GitModified.Get()
	// Added - new files in index
	case "A ", "AD":
		return GitAdded.Get()
	// Deleted - removed files
	case " D", "D ", "DD", "DM":
		return GitDeleted.Get()
	// Renamed (includes renamed + modified)
	case "R ", "RM":
		return GitRenamed.Get()
	// Untracked - new files not in index
	case "??":
		return GitUntracked.Get()
	// Type changed
	case "T ", " T", "TT":
		return GitTypeChanged.Get()
	// Conflicted
	case "UU", "AA":
		return GitConflicted.Get()
	default:
		return GitModified.Get() // Default fallback
	}
}

// GetChevronDown returns the dropdown chevron icon
func GetChevronDown() string {
	return ChevronDown.Get()
}
