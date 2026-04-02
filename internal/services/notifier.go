package services

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strings"
)

// NotificationOptions represents options for a notification
type NotificationOptions struct {
	Message          string `json:"message"`
	Title            string `json:"title,omitempty"`
	NotificationType string `json:"notification_type"`
}

// TerminalNotification provides terminal notification methods
type TerminalNotification interface {
	NotifyITerm2(opts NotificationOptions)
	NotifyKitty(opts KittyNotificationOptions)
	NotifyGhostty(opts NotificationOptions)
	NotifyBell()
}

// KittyNotificationOptions represents options for Kitty terminal notifications
type KittyNotificationOptions struct {
	NotificationOptions
	ID int `json:"id"`
}

const defaultTitle = "Claude Code"

// SendNotification sends a notification through the configured channel
func SendNotification(notif NotificationOptions, terminal TerminalNotification, preferredChannel string) error {
	methodUsed := sendToChannel(preferredChannel, notif, terminal)

	// TODO: Log analytics event
	_ = methodUsed

	return nil
}

func sendToChannel(channel string, opts NotificationOptions, terminal TerminalNotification) string {
	title := opts.Title
	if title == "" {
		title = defaultTitle
	}

	switch channel {
	case "auto":
		return sendAuto(opts, terminal)
	case "iterm2":
		terminal.NotifyITerm2(opts)
		return "iterm2"
	case "iterm2_with_bell":
		terminal.NotifyITerm2(opts)
		terminal.NotifyBell()
		return "iterm2_with_bell"
	case "kitty":
		terminal.NotifyKitty(KittyNotificationOptions{
			NotificationOptions: opts,
			ID:                  generateKittyID(),
		})
		return "kitty"
	case "ghostty":
		terminal.NotifyGhostty(opts)
		return "ghostty"
	case "terminal_bell":
		terminal.NotifyBell()
		return "terminal_bell"
	case "notifications_disabled":
		return "disabled"
	default:
		return "none"
	}
}

func sendAuto(opts NotificationOptions, terminal TerminalNotification) string {
	terminalEnv := os.Getenv("TERM_PROGRAM")

	switch terminalEnv {
	case "Apple_Terminal":
		bellDisabled := isAppleTerminalBellDisabled()
		if bellDisabled {
			terminal.NotifyBell()
			return "terminal_bell"
		}
		return "no_method_available"
	case "iTerm.app":
		terminal.NotifyITerm2(opts)
		return "iterm2"
	case "kitty":
		terminal.NotifyKitty(KittyNotificationOptions{
			NotificationOptions: opts,
			ID:                  generateKittyID(),
		})
		return "kitty"
	case "ghostty":
		terminal.NotifyGhostty(opts)
		return "ghostty"
	default:
		return "no_method_available"
	}
}

func generateKittyID() int {
	return rand.Intn(10000)
}

func isAppleTerminalBellDisabled() bool {
	// Check if we're in Apple Terminal
	terminalEnv := os.Getenv("TERM_PROGRAM")
	if terminalEnv != "Apple_Terminal" {
		return false
	}

	// Get current Terminal profile name
	cmd := exec.Command("osascript", "-e", `tell application "Terminal" to name of current settings of front window`)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	currentProfile := strings.TrimSpace(string(output))
	if currentProfile == "" {
		return false
	}

	// Get Terminal preferences
	cmd = exec.Command("defaults", "export", "com.apple.Terminal", "-")
	output, err = cmd.Output()
	if err != nil {
		return false
	}

	// Parse plist to check if bell is disabled
	// This is a simplified check - in production, you'd want to use a proper plist parser
	plistContent := string(output)
	if !strings.Contains(plistContent, currentProfile) {
		return false
	}

	// Look for Bell = 0 or Bell = false in the profile
	// This is a naive implementation - proper plist parsing would be better
	profileSection := extractProfileSection(plistContent, currentProfile)
	if profileSection == "" {
		return false
	}

	return strings.Contains(profileSection, "Bell = 0") ||
		strings.Contains(profileSection, "Bell = false")
}

func extractProfileSection(plistContent, profileName string) string {
	// Naive extraction - look for the profile section
	// In production, use a proper plist parser
	lines := strings.Split(plistContent, "\n")
	inProfile := false
	var profileLines []string

	for _, line := range lines {
		if strings.Contains(line, profileName) {
			inProfile = true
		}
		if inProfile {
			profileLines = append(profileLines, line)
			if strings.Contains(line, "};") || strings.Contains(line, "}") {
				break
			}
		}
	}

	return strings.Join(profileLines, "\n")
}

// DefaultTerminalNotification provides basic terminal notification implementations
type DefaultTerminalNotification struct{}

func (d *DefaultTerminalNotification) NotifyITerm2(opts NotificationOptions) {
	// iTerm2 escape sequence for notifications
	title := opts.Title
	if title == "" {
		title = defaultTitle
	}
	fmt.Printf("\x1B]9;%s\x07", opts.Message)
}

func (d *DefaultTerminalNotification) NotifyKitty(opts KittyNotificationOptions) {
	// Kitty terminal notification escape sequence
	title := opts.Title
	if title == "" {
		title = defaultTitle
	}
	fmt.Printf("\x1B]99;i=%d:d=0:p=title;%s\x1B\\%s\x1B\\\n", opts.ID, title, opts.Message)
}

func (d *DefaultTerminalNotification) NotifyGhostty(opts NotificationOptions) {
	// Ghostty terminal notification
	title := opts.Title
	if title == "" {
		title = defaultTitle
	}
	fmt.Printf("\x1B]777;notify;%s;%s\x1B\\", title, opts.Message)
}

func (d *DefaultTerminalNotification) NotifyBell() {
	// Terminal bell
	fmt.Print("\x07")
}
