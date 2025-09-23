package main

import (
	"fmt"
	"strings"

	"agate/git"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// WorktreeConfirmDialog represents the confirmation dialog for deleting worktrees
type WorktreeConfirmDialog struct {
	worktree        *git.WorktreeInfo
	worktreeManager *git.WorktreeManager
	width           int
	height          int
	deleting        bool
}

// Styling for confirmation dialog
var (
	confirmDialogStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(errorStatus)).
				Padding(1, 2).
				MaxWidth(50)

	confirmTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color(errorStatus)).
				MarginBottom(1)

	confirmInfoStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(textPrimary)). // White
				MarginBottom(1)

	confirmWarningStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(warningStatus)).
				MarginTop(1).
				MarginBottom(1)

	confirmButtonsStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(infoStatus)).
				MarginTop(1)

	confirmDeletingStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(textMuted)). // Gray
				MarginTop(1)
)

// NewWorktreeConfirmDialog creates a new worktree deletion confirmation dialog
func NewWorktreeConfirmDialog(worktree *git.WorktreeInfo, worktreeManager *git.WorktreeManager) *WorktreeConfirmDialog {
	return &WorktreeConfirmDialog{
		worktree:        worktree,
		worktreeManager: worktreeManager,
		deleting:        false,
	}
}

// Init implements tea.Model
func (d *WorktreeConfirmDialog) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (d *WorktreeConfirmDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't process keys if we're in the middle of deleting
		if d.deleting {
			return d, nil
		}

		switch msg.String() {
		case "y", "Y":
			// Confirm deletion
			return d, d.deleteWorktree()

		case "n", "N", "esc":
			// Cancel deletion
			return d, func() tea.Msg {
				return WorktreeDeleteCancelledMsg{}
			}
		}

	case WorktreeDeletedMsg:
		// Deletion completed successfully
		d.deleting = false
		return d, func() tea.Msg {
			return msg // Forward the message
		}

	case WorktreeDeletionErrorMsg:
		// Deletion failed
		d.deleting = false
		return d, func() tea.Msg {
			return msg // Forward the message for error handling
		}
	}

	return d, nil
}

// deleteWorktree deletes the worktree
func (d *WorktreeConfirmDialog) deleteWorktree() tea.Cmd {
	if d.worktreeManager == nil || d.worktree == nil {
		return func() tea.Msg {
			return WorktreeDeletionErrorMsg{Error: "Worktree manager or worktree not available"}
		}
	}

	// Set deleting state
	d.deleting = true

	// Delete worktree in background
	return func() tea.Msg {
		err := d.worktreeManager.DeleteWorktree(*d.worktree)
		if err != nil {
			return WorktreeDeletionErrorMsg{Error: err.Error()}
		}
		return WorktreeDeletedMsg{Worktree: d.worktree}
	}
}

// SetSize updates the dialog dimensions
func (d *WorktreeConfirmDialog) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// View implements tea.Model and renders the confirmation dialog
func (d *WorktreeConfirmDialog) View() string {
	if d.worktree == nil {
		return confirmDialogStyle.Render("Error: No worktree selected")
	}

	var content []string

	// Title
	content = append(content, confirmTitleStyle.Render("Delete Worktree?"))
	content = append(content, "")

	// Worktree info
	content = append(content, confirmInfoStyle.Render("Repository: "+d.worktree.RepoName))
	content = append(content, confirmInfoStyle.Render("Branch: "+d.worktree.Branch))

	// Status info
	if d.worktree.GitStatus != nil && !d.worktree.GitStatus.IsClean {
		status := d.worktree.GitStatus
		var statusParts []string

		if status.Modified > 0 {
			statusParts = append(statusParts, fmt.Sprintf("%d modified", status.Modified))
		}
		if status.Staged > 0 {
			statusParts = append(statusParts, fmt.Sprintf("%d staged", status.Staged))
		}
		if status.Untracked > 0 {
			statusParts = append(statusParts, fmt.Sprintf("%d untracked", status.Untracked))
		}

		if len(statusParts) > 0 {
			content = append(content, confirmInfoStyle.Render("Status: "+strings.Join(statusParts, ", ")))
		}
	} else {
		content = append(content, confirmInfoStyle.Render("Status: clean"))
	}

	content = append(content, "")

	// Warning about what will be deleted
	content = append(content, confirmWarningStyle.Render("This will:"))
	content = append(content, confirmWarningStyle.Render("- Remove the worktree directory"))
	content = append(content, confirmWarningStyle.Render("- Delete the branch '"+d.worktree.Branch+"'"))

	// Check for uncommitted changes warning
	if d.worktree.GitStatus != nil && !d.worktree.GitStatus.IsClean {
		content = append(content, "")
		content = append(content, confirmWarningStyle.Render("⚠️  This worktree has uncommitted changes!"))
		content = append(content, confirmWarningStyle.Render("   These changes will be lost."))
	}

	content = append(content, "")

	// Buttons or deleting message
	if d.deleting {
		content = append(content, confirmDeletingStyle.Render("Deleting worktree..."))
	} else {
		content = append(content, confirmButtonsStyle.Render("Press 'y' to confirm, 'n' to cancel"))
	}

	// Join all content and apply dialog styling
	dialogContent := strings.Join(content, "\n")
	return confirmDialogStyle.Render(dialogContent)
}

// WorktreeDeletedMsg indicates a worktree was successfully deleted
type WorktreeDeletedMsg struct {
	Worktree *git.WorktreeInfo
}

// WorktreeDeletionErrorMsg indicates worktree deletion failed
type WorktreeDeletionErrorMsg struct {
	Error string
}

// WorktreeDeleteCancelledMsg indicates the deletion was cancelled
type WorktreeDeleteCancelledMsg struct{}
