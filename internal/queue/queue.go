package queue

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/theakshaypant/skipjira/internal/audit"
)

// Entry represents a pending or failed PR-to-Jira link operation
// Successful entries are removed from the queue and logged in the audit log
type Entry struct {
	Branch    string    `json:"branch"`
	JiraIDs   []string  `json:"jira_ids"`
	Timestamp time.Time `json:"timestamp"`
	RepoOwner string    `json:"repo_owner"`
	RepoName  string    `json:"repo_name"`
	GitRoot   string    `json:"git_root"`
	PRURL     string    `json:"pr_url,omitempty"`
	Error     string    `json:"error,omitempty"`
}

var mu sync.Mutex

// GetQueuePath returns the global queue file path
func GetQueuePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	queueDir := filepath.Join(home, ".config", "skipjira")
	if err := os.MkdirAll(queueDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create queue directory: %w", err)
	}

	return filepath.Join(queueDir, "queue.jsonl"), nil
}

// Add appends an entry to the queue
func Add(entry *Entry) error {
	mu.Lock()
	defer mu.Unlock()

	queuePath, err := GetQueuePath()
	if err != nil {
		return err
	}

	f, err := os.OpenFile(queuePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open queue file: %w", err)
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	if err := encoder.Encode(entry); err != nil {
		return fmt.Errorf("failed to write entry to queue: %w", err)
	}

	// Log to audit
	audit.Info(audit.ActionQueueAdd, "Entry added to queue", map[string]string{
		"branch":     entry.Branch,
		"jira_ids":   fmt.Sprintf("%v", entry.JiraIDs),
		"repo_owner": entry.RepoOwner,
		"repo_name":  entry.RepoName,
	})

	return nil
}

// ReadAll reads all entries from the queue
func ReadAll() ([]Entry, error) {
	mu.Lock()
	defer mu.Unlock()

	queuePath, err := GetQueuePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(queuePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, fmt.Errorf("failed to read queue file: %w", err)
	}

	if len(data) == 0 {
		return []Entry{}, nil
	}

	var entries []Entry
	decoder := json.NewDecoder(bytes.NewReader(data))
	for decoder.More() {
		var entry Entry
		if err := decoder.Decode(&entry); err != nil {
			return nil, fmt.Errorf("failed to decode entry: %w", err)
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// WriteAll overwrites the queue with new entries
func WriteAll(entries []Entry) error {
	mu.Lock()
	defer mu.Unlock()

	queuePath, err := GetQueuePath()
	if err != nil {
		return err
	}

	f, err := os.OpenFile(queuePath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open queue file: %w", err)
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	for _, entry := range entries {
		if err := encoder.Encode(&entry); err != nil {
			return fmt.Errorf("failed to write entry: %w", err)
		}
	}

	return nil
}

