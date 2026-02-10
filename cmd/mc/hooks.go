package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var hooksCmd = &cobra.Command{
	Use:   "hooks",
	Short: "Manage git hooks for MissionControl",
}

var hooksInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install git hooks for mission validation",
	Long:  `Installs a pre-push hook that runs "mc commit --validate-only" before allowing a push.`,
	RunE:  runHooksInstall,
}

const prePushHookScript = `#!/usr/bin/env bash
# MissionControl pre-push hook
# Validates mission state before allowing push

# Find mc binary: prefer the one next to this hook, then PATH
HOOK_DIR="$(cd "$(dirname "$0")" && pwd)"
MC="$(command -v mc 2>/dev/null || true)"
if [ -x "$HOOK_DIR/mc" ]; then
  MC="$HOOK_DIR/mc"
elif [ -z "$MC" ]; then
  echo "mc: command not found — install MissionControl or add mc to PATH" >&2
  exit 1
fi

"$MC" commit --validate-only
if [ $? -ne 0 ]; then
  echo "Push blocked: mission validation failed" >&2
  exit 1
fi
`

func findGitRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if info, err := os.Stat(filepath.Join(dir, ".git")); err == nil && info.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not a git repository (or any parent)")
		}
		dir = parent
	}
}

func runHooksInstall(cmd *cobra.Command, args []string) error {
	gitRoot, err := findGitRoot()
	if err != nil {
		return err
	}

	hooksDir := filepath.Join(gitRoot, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("creating hooks directory: %w", err)
	}

	hookPath := filepath.Join(hooksDir, "pre-push")

	// Check for existing hook — don't silently overwrite
	if _, err := os.Stat(hookPath); err == nil {
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			return fmt.Errorf("pre-push hook already exists at %s — use --force to overwrite", hookPath)
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "⚠ Overwriting existing pre-push hook\n")
	}

	if err := os.WriteFile(hookPath, []byte(prePushHookScript), 0755); err != nil {
		return fmt.Errorf("writing pre-push hook: %w", err)
	}

	fmt.Printf("Installed pre-push hook at %s\n", hookPath)
	return nil
}

func init() {
	hooksInstallCmd.Flags().Bool("force", false, "Overwrite existing hooks")
	hooksCmd.AddCommand(hooksInstallCmd)
	rootCmd.AddCommand(hooksCmd)
}
