package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(teamCmd)
	teamCmd.AddCommand(teamListCmd)
	teamCmd.AddCommand(teamSpawnCmd)
}

var teamCmd = &cobra.Command{
	Use:   "team",
	Short: "Manage agent teams",
	Long:  `List and spawn pre-configured teams of worker personas.`,
}

var teamListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured teams",
	RunE:  runTeamList,
}

var teamSpawnCmd = &cobra.Command{
	Use:   "spawn <name>",
	Short: "Spawn all workers in a team",
	Args:  cobra.ExactArgs(1),
	RunE:  runTeamSpawn,
}

func loadConfig(missionDir string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(filepath.Join(missionDir, "config.json"))
	if err != nil {
		return cfg, fmt.Errorf("failed to read config: %w", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("failed to parse config: %w", err)
	}
	return cfg, nil
}

func runTeamList(cmd *cobra.Command, args []string) error {
	missionDir, err := findMissionDir()
	if err != nil {
		return err
	}

	cfg, err := loadConfig(missionDir)
	if err != nil {
		return err
	}

	if len(cfg.Teams) == 0 {
		fmt.Println("No teams configured.")
		return nil
	}

	for name, team := range cfg.Teams {
		zone := team.Zone
		if zone == "" {
			zone = "(any)"
		}
		fmt.Printf("%-20s zone=%-12s personas=%v\n", name, zone, team.Personas)
	}
	return nil
}

func runTeamSpawn(cmd *cobra.Command, args []string) error {
	teamName := args[0]

	missionDir, err := findMissionDir()
	if err != nil {
		return err
	}

	cfg, err := loadConfig(missionDir)
	if err != nil {
		return err
	}

	team, ok := cfg.Teams[teamName]
	if !ok {
		return fmt.Errorf("team %q not found in config", teamName)
	}

	if len(team.Personas) == 0 {
		return fmt.Errorf("team %q has no personas", teamName)
	}

	fmt.Printf("Spawning team %q (%d workers)...\n", teamName, len(team.Personas))

	for _, persona := range team.Personas {
		taskDesc := fmt.Sprintf("Team %s: %s worker", teamName, persona)
		spawnArgs := []string{persona, taskDesc}
		if team.Zone != "" {
			spawnCmd.Flags().Set("zone", team.Zone)
		}
		if err := runSpawn(spawnCmd, spawnArgs); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to spawn %s: %v\n", persona, err)
			continue
		}
	}

	return nil
}
