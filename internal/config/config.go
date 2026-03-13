package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents repository configuration
type Config struct {
	GithubToken  string `yaml:"github_token,omitempty"`
	JiraURL      string `yaml:"jira_url,omitempty"`
	JiraToken    string `yaml:"jira_token,omitempty"`
	JiraPRField  string `yaml:"jira_pr_field,omitempty"`
	RepoOwner    string `yaml:"repo_owner"`
	RepoName     string `yaml:"repo_name"`
}

// GlobalConfig represents global configuration (shared across repos)
type GlobalConfig struct {
	GithubToken string `yaml:"github_token,omitempty"`
	JiraURL     string `yaml:"jira_url,omitempty"`
	JiraToken   string `yaml:"jira_token,omitempty"`
	JiraPRField string `yaml:"jira_pr_field,omitempty"`
}

// GetGlobalConfigPath returns the global config file path
func GetGlobalConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".skipjira")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return filepath.Join(configDir, "config.yaml"), nil
}

// LoadGlobal reads global config from ~/.skipjira/config.yaml
func LoadGlobal() (*GlobalConfig, error) {
	configPath, err := GetGlobalConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Global config doesn't exist, return empty config
			return &GlobalConfig{}, nil
		}
		return nil, fmt.Errorf("failed to read global config: %w", err)
	}

	var cfg GlobalConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse global config: %w", err)
	}

	return &cfg, nil
}

// SaveGlobal writes global config to ~/.skipjira/config.yaml
func SaveGlobal(cfg *GlobalConfig) error {
	configPath, err := GetGlobalConfigPath()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal global config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write global config: %w", err)
	}

	return nil
}

// Load reads config from .git/skipjira-config.yaml and merges with global config
func Load(gitRoot string) (*Config, error) {
	// Load global config first
	globalCfg, err := LoadGlobal()
	if err != nil {
		return nil, fmt.Errorf("failed to load global config: %w", err)
	}

	// Load local config
	configPath := filepath.Join(gitRoot, ".git", "skipjira-config.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Merge: local config overrides global config
	if cfg.GithubToken == "" {
		cfg.GithubToken = globalCfg.GithubToken
	}
	if cfg.JiraURL == "" {
		cfg.JiraURL = globalCfg.JiraURL
	}
	if cfg.JiraToken == "" {
		cfg.JiraToken = globalCfg.JiraToken
	}
	if cfg.JiraPRField == "" {
		cfg.JiraPRField = globalCfg.JiraPRField
	}

	// Validate required fields
	if cfg.GithubToken == "" {
		return nil, fmt.Errorf("github_token is required (set in global or local config)")
	}
	if cfg.JiraURL == "" {
		return nil, fmt.Errorf("jira_url is required (set in global or local config)")
	}
	if cfg.JiraToken == "" {
		return nil, fmt.Errorf("jira_token is required (set in global or local config)")
	}
	if cfg.JiraPRField == "" {
		return nil, fmt.Errorf("jira_pr_field is required (set in global or local config)")
	}
	if cfg.RepoOwner == "" {
		return nil, fmt.Errorf("repo_owner is required in local config")
	}
	if cfg.RepoName == "" {
		return nil, fmt.Errorf("repo_name is required in local config")
	}

	return &cfg, nil
}

// Save writes config to .git/skipjira-config.yaml
func Save(gitRoot string, cfg *Config) error {
	configPath := filepath.Join(gitRoot, ".git", "skipjira-config.yaml")

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// CreateTemplate creates a config template with empty values
func CreateTemplate(gitRoot string) error {
	cfg := &Config{
		GithubToken:  "",
		JiraURL:      "",
		JiraToken:    "",
		JiraPRField:  "",
		RepoOwner:    "",
		RepoName:     "",
	}
	return Save(gitRoot, cfg)
}
