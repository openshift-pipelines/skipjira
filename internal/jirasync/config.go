package jirasync

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents jirasync configuration
type Config struct {
	Repositories []Repository `yaml:"repositories"`
}

// Repository represents a GitHub repository to monitor
type Repository struct {
	Owner string `yaml:"owner"`
	Name  string `yaml:"name"`
}

// LoadConfig loads repository list from YAML file
func LoadConfig(configFile string) (*Config, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configFile, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate
	if len(cfg.Repositories) == 0 {
		return nil, fmt.Errorf("no repositories configured")
	}

	for i, repo := range cfg.Repositories {
		if repo.Owner == "" {
			return nil, fmt.Errorf("repository %d: owner is required", i)
		}
		if repo.Name == "" {
			return nil, fmt.Errorf("repository %d: name is required", i)
		}
	}

	return &cfg, nil
}

// ParseDate parses a date string in multiple formats
// Supports: "2006-01-02" (YYYY-MM-DD) and "DD/MM/YYYY"
func ParseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty date string")
	}

	// Try ISO format first (2006-01-02)
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}

	// Try DD/MM/YYYY format
	if t, err := time.Parse("02/01/2006", s); err == nil {
		return t, nil
	}

	// Try DD-MM-YYYY format
	if t, err := time.Parse("02-01-2006", s); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("invalid date format: %s (expected: YYYY-MM-DD or DD/MM/YYYY)", s)
}
