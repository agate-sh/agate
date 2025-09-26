package config

// UIState captures UI-related state.
type UIState struct {
	Welcome WelcomeState `json:"welcome"`
}

// WelcomeState stores welcome overlay visibility.
type WelcomeState struct {
	Shown bool `json:"shown"`
}

// GetWelcomeShownState returns whether the welcome overlay has been shown
func GetWelcomeShownState() (bool, error) {
	state, err := LoadState()
	if err != nil {
		return false, err
	}
	return state.UI.Welcome.Shown, nil
}

// SetWelcomeShown sets the welcome shown state
func SetWelcomeShown(shown bool) error {
	state, err := LoadState()
	if err != nil {
		return err
	}

	state.UI.Welcome.Shown = shown
	return SaveState(state)
}
