package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// Global flag for --project
var projectFlag string

// ProjectRegistry maps project names to their .mission/ directory paths.
type ProjectRegistry struct {
	Projects map[string]string `json:"projects"`
}

func init() {
	// Add --project flag to root command (persistent = available to all subcommands)
	rootCmd.PersistentFlags().StringVar(&projectFlag, "project", "", "Use a registered project by name (see 'mc project list')")

	// Add project management commands
	rootCmd.AddCommand(projectCmd)
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectRegisterCmd)
	projectCmd.AddCommand(projectRemoveCmd)
	projectCmd.AddCommand(projectLinkCmd)
}

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage the global project registry",
	Long: `Manage registered MissionControl projects.

The project registry (~/.mc/projects.json) maps project names to .mission/
directory paths, enabling 'mc --project <name>' from any directory.

Symlinks are also supported: .mission/ can be a symlink pointing to a shared
mission directory, allowing multiple projects to reference the same configs.`,
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		reg, err := loadRegistry()
		if err != nil {
			return err
		}

		if len(reg.Projects) == 0 {
			fmt.Println("No projects registered. Use 'mc project register <name> <path>' to add one.")
			return nil
		}

		// Sort by name
		names := make([]string, 0, len(reg.Projects))
		for name := range reg.Projects {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			missionDir := reg.Projects[name]
			status := "✓"
			if _, err := os.Stat(missionDir); os.IsNotExist(err) {
				status = "✗ (not found)"
			}
			fmt.Printf("  %s  %-20s  %s\n", status, name, missionDir)
		}
		return nil
	},
}

var projectRegisterCmd = &cobra.Command{
	Use:   "register <name> [path]",
	Short: "Register a project in the global registry",
	Long: `Register a project name mapped to a .mission/ directory path.

If path is omitted, the .mission/ directory is found by walking up from cwd.
The path stored is the resolved absolute path to the .mission/ directory.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		var missionDir string
		if len(args) == 2 {
			p := args[1]
			// Expand ~
			if strings.HasPrefix(p, "~") {
				home, _ := os.UserHomeDir()
				p = filepath.Join(home, p[1:])
			}
			abs, err := filepath.Abs(p)
			if err != nil {
				return fmt.Errorf("failed to resolve path: %w", err)
			}
			// If they passed a project root, append .mission
			if filepath.Base(abs) != ".mission" {
				abs = filepath.Join(abs, ".mission")
			}
			missionDir = abs
		} else {
			var err error
			missionDir, err = findMissionDir()
			if err != nil {
				return err
			}
		}

		// Resolve symlinks to validate the target exists
		resolved, err := filepath.EvalSymlinks(missionDir)
		if err != nil {
			return fmt.Errorf(".mission/ path does not exist: %s", missionDir)
		}

		// Verify it's actually a directory
		info, err := os.Stat(resolved)
		if err != nil {
			return fmt.Errorf("cannot stat .mission/ path: %w", err)
		}
		if !info.IsDir() {
			return fmt.Errorf("%s is not a directory", missionDir)
		}

		// Verify it's actually a .mission directory (has state/ or config.json)
		_, errCfg := os.Stat(filepath.Join(resolved, "config.json"))
		_, errState := os.Stat(filepath.Join(resolved, "state"))
		if errCfg != nil && errState != nil {
			return fmt.Errorf("%s does not appear to be a valid .mission/ directory", missionDir)
		}

		reg, err := loadRegistry()
		if err != nil {
			return err
		}

		reg.Projects[name] = missionDir
		if err := saveRegistry(reg); err != nil {
			return err
		}

		fmt.Printf("Registered project '%s' → %s\n", name, missionDir)
		return nil
	},
}

var projectRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a project from the registry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		reg, err := loadRegistry()
		if err != nil {
			return err
		}

		if _, ok := reg.Projects[name]; !ok {
			return fmt.Errorf("project '%s' not found in registry", name)
		}

		delete(reg.Projects, name)
		if err := saveRegistry(reg); err != nil {
			return err
		}

		fmt.Printf("Removed project '%s' from registry\n", name)
		return nil
	},
}

var projectLinkCmd = &cobra.Command{
	Use:   "link <target-mission-dir> [link-location]",
	Short: "Create a .mission symlink to a shared mission directory",
	Long: `Create a symbolic link from .mission/ in the current (or specified) directory
to a target .mission/ directory. This allows multiple projects to share the
same mission configuration.

Examples:
  mc project link /path/to/shared/.mission
  mc project link ~/shared-mission ./my-project`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]

		// Expand ~
		if strings.HasPrefix(target, "~") {
			home, _ := os.UserHomeDir()
			target = filepath.Join(home, target[1:])
		}

		absTarget, err := filepath.Abs(target)
		if err != nil {
			return fmt.Errorf("failed to resolve target path: %w", err)
		}

		// Verify target exists
		if _, err := os.Stat(absTarget); os.IsNotExist(err) {
			return fmt.Errorf("target does not exist: %s", absTarget)
		}

		// Determine link location
		linkDir := ""
		if len(args) == 2 {
			linkDir = args[1]
			if strings.HasPrefix(linkDir, "~") {
				home, _ := os.UserHomeDir()
				linkDir = filepath.Join(home, linkDir[1:])
			}
		} else {
			linkDir, _ = os.Getwd()
		}

		linkPath := filepath.Join(linkDir, ".mission")

		// Check if .mission already exists
		if _, err := os.Lstat(linkPath); err == nil {
			return fmt.Errorf(".mission already exists at %s — remove it first", linkPath)
		}

		// Create symlink
		if err := os.Symlink(absTarget, linkPath); err != nil {
			return fmt.Errorf("failed to create symlink: %w", err)
		}

		fmt.Printf("Created symlink: %s → %s\n", linkPath, absTarget)
		return nil
	},
}

// registryPath returns the path to ~/.mc/projects.json
func registryPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine home directory: %w", err)
	}
	return filepath.Join(home, ".mc", "projects.json"), nil
}

func loadRegistry() (*ProjectRegistry, error) {
	reg := &ProjectRegistry{Projects: make(map[string]string)}

	path, err := registryPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return reg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read registry: %w", err)
	}

	if err := json.Unmarshal(data, reg); err != nil {
		return nil, fmt.Errorf("failed to parse registry: %w", err)
	}

	if reg.Projects == nil {
		reg.Projects = make(map[string]string)
	}

	return reg, nil
}

func saveRegistry(reg *ProjectRegistry) error {
	path, err := registryPath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create registry directory: %w", err)
	}

	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}
