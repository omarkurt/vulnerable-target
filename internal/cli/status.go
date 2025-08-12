package cli

import (
	"context"
	"time"

	"github.com/happyhackingspace/vulnerable-target/internal/status"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Shows the status of running vulnerable targets",
	Long:  `Displays a real-time status of all running vulnerable targets with their health status, exposed ports, and resource usage.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Create and run the status TUI
		tui := status.NewStatusTUI()
		if err := tui.Run(ctx); err != nil {
			log.Fatal().Err(err).Msg("Failed to run status TUI")
		}
	},
}

var statusWatchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Continuously monitor status of running targets",
	Long:  `Continuously monitors and updates the status of running vulnerable targets in real-time.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Create and run the status TUI in watch mode
		tui := status.NewStatusTUI()
		tui.SetWatchMode(true)
		
		ctx := context.Background()
		if err := tui.Run(ctx); err != nil {
			log.Fatal().Err(err).Msg("Failed to run status TUI in watch mode")
		}
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.AddCommand(statusWatchCmd)
}
