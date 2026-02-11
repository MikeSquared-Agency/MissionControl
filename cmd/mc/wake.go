package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

var wakeCmd = &cobra.Command{
	Use:   "wake",
	Short: "Wake DutyBound services via the launcher",
	Long:  `Sends a wake request to the launcher and waits until all services are ready.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")
		baseURL := fmt.Sprintf("http://localhost:%d", port)
		client := &http.Client{Timeout: 5 * time.Second}

		// Send wake request
		fmt.Println("Waking DutyBound services...")
		resp, err := client.Post(baseURL+"/api/wake", "application/json", nil)
		if err != nil {
			return fmt.Errorf("could not reach launcher at %s: %w", baseURL, err)
		}
		resp.Body.Close()

		// Poll until ready
		timeout := time.After(90 * time.Second)
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-timeout:
				return fmt.Errorf("timed out waiting for services to become ready")
			case <-ticker.C:
				resp, err := client.Get(baseURL + "/api/health")
				if err != nil {
					continue
				}
				var health struct {
					Status string `json:"status"`
				}
				json.NewDecoder(resp.Body).Decode(&health)
				resp.Body.Close()

				switch health.Status {
				case "ready":
					fmt.Println("DutyBound services are ready.")
					return nil
				case "starting":
					fmt.Println("  Starting...")
				case "sleeping":
					// Shouldn't happen after wake, but retry
					fmt.Println("  Still sleeping, retrying wake...")
					r, err := client.Post(baseURL+"/api/wake", "application/json", nil)
					if err == nil {
						r.Body.Close()
					}
				}
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(wakeCmd)
	wakeCmd.Flags().Int("port", 8080, "Launcher port")
}
