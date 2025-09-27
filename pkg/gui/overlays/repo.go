package overlays

import (
	"agate/internal/debug"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// RepoDialog represents the dialog for adding new repositories using fzf
type RepoDialog struct {
	width     int
	height    int
	searching bool
	err       string
	startTime time.Time
	spinner   spinner.Model
}

// NewRepoDialog creates a new repository search dialog
func NewRepoDialog() *RepoDialog {
	debug.DebugLog("NewRepoDialog() called")
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = dialogInfoStyle

	return &RepoDialog{
		searching: false,
		spinner:   s,
	}
}

// Init implements tea.Model
func (d *RepoDialog) Init() tea.Cmd {
	debug.DebugLog("RepoDialog.Init() called")
	// Start the repository discovery and fzf selection process
	d.searching = true
	d.startTime = time.Now()
	debug.DebugLog("Starting repository search with spinner and StartRepoSelection()")
	return tea.Batch(
		d.spinner.Tick,
		StartRepoSelection(),
	)
}

// Update implements tea.Model
func (d *RepoDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// ESC cancels the dialog
		if msg.String() == "esc" && !d.searching {
			return d, func() tea.Msg {
				return RepoDialogCancelledMsg{}
			}
		}

	case RepoSelectedMsg:
		d.searching = false
		if msg.Error != "" {
			d.err = msg.Error
			return d, nil
		}

		// Repository was successfully selected
		return d, func() tea.Msg {
			return RepoAddedMsg{Path: msg.Path}
		}

	case RepoSelectionCancelledMsg:
		// User cancelled fzf selection
		return d, func() tea.Msg {
			return RepoDialogCancelledMsg{}
		}

	default:
		// Update spinner if searching
		if d.searching {
			d.spinner, cmd = d.spinner.Update(msg)
		}
	}

	return d, cmd
}

// SetSize updates the dialog dimensions
func (d *RepoDialog) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// View implements tea.Model and renders the dialog
func (d *RepoDialog) View() string {
	var content []string

	// Title
	content = append(content, dialogTitleStyle.Render("Add a new repo"))
	content = append(content, "")

	if d.searching {
		// Show searching state with spinner
		content = append(content, d.spinner.View()+" Discovering repositories...")
		content = append(content, "")
		content = append(content, dialogInfoStyle.Render("This may take a moment..."))
		content = append(content, "")
		content = append(content, dialogButtonStyle.Render("[ Cancel (ESC) ]"))
	} else if d.err != "" {
		// Show error
		content = append(content, dialogErrorStyle.Render("Error: "+d.err))
		content = append(content, "")
		content = append(content, dialogButtonStyle.Render("[ Cancel (ESC) ]"))
	} else {
		// Default state (shouldn't normally be seen)
		content = append(content, dialogInfoStyle.Render("Preparing repository search..."))
		content = append(content, "")
		content = append(content, dialogButtonStyle.Render("[ Cancel (ESC) ]"))
	}

	// Join all content and apply dialog styling
	dialogContent := strings.Join(content, "\n")
	return dialogStyle.Render(dialogContent)
}

// discoverRepositories finds git repositories in common search paths
func discoverRepositories() ([]string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	// Common paths where developers keep their projects (excluding home directory to avoid permission dialogs)
	searchPaths := []string{
		filepath.Join(homeDir, "Dev"),
		filepath.Join(homeDir, "Development"),
		filepath.Join(homeDir, "Projects"),
		filepath.Join(homeDir, "Git"),
		filepath.Join(homeDir, "Code"),
		filepath.Join(homeDir, "src"),
		filepath.Join(homeDir, "workspace"),
		filepath.Join(homeDir, "Documents"),
		filepath.Join(homeDir, "Desktop"),
		// Note: Removed homeDir directly to avoid macOS permission dialogs
	}

	var repos []string

	for _, searchPath := range searchPaths {
		// Check if the search path exists
		if _, err := os.Stat(searchPath); os.IsNotExist(err) {
			continue
		}

		// Walk the directory looking for .git directories
		err := filepath.WalkDir(searchPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // Skip directories we can't read
			}

			// Skip macOS system and protected directories to avoid permission dialogs
			if d.IsDir() {
				dirName := d.Name()
				// Skip hidden directories and macOS system directories
				if strings.HasPrefix(dirName, ".") ||
					dirName == "Library" ||
					dirName == "Applications" ||
					dirName == "System" ||
					dirName == "usr" ||
					dirName == "bin" ||
					dirName == "sbin" ||
					dirName == "private" ||
					dirName == "var" ||
					dirName == "tmp" ||
					dirName == "Volumes" ||
					dirName == "Network" ||
					dirName == "cores" {
					return filepath.SkipDir
				}
			}

			// If we find a .git directory, the parent is a repository
			if d.IsDir() && d.Name() == ".git" {
				repoPath := filepath.Dir(path)
				repos = append(repos, repoPath)
				return filepath.SkipDir // Don't recurse into .git
			}

			// Don't recurse too deeply to avoid performance issues
			depth := strings.Count(strings.TrimPrefix(path, searchPath), string(filepath.Separator))
			if depth > 2 { // Reduced from 3 to 2 for better performance
				return filepath.SkipDir
			}

			return nil
		})

		if err != nil {
			continue // Skip paths that cause errors
		}
	}

	return repos, nil
}

// checkFzfInstalled checks if fzf is available
func checkFzfInstalled() error {
	_, err := exec.LookPath("fzf")
	if err != nil {
		return fmt.Errorf("fzf is not installed. Please install fzf to use repository search.\n" +
			"Install with:\n" +
			"  macOS: brew install fzf\n" +
			"  Ubuntu/Debian: sudo apt install fzf\n" +
			"  Other: https://github.com/junegunn/fzf#installation")
	}
	return nil
}

// runFzfSelection runs fzf with the discovered repositories and returns the selected one
func runFzfSelection(repos []string) (string, error) {
	if len(repos) == 0 {
		return "", fmt.Errorf("no repositories found in common directories")
	}

	// Check if fzf is installed
	if err := checkFzfInstalled(); err != nil {
		return "", err
	}

	// Prepare input for fzf
	input := strings.Join(repos, "\n")

	// Create fzf command with nice options
	cmd := exec.Command("fzf",
		"--prompt=Select repository: ",
		"--header=Use arrow keys to navigate, Enter to select, Ctrl+C to cancel",
		"--height=50%",
		"--layout=reverse",
		"--border",
	)

	// Set up pipes
	cmd.Stdin = strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run fzf
	err := cmd.Run()
	if err != nil {
		// Check if user cancelled (exit code 130 for Ctrl+C, 1 for ESC)
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() == 130 || exitError.ExitCode() == 1 {
				return "", nil // User cancelled
			}
		}
		return "", fmt.Errorf("fzf error: %v", err)
	}

	// Get the selected repository path
	selected := strings.TrimSpace(stdout.String())
	if selected == "" {
		return "", nil // No selection made
	}

	return selected, nil
}

// StartRepoSelection starts the background repository discovery and fzf selection
func StartRepoSelection() tea.Cmd {
	return func() tea.Msg {
		// Add debug output
		debug.DebugLog("Starting repository discovery...")

		// First discover repositories
		repos, err := discoverRepositories()
		if err != nil {
			debug.DebugLog("Repository discovery failed: %v", err)
			return RepoSelectedMsg{Error: err.Error()}
		}

		debug.DebugLog("Found %d repositories, starting fzf...", len(repos))

		// Then run fzf selection
		selected, err := runFzfSelection(repos)
		if err != nil {
			debug.DebugLog("fzf selection failed: %v", err)
			return RepoSelectedMsg{Error: err.Error()}
		}

		if selected == "" {
			debug.DebugLog("User cancelled fzf selection")
			// User cancelled
			return RepoSelectionCancelledMsg{}
		}

		debug.DebugLog("User selected: %s", selected)
		return RepoSelectedMsg{Path: selected}
	}
}

// Message types for repository dialog

type RepoSelectedMsg struct {
	Path  string
	Error string
}

type RepoSelectionCancelledMsg struct{}

type RepoAddedMsg struct {
	Path string
}

type RepoDialogCancelledMsg struct{}
