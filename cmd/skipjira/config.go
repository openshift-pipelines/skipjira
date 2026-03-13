package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/theakshaypant/skipjira/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage global configuration",
	Long: `Configure global settings that apply to all repositories.

Per-repository installations can override these settings.`,
	RunE: runConfig,
}

func runConfig(cmd *cobra.Command, args []string) error {
	// Load existing global config
	globalCfg, err := config.LoadGlobal()
	if err != nil {
		return fmt.Errorf("failed to load global config: %w", err)
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Configure global skipjira settings")
	fmt.Println("(Press Enter to keep current value, or type new value)")
	fmt.Println()

	// GitHub token
	if globalCfg.GithubToken != "" {
		fmt.Printf("GitHub token [current: %s...]: ", globalCfg.GithubToken[:10])
	} else {
		fmt.Print("GitHub token: ")
	}
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read GitHub token: %w", err)
	}
	input = strings.TrimSpace(input)
	if input != "" {
		globalCfg.GithubToken = input
	}

	// Jira URL
	if globalCfg.JiraURL != "" {
		fmt.Printf("Jira URL [current: %s]: ", globalCfg.JiraURL)
	} else {
		fmt.Print("Jira URL (e.g., https://jira.company.com): ")
	}
	input, err = reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read Jira URL: %w", err)
	}
	input = strings.TrimSpace(input)
	if input != "" {
		globalCfg.JiraURL = input
	}

	// Jira token
	if globalCfg.JiraToken != "" {
		fmt.Printf("Jira token [current: %s...]: ", globalCfg.JiraToken[:10])
	} else {
		fmt.Print("Jira token: ")
	}
	input, err = reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read Jira token: %w", err)
	}
	input = strings.TrimSpace(input)
	if input != "" {
		globalCfg.JiraToken = input
	}

	// Jira PR field
	if globalCfg.JiraPRField != "" {
		fmt.Printf("Jira PR custom field ID [current: %s]: ", globalCfg.JiraPRField)
	} else {
		fmt.Print("Jira PR custom field ID (e.g., customfield_12345): ")
	}
	input, err = reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read Jira PR field: %w", err)
	}
	input = strings.TrimSpace(input)
	if input != "" {
		globalCfg.JiraPRField = input
	}

	// Save global config
	if err := config.SaveGlobal(globalCfg); err != nil {
		return fmt.Errorf("failed to save global config: %w", err)
	}

	configPath, _ := config.GetGlobalConfigPath()
	fmt.Printf("\n✓ Global configuration saved to %s\n", configPath)
	fmt.Println("\nThese settings will be used by default in all repositories.")
	fmt.Println("Run 'skipjira install' in a repository to use these global settings.")

	return nil
}
