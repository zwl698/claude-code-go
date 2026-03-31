package utils

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// MaxOutputSize is the maximum output size in bytes.
const MaxOutputSize = 0.25 * 1024 * 1024 // 0.25MB

// PathExists checks if a path exists.
func PathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsDir checks if a path is a directory.
func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// IsFile checks if a path is a regular file.
func IsFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// FileSize returns the size of a file.
func FileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// ReadFileSafe reads a file and returns nil on error.
func ReadFileSafe(filepath string) string {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return ""
	}
	return string(content)
}

// WriteFileSync writes content to a file atomically.
func WriteFileSync(filepath string, content string, perm ...fs.FileMode) error {
	mode := os.FileMode(0644)
	if len(perm) > 0 {
		mode = perm[0]
	}

	// Create temp file
	tempPath := filepath + ".tmp." + GenerateShortID()

	// Write to temp file
	err := os.WriteFile(tempPath, []byte(content), mode)
	if err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Rename atomically
	err = os.Rename(tempPath, filepath)
	if err != nil {
		_ = os.Remove(tempPath) // Cleanup temp file
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// GetDisplayPath returns a user-friendly path.
func GetDisplayPath(filePath string) string {
	cwd, _ := os.Getwd()
	homeDir, _ := os.UserHomeDir()

	// Use relative path if in current working directory
	if cwd != "" {
		if relPath, err := filepath.Rel(cwd, filePath); err == nil {
			if !strings.HasPrefix(relPath, "..") {
				return relPath
			}
		}
	}

	// Use tilde notation for home directory
	if homeDir != "" && strings.HasPrefix(filePath, homeDir+string(os.PathSeparator)) {
		return "~" + filePath[len(homeDir):]
	}

	return filePath
}

// IsENOENT checks if an error is a "file not found" error.
func IsENOENT(err error) bool {
	if err == nil {
		return false
	}
	return os.IsNotExist(err)
}

// IsDirEmpty checks if a directory is empty.
func IsDirEmpty(dirPath string) bool {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return IsENOENT(err)
	}
	return len(entries) == 0
}

// FindSimilarFile finds files with the same name but different extensions.
func FindSimilarFile(filePath string) (string, error) {
	dir := filepath.Dir(filePath)
	base := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		name := entry.Name()
		entryBase := strings.TrimSuffix(name, filepath.Ext(name))
		if entryBase == base && name != filepath.Base(filePath) {
			return name, nil
		}
	}

	return "", nil
}

// AddLineNumbers adds line numbers to content.
func AddLineNumbers(content string, startLine int) string {
	if content == "" {
		return ""
	}

	lines := strings.Split(content, "\n")
	result := make([]string, len(lines))

	for i, line := range lines {
		// Use compact format: N\t
		result[i] = fmt.Sprintf("%d\t%s", i+startLine, line)
	}

	return strings.Join(result, "\n")
}

// StripLineNumberPrefix removes line number prefix from a line.
var lineNumberRegex = regexp.MustCompile(`^\s*\d+[\t](.*)$`)

func StripLineNumberPrefix(line string) string {
	match := lineNumberRegex.FindStringSubmatch(line)
	if match != nil {
		return match[1]
	}
	return line
}

// ConvertLeadingTabsToSpaces converts leading tabs to spaces.
func ConvertLeadingTabsToSpaces(content string) string {
	if !strings.Contains(content, "\t") {
		return content
	}
	return regexp.MustCompile(`(?m)^\t+`).ReplaceAllStringFunc(content, func(match string) string {
		return strings.Repeat("  ", len(match))
	})
}

// DetectLineEndings detects the line ending type of content.
type LineEndingType string

const (
	LineEndingLF   LineEndingType = "LF"
	LineEndingCRLF LineEndingType = "CRLF"
)

func DetectLineEndings(content string) LineEndingType {
	if strings.Contains(content, "\r\n") {
		return LineEndingCRLF
	}
	return LineEndingLF
}

// NormalizeLineEndings normalizes line endings to LF.
func NormalizeLineEndings(content string) string {
	return strings.ReplaceAll(content, "\r\n", "\n")
}

// IsFileWithinReadSizeLimit checks if a file is within the read size limit.
func IsFileWithinReadSizeLimit(filePath string, maxSize ...int64) bool {
	max := int64(MaxOutputSize)
	if len(maxSize) > 0 {
		max = maxSize[0]
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return false
	}
	return info.Size() <= max
}

// PathsEqual compares two paths for equality, handling platform differences.
func PathsEqual(path1, path2 string) bool {
	return filepath.Clean(path1) == filepath.Clean(path2)
}

// ExpandPath expands ~ and environment variables in a path.
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, _ := os.UserHomeDir()
		if homeDir != "" {
			path = filepath.Join(homeDir, path[2:])
		}
	}
	return os.ExpandEnv(path)
}
