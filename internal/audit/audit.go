package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogLevel represents the severity of an audit log entry
type LogLevel string

const (
	LevelInfo  LogLevel = "INFO"
	LevelWarn  LogLevel = "WARN"
	LevelError LogLevel = "ERROR"
)

// Action represents the type of operation being logged
type Action string

const (
	ActionQueueAdd    Action = "QUEUE_ADD"
	ActionSyncStart   Action = "SYNC_START"
	ActionSyncEnd     Action = "SYNC_END"
	ActionPRFetch     Action = "PR_FETCH"
	ActionJiraUpdate  Action = "JIRA_UPDATE"
	ActionEntryUpdate Action = "ENTRY_UPDATE"
)

// Entry represents a single audit log entry
type Entry struct {
	Timestamp time.Time         `json:"timestamp"`
	Level     LogLevel          `json:"level"`
	Action    Action            `json:"action"`
	Message   string            `json:"message"`
	Details   map[string]string `json:"details,omitempty"`
}

var mu sync.Mutex

// GetAuditLogPath returns the global audit log file path
func GetAuditLogPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	auditDir := filepath.Join(home, ".config", "skipjira")
	if err := os.MkdirAll(auditDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create audit directory: %w", err)
	}

	return filepath.Join(auditDir, "audit.jsonl"), nil
}

// Log appends an entry to the audit log
func Log(level LogLevel, action Action, message string, details map[string]string) error {
	entry := &Entry{
		Timestamp: time.Now(),
		Level:     level,
		Action:    action,
		Message:   message,
		Details:   details,
	}

	return WriteEntry(entry)
}

// WriteEntry appends an entry to the audit log
func WriteEntry(entry *Entry) error {
	mu.Lock()
	defer mu.Unlock()

	auditPath, err := GetAuditLogPath()
	if err != nil {
		return err
	}

	f, err := os.OpenFile(auditPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open audit log file: %w", err)
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	if err := encoder.Encode(entry); err != nil {
		return fmt.Errorf("failed to write entry to audit log: %w", err)
	}

	return nil
}

// Info logs an informational message
func Info(action Action, message string, details map[string]string) error {
	return Log(LevelInfo, action, message, details)
}

// Warn logs a warning message
func Warn(action Action, message string, details map[string]string) error {
	return Log(LevelWarn, action, message, details)
}

// Error logs an error message
func Error(action Action, message string, details map[string]string) error {
	return Log(LevelError, action, message, details)
}
