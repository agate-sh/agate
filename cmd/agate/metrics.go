package main

import (
	"fmt"

	"agate/pkg/gui/metrics"
	"agate/pkg/session"
	"agate/pkg/state"

	"github.com/spf13/cobra"
)

// SetupMetricsCommand creates and configures the metrics subcommand
func SetupMetricsCommand() *cobra.Command {
	var metricsSessionID string

	metricsCmd := &cobra.Command{
		Use:   "metrics",
		Short: "View live metrics for a session",
		Long:  `Display a live metrics view for the specified session ID.`,
		RunE: func(_ *cobra.Command, args []string) error {
			if metricsSessionID == "" {
				return fmt.Errorf("--session-id is required")
			}

			// Initialize state manager to load session
			stateManager, err := state.NewManager()
			if err != nil {
				return fmt.Errorf("failed to initialize state manager: %w", err)
			}

			// Load sessions from state
			sessionMappings := stateManager.GetSessionMappings()
			persistedSession, exists := sessionMappings[metricsSessionID]
			if !exists {
				return fmt.Errorf("session not found: %s", metricsSessionID)
			}

			// Reconstruct session object from persisted state
			sess := &session.Session{
				ID:             persistedSession.ID,
				Prompt:         persistedSession.Prompt,
				Description:    persistedSession.Description,
				BranchBaseName: persistedSession.BranchBaseName,
				CreatedAt:      persistedSession.CreatedAt,
				LastAccessed:   persistedSession.LastAccessed,
			}

			// Run metrics TUI
			return metrics.Run(sess)
		},
	}

	metricsCmd.Flags().StringVar(&metricsSessionID, "session-id", "", "Session ID to view metrics for (required)")

	return metricsCmd
}
