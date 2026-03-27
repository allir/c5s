package claude

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// TmuxAvailable returns true if tmux is installed and we're inside a tmux session.
func TmuxAvailable() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

// FindTmuxPane finds the tmux pane ID for a given PID by mapping PID → TTY → pane.
// Returns empty string if the PID is not in a tmux pane.
func FindTmuxPane(pid int) string {
	// Get the TTY for the process
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "tty=").Output()
	if err != nil {
		return ""
	}
	tty := strings.TrimSpace(string(out))
	if tty == "" || tty == "?" {
		return ""
	}

	// Normalize TTY to /dev/ path
	ttyPath := "/dev/" + tty

	// Map TTY to tmux pane ID
	out, err = exec.Command("tmux", "list-panes", "-a", "-F", "#{pane_id} #{pane_tty}").Output()
	if err != nil {
		return ""
	}

	for line := range strings.SplitSeq(string(out), "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), " ", 2)
		if len(parts) == 2 && parts[1] == ttyPath {
			return parts[0]
		}
	}

	return ""
}

// SendTmuxKeys sends text followed by Enter to a tmux pane.
// The message is sent as literal keys, not through the shell.
func SendTmuxKeys(paneID, message string) error {
	if paneID == "" {
		return fmt.Errorf("no tmux pane")
	}
	return exec.Command("tmux", "send-keys", "-t", paneID, message, "Enter").Run()
}
