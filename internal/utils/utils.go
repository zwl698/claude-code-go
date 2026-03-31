package utils

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ========================================
// File Utilities
// ========================================

// FileExists checks if a file exists.
// Deprecated: Use PathExists from file.go instead.
func FileExists(path string) bool {
	return PathExists(path)
}

// ReadFileLines reads a file and returns lines.
func ReadFileLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// EnsureDir ensures a directory exists.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// ========================================
// Git Utilities
// ========================================

// IsGitRepo checks if the current directory is a git repository.
func IsGitRepo(path string) bool {
	gitDir := filepath.Join(path, ".git")
	return FileExists(gitDir)
}

// FindGitRoot finds the root of the git repository.
func FindGitRoot(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	for {
		if IsGitRepo(absPath) {
			return absPath, nil
		}
		parent := filepath.Dir(absPath)
		if parent == absPath {
			return "", fmt.Errorf("not a git repository")
		}
		absPath = parent
	}
}

// GetCurrentBranch returns the current git branch.
func GetCurrentBranch(repoPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// GetGitRemote returns the git remote URL.
func GetGitRemote(repoPath string, remote string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", remote)
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// ========================================
// Process Utilities
// ========================================

// RunCommand runs a shell command.
func RunCommand(ctx context.Context, command string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("command failed: %w", err)
	}
	return string(output), nil
}

// RunCommandInDir runs a command in a specific directory.
func RunCommandInDir(ctx context.Context, dir, command string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("command failed: %w", err)
	}
	return string(output), nil
}

// ========================================
// Platform Utilities
// ========================================

// GetPlatform returns the current platform.
func GetPlatform() string {
	return runtime.GOOS + "/" + runtime.GOARCH
}

// IsWindows returns true if running on Windows.
func IsWindows() bool {
	return runtime.GOOS == "windows"
}

// IsMacOS returns true if running on macOS.
func IsMacOS() bool {
	return runtime.GOOS == "darwin"
}

// IsLinux returns true if running on Linux.
func IsLinux() bool {
	return runtime.GOOS == "linux"
}

// ========================================
// Environment Utilities
// ========================================

// GetEnv returns an environment variable with a default value.
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetHomeDir returns the user's home directory.
func GetHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

// GetCwd returns the current working directory.
func GetCwd() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return cwd
}

// ========================================
// String Utilities
// ========================================

// TruncateString truncates a string to a maximum length.
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// Plural returns the plural form of a word if count is not 1.
func Plural(count int, singular string) string {
	if count == 1 {
		return singular
	}
	return singular + "s"
}

// Indent indents each line of a string.
func Indent(s string, indent string) string {
	var result strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		result.WriteString(indent)
		result.WriteString(scanner.Text())
		result.WriteString("\n")
	}
	return result.String()
}

// ========================================
// Slice Utilities
// ========================================

// Contains returns true if a slice contains an element.
func Contains[T comparable](slice []T, element T) bool {
	for _, e := range slice {
		if e == element {
			return true
		}
	}
	return false
}

// Unique returns unique elements from a slice.
func Unique[T comparable](slice []T) []T {
	seen := make(map[T]bool)
	result := []T{}
	for _, e := range slice {
		if !seen[e] {
			seen[e] = true
			result = append(result, e)
		}
	}
	return result
}

// Filter returns elements that match a predicate.
func Filter[T any](slice []T, predicate func(T) bool) []T {
	result := []T{}
	for _, e := range slice {
		if predicate(e) {
			result = append(result, e)
		}
	}
	return result
}

// Map applies a function to each element.
func Map[T, U any](slice []T, fn func(T) U) []U {
	result := make([]U, len(slice))
	for i, e := range slice {
		result[i] = fn(e)
	}
	return result
}

// ========================================
// Map Utilities
// ========================================

// Keys returns the keys of a map.
func Keys[K comparable, V any](m map[K]V) []K {
	result := make([]K, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}

// Values returns the values of a map.
func Values[K comparable, V any](m map[K]V) []V {
	result := make([]V, 0, len(m))
	for _, v := range m {
		result = append(result, v)
	}
	return result
}
