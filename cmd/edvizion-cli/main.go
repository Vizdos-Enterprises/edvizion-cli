package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vizdos-enterprises/edvizion-cli/domain"
	"github.com/vizdos-enterprises/edvizion-cli/releases"
)

var tools = map[string]domain.Tool{
	"bis-ecommerce-theme-dev": {
		Name:          "bis-ecommerce-theme-dev",
		Repository:    "Vizdos-Enterprises/DLive",
		BinaryName:    "bis-ecommerce-theme-dev",
		ReleasePrefix: "bis-v",
	},

	// Add more predefined tools here:
	//
	// "another-predefined": {
	// 	Name:       "another-predefined",
	// 	Repository: "Vizdos-Enterprises/another-predefined",
	// 	BinaryName: "another-predefined",
	// },
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "edvizion-cli",
		Short: "EdVizion development tools",
	}

	rootCmd.AddCommand(
		newGetCommand(),
		newUpgradeCommand(),
		newRunCommand(),
		newListCommand(),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func newGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get <tool>",
		Short: "Download and install an EdVizion tool",
		Args:  cobra.ExactArgs(1),

		RunE: func(cmd *cobra.Command, args []string) error {
			tool, err := getTool(args[0])
			if err != nil {
				return err
			}

			return installTool(cmd.Context(), tool)
		},
	}
}

func newUpgradeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade <tool>",
		Short: "Upgrade an installed EdVizion tool",
		Args:  cobra.ExactArgs(1),

		RunE: func(cmd *cobra.Command, args []string) error {
			tool, err := getTool(args[0])
			if err != nil {
				return err
			}

			return upgradeTool(cmd.Context(), tool)
		},
	}
}

func newRunCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "run <tool> [args...]",
		Short: "Run an installed EdVizion tool",

		// Everything after "run" is passed through without Cobra
		// attempting to interpret flags belonging to the child binary.
		DisableFlagParsing: true,

		Args: cobra.MinimumNArgs(1),

		RunE: func(cmd *cobra.Command, args []string) error {
			tool, err := getTool(args[0])
			if err != nil {
				return err
			}

			path, err := toolPath(tool)
			if err != nil {
				return err
			}

			if _, err := os.Stat(path); err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf(
						"%s is not installed; run `edvizion-cli get %s` first",
						tool.Name,
						tool.Name,
					)
				}

				return err
			}

			if err := ensureLatest(cmd.Context(), tool); err != nil {
				return err
			}

			child := exec.CommandContext(
				cmd.Context(),
				path,
				args[1:]...,
			)

			child.Stdin = os.Stdin
			child.Stdout = os.Stdout
			child.Stderr = os.Stderr

			return child.Run()
		},
	}
}

func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available EdVizion tools",

		RunE: func(cmd *cobra.Command, args []string) error {
			for _, tool := range tools {
				path, err := toolPath(tool)
				if err != nil {
					return err
				}

				status := "not installed"

				if _, err := os.Stat(path); err == nil {
					status = "installed"
				}

				fmt.Printf("%-30s %s\n", tool.Name, status)
			}

			return nil
		},
	}
}

func getTool(name string) (domain.Tool, error) {
	tool, ok := tools[name]
	if !ok {
		return domain.Tool{}, fmt.Errorf("unknown tool %q", name)
	}

	return tool, nil
}

func toolPath(tool domain.Tool) (string, error) {
	dir, err := toolsDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, tool.BinaryName), nil
}

func toolsDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "edvizion", "bin"), nil
}

func installTool(ctx context.Context, tool domain.Tool) error {
	path, err := toolPath(tool)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	fmt.Printf("Getting latest release for %s...\n", tool.Name)

	latestReleaseID, latestVersion, err := releases.GetLatestReleaseID(ctx, tool)
	if err != nil {
		return err
	}

	if latestReleaseID == nil {
		return fmt.Errorf("no release found for %s", tool.Name)
	}

	err = releases.Download(ctx, tool, *latestReleaseID, path)
	if err != nil {
		return err
	}

	fmt.Printf("Installed %s@%s!\n", tool.Name, latestVersion)

	return nil
}

func upgradeTool(ctx context.Context, tool domain.Tool) error {
	path, err := toolPath(tool)
	if err != nil {
		return err
	}

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf(
				"%s is not installed; run `edvizion-cli get %s` first",
				tool.Name,
				tool.Name,
			)
		}

		return err
	}

	currentVersion, err := installedVersion(tool)
	if err != nil {
		return err
	}

	if err := ensureLatest(ctx, tool); err != nil {
		return err
	}

	newVersion, err := installedVersion(tool)
	if err != nil {
		return err
	}

	if currentVersion == newVersion {
		fmt.Printf("%s is already up to date (%s)\n", tool.Name, currentVersion)
	}

	return nil
}

func installedVersion(tool domain.Tool) (string, error) {
	path, err := toolPath(tool)
	if err != nil {
		return "", err
	}

	out, err := exec.Command(path, "--version").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get installed version: %w", err)
	}

	return strings.TrimPrefix(strings.TrimSpace(string(out)), "v"), nil
}

func ensureLatest(ctx context.Context, tool domain.Tool) error {
	path, err := toolPath(tool)
	if err != nil {
		return err
	}

	currentVersion, err := installedVersion(tool)
	if err != nil {
		return err
	}

	latestReleaseID, latestVersion, err := releases.GetLatestReleaseID(ctx, tool)
	if err != nil {
		return err
	}

	if latestReleaseID == nil {
		return fmt.Errorf("no release found for %s", tool.Name)
	}

	if currentVersion == latestVersion {
		return nil
	}

	fmt.Printf(
		"Update available for %s: %s -> %s\n",
		tool.Name,
		currentVersion,
		latestVersion,
	)

	if err := releases.Download(ctx, tool, *latestReleaseID, path); err != nil {
		return fmt.Errorf("failed to upgrade %s: %w", tool.Name, err)
	}

	fmt.Printf("Updated %s to %s!\n", tool.Name, latestVersion)

	return nil
}
