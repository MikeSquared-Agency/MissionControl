package main

import (
	"fmt"
	"path/filepath"

	"github.com/MikeSquared-Agency/MissionControl/serve"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MissionControl orchestrator server",
	Long:  `Starts the HTTP/WebSocket server that provides the API for the MC dashboard.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")
		apiOnly, _ := cmd.Flags().GetBool("api-only")
		headless, _ := cmd.Flags().GetBool("headless")

		missionPath, err := findMissionDir()
		if err != nil {
			return fmt.Errorf("no .mission/ directory found. Run 'mc init' first: %w", err)
		}
		// findMissionDir returns the .mission/ path; serve expects the parent project dir
		missionDir := filepath.Dir(missionPath)

		return serve.Run(serve.Config{
			Port:       port,
			MissionDir: missionDir,
			APIOnly:    apiOnly,
			Headless:   headless,
		})
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().Int("port", 8080, "Port to listen on")
	serveCmd.Flags().Bool("api-only", false, "Disable file watcher and process tracker")
	serveCmd.Flags().Bool("headless", false, "API only, no dashboard")
}
