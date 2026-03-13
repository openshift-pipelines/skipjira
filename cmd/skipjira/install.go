package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/theakshaypant/skipjira/internal/config"
	"github.com/theakshaypant/skipjira/internal/git"
	"github.com/theakshaypant/skipjira/scripts"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install pre-push hook in current repository",
	Long: `Install the skipjira pre-push hook in the current git repository.

This will:
1. Create .git/skipjira-config.yaml with your credentials
2. Install .git/hooks/pre-push hook
3. Make the hook executable

You'll be prompted for:
- GitHub token
- Jira URL
- Jira token
- Jira PR custom field ID`,
	RunE: runInstall,
}

func runInstall(cmd *cobra.Command, args []string) error {
	gitRoot, err := git.GetRoot()
	if err != nil {
		return fmt.Errorf("not a git repository: %w", err)
	}

	remoteURL, err := git.GetRemoteURL()
	if err != nil {
		return fmt.Errorf("failed to get remote URL: %w", err)
	}

	owner, repo, err := git.ParseRepoFromURL(remoteURL)
	if err != nil {
		return fmt.Errorf("failed to parse repository from remote URL: %w", err)
	}

	fmt.Printf("Detected repository: %s/%s\n\n", owner, repo)

	cfg, err := generateConfig(owner, repo)
	if err != nil {
		return fmt.Errorf("failed to generate config: %w", err)
	}

	if err := config.Save(gitRoot, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println("\n✓ Configuration saved to .git/skipjira-config.yaml")

	hookPath := gitRoot + "/.git/hooks/pre-push"

	if err := os.WriteFile(hookPath, []byte(scripts.PrePushHook), 0755); err != nil {
		return fmt.Errorf("failed to write pre-push hook: %w", err)
	}

	fmt.Println("✓ Pre-push hook installed at .git/hooks/pre-push")
	fmt.Println("\nSetup complete!")

	return nil
}

func generateConfig(owner, repo string) (*config.Config, error) {
	globalCfg, err := config.LoadGlobal()
	if err != nil {
		return nil, fmt.Errorf("failed to load global config: %w", err)
	}

	reader := bufio.NewReader(os.Stdin)
	prompt := func(label, hint, placeholder string) (string, error) {
		return promptWithDefault(reader, label, hint, placeholder)
	}

	githubToken, err := prompt("GitHub token", globalCfg.GithubToken, "")
	if err != nil {
		return nil, fmt.Errorf("failed to read GitHub token: %w", err)
	}

	jiraURL, err := prompt("Jira URL", globalCfg.JiraURL, "https://jira.company.com")
	if err != nil {
		return nil, fmt.Errorf("failed to read Jira URL: %w", err)
	}

	jiraToken, err := prompt("Jira token", globalCfg.JiraToken, "")
	if err != nil {
		return nil, fmt.Errorf("failed to read Jira token: %w", err)
	}

	jiraPRField, err := prompt("Jira PR custom field ID", globalCfg.JiraPRField, "customfield_12345")
	if err != nil {
		return nil, fmt.Errorf("failed to read Jira PR field: %w", err)
	}

	return &config.Config{
		GithubToken: githubToken,
		JiraURL:     jiraURL,
		JiraToken:   jiraToken,
		JiraPRField: jiraPRField,
		RepoOwner:   owner,
		RepoName:    repo,
	}, nil
}

func promptWithDefault(r *bufio.Reader, label, globalVal, placeholder string) (string, error) {
	if globalVal != "" {
		shown := globalVal
		if len(shown) > 16 && !strings.HasPrefix(shown, "http") && !strings.HasPrefix(shown, "customfield") {
			shown = shown[:8] + "…"
		}
		fmt.Printf("%s [using global: %s]: ", label, shown)
	} else if placeholder != "" {
		fmt.Printf("%s (e.g., %s): ", label, placeholder)
	} else {
		fmt.Printf("%s: ", label)
	}

	input, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}
