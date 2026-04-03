package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// =============================================================================
// Grep Search Utilities
// =============================================================================

// GrepSearchOptions represents options for grep search.
type GrepSearchOptions struct {
	Pattern         string
	Path            string
	CaseInsensitive bool
	Multiline       bool
	FileTypes       []string
	ExcludeGlobs    []string
	MaxResults      int
}

// GrepMatch represents a grep match.
type GrepMatch struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Content string `json:"content"`
	Match   string `json:"match"`
}

// ExecuteGrepSearch executes a grep search.
func ExecuteGrepSearch(ctx context.Context, opts GrepSearchOptions) ([]GrepMatch, error) {
	// Validate pattern
	if opts.Pattern == "" {
		return nil, fmt.Errorf("pattern is required")
	}

	// Set default path
	searchPath := opts.Path
	if searchPath == "" {
		searchPath = "."
	}

	// Resolve path
	if !filepath.IsAbs(searchPath) {
		wd, _ := os.Getwd()
		searchPath = filepath.Join(wd, searchPath)
	}

	// Build regex
	flags := ""
	if opts.CaseInsensitive {
		flags = "(?i)"
	}
	patternStr := flags + opts.Pattern

	pattern, err := regexp.Compile(patternStr)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	// Search files
	var matches []GrepMatch
	err = filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip directories
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check file type
		if len(opts.FileTypes) > 0 {
			ext := strings.TrimPrefix(filepath.Ext(path), ".")
			if !isExpectedFileType(ext, opts.FileTypes) {
				return nil
			}
		}

		// Search in file
		fileMatches, err := searchInFile(path, pattern)
		if err != nil {
			return nil
		}
		matches = append(matches, fileMatches...)

		// Check max results
		if opts.MaxResults > 0 && len(matches) >= opts.MaxResults {
			return filepath.SkipAll
		}

		return nil
	})

	return matches, err
}

// isExpectedFileType checks if the file extension matches expected types.
func isExpectedFileType(ext string, expectedTypes []string) bool {
	typeMap := map[string][]string{
		"go":    {"go"},
		"js":    {"js", "jsx", "mjs", "cjs"},
		"ts":    {"ts", "tsx", "mts", "cts"},
		"py":    {"py", "pyw", "pyi"},
		"java":  {"java"},
		"rust":  {"rs"},
		"c":     {"c", "h"},
		"cpp":   {"cpp", "cc", "cxx", "hpp", "hh", "hxx"},
		"rb":    {"rb", "rake"},
		"php":   {"php", "phtml"},
		"swift": {"swift"},
		"kt":    {"kt", "kts"},
		"scala": {"scala", "sc"},
	}

	for _, expectedType := range expectedTypes {
		extensions, ok := typeMap[expectedType]
		if !ok {
			if ext == expectedType {
				return true
			}
			continue
		}
		for _, e := range extensions {
			if ext == e {
				return true
			}
		}
	}
	return false
}

// searchInFile searches for pattern matches in a file.
func searchInFile(path string, pattern *regexp.Regexp) ([]GrepMatch, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var matches []GrepMatch
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if pattern.MatchString(line) {
			matches = append(matches, GrepMatch{
				File:    path,
				Line:    lineNum,
				Content: line,
				Match:   pattern.FindString(line),
			})
		}
	}

	return matches, scanner.Err()
}

// =============================================================================
// Glob Search Utilities
// =============================================================================

// GlobSearchOptions represents options for glob search.
type GlobSearchOptions struct {
	Pattern       string
	Path          string
	IgnoreGlobs   []string
	IncludeHidden bool
}

// GlobMatch represents a glob match.
type GlobMatch struct {
	Path string
	Info os.FileInfo
}

// ExecuteGlobSearch executes a glob search.
func ExecuteGlobSearch(opts GlobSearchOptions) ([]GlobMatch, error) {
	if opts.Pattern == "" {
		return nil, fmt.Errorf("pattern is required")
	}

	searchPath := opts.Path
	if searchPath == "" {
		searchPath = "."
	}

	// Resolve path
	if !filepath.IsAbs(searchPath) {
		wd, _ := os.Getwd()
		searchPath = filepath.Join(wd, searchPath)
	}

	// Execute glob
	pattern := filepath.Join(searchPath, opts.Pattern)
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid glob pattern: %w", err)
	}

	var matches []GlobMatch
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		// Filter hidden files
		if !opts.IncludeHidden && strings.HasPrefix(filepath.Base(file), ".") {
			continue
		}

		matches = append(matches, GlobMatch{
			Path: file,
			Info: info,
		})
	}

	return matches, nil
}

// =============================================================================
// Web Fetch Utilities
// =============================================================================

// WebFetchOptions represents options for web fetch.
type WebFetchOptions struct {
	URLs      []string
	Timeout   int
	MaxSize   int
	UserAgent string
}

// WebFetchResult represents the result of web fetch.
type WebFetchResult struct {
	URL     string `json:"url"`
	Content string `json:"content"`
	Error   string `json:"error,omitempty"`
}

// Note: Actual HTTP fetching should use net/http package
// This is a placeholder for the utility structure

// =============================================================================
// Search Utilities
// =============================================================================

// SearchFiles searches for files matching a pattern.
func SearchFiles(root, pattern string, caseInsensitive bool) ([]string, error) {
	var matches []string

	re, err := regexp.Compile("(?i)" + pattern)
	if !caseInsensitive {
		re, err = regexp.Compile(pattern)
	}
	if err != nil {
		return nil, err
	}

	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && re.MatchString(info.Name()) {
			matches = append(matches, path)
		}
		return nil
	})

	return matches, err
}

// SearchContent searches for content in files.
func SearchContent(root, pattern string, caseInsensitive bool) (map[string][]int, error) {
	results := make(map[string][]int)

	flags := ""
	if caseInsensitive {
		flags = "(?i)"
	}
	re, err := regexp.Compile(flags + pattern)
	if err != nil {
		return nil, err
	}

	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNum := 0
		var matchedLines []int

		for scanner.Scan() {
			lineNum++
			if re.MatchString(scanner.Text()) {
				matchedLines = append(matchedLines, lineNum)
			}
		}

		if len(matchedLines) > 0 {
			results[path] = matchedLines
		}

		return nil
	})

	return results, err
}

// =============================================================================
// Path Utilities
// =============================================================================

// IsBinaryFile checks if a file is binary.
func IsBinaryFile(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil {
		return false, err
	}

	// Check for null bytes (common in binary files)
	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return true, nil
		}
	}

	return false, nil
}

// GetFileExtension returns the file extension without dot.
func GetFileExtension(path string) string {
	ext := filepath.Ext(path)
	return strings.TrimPrefix(ext, ".")
}

// ShouldIndexFile determines if a file should be indexed for search.
func ShouldIndexFile(path string) bool {
	// Skip common non-indexable patterns
	patterns := []string{
		"node_modules", "vendor", ".git", ".svn", ".hg",
		"dist", "build", "target", "out", "bin",
		".idea", ".vscode", ".settings",
	}

	for _, p := range patterns {
		if strings.Contains(path, p) {
			return false
		}
	}

	// Skip binary extensions
	binaryExts := []string{
		"exe", "dll", "so", "dylib", "a", "o", "obj",
		"png", "jpg", "jpeg", "gif", "bmp", "ico", "svg",
		"pdf", "doc", "docx", "xls", "xlsx", "ppt", "pptx",
		"zip", "tar", "gz", "bz2", "7z", "rar",
		"mp3", "mp4", "avi", "mov", "mkv", "flv",
	}

	ext := GetFileExtension(path)
	for _, be := range binaryExts {
		if ext == be {
			return false
		}
	}

	return true
}
