package icons

import (
	"os"
	"strings"
)

// Icon represents a glyph with Nerd Font and fallback characters.
type Icon struct {
	NerdFont string
	Fallback string
}

var (
	GitRepo = Icon{NerdFont: "\ue725", Fallback: "Ꮧ"}
	Ready   = Icon{NerdFont: "\uf00c", Fallback: "●"}
	Home    = Icon{NerdFont: "\uf015", Fallback: "🏠"}
	Link    = Icon{NerdFont: "\uf0c1", Fallback: "🔗"}
)

var useNerdFonts *bool

func hasNerdFonts() bool {
	if useNerdFonts != nil {
		return *useNerdFonts
	}

	termProgram := strings.ToLower(os.Getenv("TERM_PROGRAM"))
	term := strings.ToLower(os.Getenv("TERM"))

	candidates := []string{
		"alacritty", "kitty", "wezterm", "iterm", "hyper", "ghostty",
		"tmux-256color", "xterm-256color", "xterm-ghostty",
	}

	result := false
	for _, candidate := range candidates {
		if strings.Contains(termProgram, candidate) || strings.Contains(term, candidate) {
			result = true
			break
		}
	}

	useNerdFonts = &result
	return result
}

// SetNerdFonts allows forcing Nerd Font usage (useful in tests).
func SetNerdFonts(enabled bool) {
	useNerdFonts = &enabled
}

func (i Icon) Get() string {
	if hasNerdFonts() {
		return i.NerdFont
	}
	return i.Fallback
}
