// Package claude provides types and discovery for Claude Code sessions.
package claude

import "time"

// Status represents the current state of a Claude Code session.
type Status string

const (
	StatusWorking  Status = "working"
	StatusIdle     Status = "idle"
	StatusInput    Status = "input"
	StatusFinished Status = "finished"
	StatusUnknown  Status = "unknown"
)

// String returns the display string for a status.
func (s Status) String() string {
	return string(s)
}

// Session represents a discovered Claude Code session from the filesystem.
type Session struct {
	ID              string
	PID             int
	Summary         string
	Project         string // derived from cwd directory name
	Cwd             string
	GitBranch       string
	Model           string
	JSONLPath       string // path to the session transcript file
	StartedAt       time.Time
	LastModified    time.Time
	Status          Status
	PendingApproval *PendingApproval // non-nil when a tool approval is waiting
}
