package utils

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// =============================================================================
// Git Types
// =============================================================================

// GitStatus represents the status of a git repository.
type GitStatus struct {
	IsRepo          bool     `json:"isRepo"`
	Branch          string   `json:"branch"`
	Remote          string   `json:"remote,omitempty"`
	Ahead           int      `json:"ahead,omitempty"`
	Behind          int      `json:"behind,omitempty"`
	HasUncommitted  bool     `json:"hasUncommitted"`
	HasUnstaged     bool     `json:"hasUnstaged"`
	HasUntracked    bool     `json:"hasUntracked"`
	ModifiedFiles   []string `json:"modifiedFiles,omitempty"`
	StagedFiles     []string `json:"stagedFiles,omitempty"`
	UntrackedFiles  []string `json:"untrackedFiles,omitempty"`
	ConflictedFiles []string `json:"conflictedFiles,omitempty"`
	HeadCommit      string   `json:"headCommit,omitempty"`
	HeadMessage     string   `json:"headMessage,omitempty"`
}

// GitDiff represents a diff between commits or working tree.
type GitDiff struct {
	Files       []GitDiffFile `json:"files"`
	TotalAdd    int           `json:"totalAdd"`
	TotalDelete int           `json:"totalDelete"`
}

// GitDiffFile represents a diff for a single file.
type GitDiffFile struct {
	Path      string        `json:"path"`
	Status    string        `json:"status"` // added, modified, deleted, renamed
	Additions int           `json:"additions"`
	Deletions int           `json:"deletions"`
	Hunks     []GitDiffHunk `json:"hunks,omitempty"`
}

// GitDiffHunk represents a diff hunk.
type GitDiffHunk struct {
	Header   string   `json:"header"`
	OldStart int      `json:"oldStart"`
	OldLines int      `json:"oldLines"`
	NewStart int      `json:"newStart"`
	NewLines int      `json:"newLines"`
	Lines    []string `json:"lines,omitempty"`
}

// GitLogEntry represents a log entry.
type GitLogEntry struct {
	Hash      string   `json:"hash"`
	ShortHash string   `json:"shortHash"`
	Author    string   `json:"author"`
	Email     string   `json:"email"`
	Date      string   `json:"date"`
	Message   string   `json:"message"`
	Parents   []string `json:"parents,omitempty"`
	Refs      []string `json:"refs,omitempty"`
}

// GitBranch represents a branch.
type GitBranch struct {
	Name      string `json:"name"`
	IsCurrent bool   `json:"isCurrent"`
	IsRemote  bool   `json:"isRemote"`
	Upstream  string `json:"upstream,omitempty"`
	Ahead     int    `json:"ahead,omitempty"`
	Behind    int    `json:"behind,omitempty"`
}

// GitRemote represents a remote.
type GitRemote struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Type string `json:"type"` // fetch, push
}

// =============================================================================
// Git Utility Functions
// =============================================================================

// IsGitRepo checks if the current directory is a git repository.
func IsGitRepo(path string) bool {
	gitDir := filepath.Join(path, ".git")
	_, err := os.Stat(gitDir)
	return err == nil
}

// FindGitRoot finds the root of the git repository.
func FindGitRoot(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	// Walk up the directory tree looking for .git
	current := absPath
	for {
		gitDir := filepath.Join(current, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("not a git repository: %s", path)
		}
		current = parent
	}
}

// RunGitCommand runs a git command and returns the output.
func RunGitCommand(args ...string) (string, error) {
	return RunGitCommandWithDir("", args...)
}

// RunGitCommandWithDir runs a git command in a specific directory.
func RunGitCommandWithDir(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %w: %s", strings.Join(args, " "), err, string(output))
	}

	return strings.TrimSpace(string(output)), nil
}

// RunGitCommandWithInput runs a git command with stdin input.
func RunGitCommandWithInput(input string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Stdin = strings.NewReader(input)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %w: %s", strings.Join(args, " "), err, string(output))
	}

	return strings.TrimSpace(string(output)), nil
}

// =============================================================================
// Git Status Functions
// =============================================================================

// GetGitStatus gets the status of a git repository.
func GetGitStatus(path string) (*GitStatus, error) {
	root, err := FindGitRoot(path)
	if err != nil {
		return &GitStatus{IsRepo: false}, nil
	}

	status := &GitStatus{
		IsRepo:          true,
		ModifiedFiles:   []string{},
		StagedFiles:     []string{},
		UntrackedFiles:  []string{},
		ConflictedFiles: []string{},
	}

	// Get branch name
	branch, err := RunGitCommandWithDir(root, "rev-parse", "--abbrev-ref", "HEAD")
	if err == nil {
		status.Branch = branch
	}

	// Get remote
	remote, err := RunGitCommandWithDir(root, "rev-parse", "--abbrev-ref", "@{upstream}")
	if err == nil {
		status.Remote = remote
	}

	// Get ahead/behind counts
	if status.Remote != "" {
		counts, err := RunGitCommandWithDir(root, "rev-list", "--left-right", "--count", status.Branch+"..."+status.Remote)
		if err == nil {
			parts := strings.Split(counts, "\t")
			if len(parts) == 2 {
				fmt.Sscanf(parts[0], "%d", &status.Ahead)
				fmt.Sscanf(parts[1], "%d", &status.Behind)
			}
		}
	}

	// Get status porcelain
	porcelain, err := RunGitCommandWithDir(root, "status", "--porcelain=v1")
	if err == nil {
		scanner := bufio.NewScanner(strings.NewReader(porcelain))
		for scanner.Scan() {
			line := scanner.Text()
			if len(line) < 3 {
				continue
			}

			x := line[0]
			y := line[1]
			file := strings.TrimSpace(line[3:])

			// Check for conflicts
			if x == 'U' || y == 'U' || (x == 'A' && y == 'A') || (x == 'D' && y == 'D') {
				status.ConflictedFiles = append(status.ConflictedFiles, file)
			}

			// Check staged
			if x != ' ' && x != '?' {
				status.StagedFiles = append(status.StagedFiles, file)
			}

			// Check modified
			if y == 'M' {
				status.ModifiedFiles = append(status.ModifiedFiles, file)
			}

			// Check untracked
			if x == '?' && y == '?' {
				status.UntrackedFiles = append(status.UntrackedFiles, file)
			}
		}
	}

	status.HasUncommitted = len(status.StagedFiles) > 0 || len(status.ModifiedFiles) > 0
	status.HasUnstaged = len(status.ModifiedFiles) > 0
	status.HasUntracked = len(status.UntrackedFiles) > 0

	// Get head commit
	head, err := RunGitCommandWithDir(root, "rev-parse", "HEAD")
	if err == nil {
		status.HeadCommit = head
	}

	// Get head message
	message, err := RunGitCommandWithDir(root, "log", "-1", "--format=%s")
	if err == nil {
		status.HeadMessage = message
	}

	return status, nil
}

// =============================================================================
// Git Diff Functions
// =============================================================================

// GetGitDiff gets the diff of the repository.
func GetGitDiff(path string, staged bool) (*GitDiff, error) {
	root, err := FindGitRoot(path)
	if err != nil {
		return nil, err
	}

	args := []string{"diff", "--numstat"}
	if staged {
		args = append(args, "--cached")
	}
	args = append(args, "HEAD")

	output, err := RunGitCommandWithDir(root, args...)
	if err != nil {
		return nil, err
	}

	diff := &GitDiff{
		Files: []GitDiffFile{},
	}

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			continue
		}

		file := GitDiffFile{
			Path: parts[2],
		}

		if parts[0] != "-" {
			fmt.Sscanf(parts[0], "%d", &file.Additions)
		}
		if parts[1] != "-" {
			fmt.Sscanf(parts[1], "%d", &file.Deletions)
		}

		// Determine status
		if file.Additions > 0 && file.Deletions == 0 {
			file.Status = "added"
		} else if file.Additions == 0 && file.Deletions > 0 {
			file.Status = "deleted"
		} else {
			file.Status = "modified"
		}

		diff.Files = append(diff.Files, file)
		diff.TotalAdd += file.Additions
		diff.TotalDelete += file.Deletions
	}

	return diff, nil
}

// GetGitDiffForFile gets the diff for a specific file.
func GetGitDiffForFile(path, filePath string, staged bool) (string, error) {
	root, err := FindGitRoot(path)
	if err != nil {
		return "", err
	}

	args := []string{"diff"}
	if staged {
		args = append(args, "--cached")
	}
	args = append(args, "--", filePath)

	return RunGitCommandWithDir(root, args...)
}

// =============================================================================
// Git Log Functions
// =============================================================================

// GetGitLog gets the git log.
func GetGitLog(path string, count int, since string) ([]GitLogEntry, error) {
	root, err := FindGitRoot(path)
	if err != nil {
		return nil, err
	}

	args := []string{
		"log",
		fmt.Sprintf("-%d", count),
		"--format=%H|%h|%an|%ae|%ad|%s|%P|%D",
		"--date=iso",
	}

	if since != "" {
		args = append(args, "--since="+since)
	}

	output, err := RunGitCommandWithDir(root, args...)
	if err != nil {
		return nil, err
	}

	var entries []GitLogEntry
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "|")
		if len(parts) < 6 {
			continue
		}

		entry := GitLogEntry{
			Hash:      parts[0],
			ShortHash: parts[1],
			Author:    parts[2],
			Email:     parts[3],
			Date:      parts[4],
			Message:   parts[5],
		}

		if len(parts) > 6 && parts[6] != "" {
			entry.Parents = strings.Split(parts[6], " ")
		}
		if len(parts) > 7 && parts[7] != "" {
			entry.Refs = strings.Split(parts[7], ", ")
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// =============================================================================
// Git Branch Functions
// =============================================================================

// GetGitBranches gets all branches.
func GetGitBranches(path string) ([]GitBranch, error) {
	root, err := FindGitRoot(path)
	if err != nil {
		return nil, err
	}

	output, err := RunGitCommandWithDir(root, "branch", "-vv", "--all")
	if err != nil {
		return nil, err
	}

	var branches []GitBranch
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 3 {
			continue
		}

		branch := GitBranch{}

		// Check if current branch
		if line[0] == '*' {
			branch.IsCurrent = true
		}

		// Parse branch name
		line = strings.TrimSpace(line[1:])
		if strings.HasPrefix(line, "remotes/") {
			branch.IsRemote = true
			line = strings.TrimPrefix(line, "remotes/")
		}

		// Extract branch name
		parts := strings.Fields(line)
		if len(parts) > 0 {
			branch.Name = parts[0]
		}

		// Parse upstream
		for i, part := range parts {
			if strings.HasPrefix(part, "[") {
				// Parse upstream and ahead/behind
				upstream := strings.Trim(part, "[]")
				if idx := strings.Index(upstream, ":"); idx != -1 {
					branch.Upstream = upstream[:idx]
					// Parse ahead/behind if present
					stats := upstream[idx+1:]
					if strings.Contains(stats, "ahead") {
						fmt.Sscanf(stats, "ahead %d", &branch.Ahead)
					}
					if strings.Contains(stats, "behind") {
						fmt.Sscanf(stats, "behind %d", &branch.Behind)
					}
				} else {
					branch.Upstream = upstream
				}
				break
			}
			if i == 1 && !strings.HasPrefix(part, "[") {
				break
			}
		}

		branches = append(branches, branch)
	}

	return branches, nil
}

// GetCurrentBranch gets the current branch name.
func GetCurrentBranch(path string) (string, error) {
	root, err := FindGitRoot(path)
	if err != nil {
		return "", err
	}

	return RunGitCommandWithDir(root, "rev-parse", "--abbrev-ref", "HEAD")
}

// =============================================================================
// Git Remote Functions
// =============================================================================

// GetGitRemotes gets all remotes.
func GetGitRemotes(path string) ([]GitRemote, error) {
	root, err := FindGitRoot(path)
	if err != nil {
		return nil, err
	}

	output, err := RunGitCommandWithDir(root, "remote", "-v")
	if err != nil {
		return nil, err
	}

	var remotes []GitRemote
	seen := make(map[string]bool)

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		name := parts[0]
		url := parts[1]
		remoteType := parts[2]
		remoteType = strings.Trim(remoteType, "()")

		// Avoid duplicates (fetch and push are separate lines)
		key := name + ":" + url
		if seen[key] {
			continue
		}
		seen[key] = true

		remotes = append(remotes, GitRemote{
			Name: name,
			URL:  url,
			Type: remoteType,
		})
	}

	return remotes, nil
}

// =============================================================================
// Git Operations
// =============================================================================

// GitStage stages files for commit.
func GitStage(path string, files ...string) error {
	root, err := FindGitRoot(path)
	if err != nil {
		return err
	}

	args := append([]string{"add"}, files...)
	_, err = RunGitCommandWithDir(root, args...)
	return err
}

// GitUnstage unstages files.
func GitUnstage(path string, files ...string) error {
	root, err := FindGitRoot(path)
	if err != nil {
		return err
	}

	args := append([]string{"reset", "HEAD", "--"}, files...)
	_, err = RunGitCommandWithDir(root, args...)
	return err
}

// GitCommit creates a commit.
func GitCommit(path, message string) error {
	root, err := FindGitRoot(path)
	if err != nil {
		return err
	}

	_, err = RunGitCommandWithDir(root, "commit", "-m", message)
	return err
}

// GitCommitWithAuthor creates a commit with a specific author.
func GitCommitWithAuthor(path, message, author, email string) error {
	root, err := FindGitRoot(path)
	if err != nil {
		return err
	}

	authorStr := fmt.Sprintf("%s <%s>", author, email)
	_, err = RunGitCommandWithDir(root, "commit", "-m", message, "--author", authorStr)
	return err
}

// GitPush pushes to a remote.
func GitPush(path, remote, branch string, force bool) error {
	root, err := FindGitRoot(path)
	if err != nil {
		return err
	}

	args := []string{"push"}
	if force {
		args = append(args, "--force-with-lease")
	}
	args = append(args, remote, branch)

	_, err = RunGitCommandWithDir(root, args...)
	return err
}

// GitPull pulls from a remote.
func GitPull(path, remote, branch string) error {
	root, err := FindGitRoot(path)
	if err != nil {
		return err
	}

	_, err = RunGitCommandWithDir(root, "pull", remote, branch)
	return err
}

// GitCheckout checks out a branch.
func GitCheckout(path, branch string, create bool) error {
	root, err := FindGitRoot(path)
	if err != nil {
		return err
	}

	args := []string{"checkout"}
	if create {
		args = append(args, "-b")
	}
	args = append(args, branch)

	_, err = RunGitCommandWithDir(root, args...)
	return err
}

// GitMerge merges a branch.
func GitMerge(path, branch string) error {
	root, err := FindGitRoot(path)
	if err != nil {
		return err
	}

	_, err = RunGitCommandWithDir(root, "merge", branch)
	return err
}

// GitRebase rebases onto a branch.
func GitRebase(path, onto string) error {
	root, err := FindGitRoot(path)
	if err != nil {
		return err
	}

	_, err = RunGitCommandWithDir(root, "rebase", onto)
	return err
}

// GitStash stashes changes.
func GitStash(path, message string) error {
	root, err := FindGitRoot(path)
	if err != nil {
		return err
	}

	args := []string{"stash", "push"}
	if message != "" {
		args = append(args, "-m", message)
	}

	_, err = RunGitCommandWithDir(root, args...)
	return err
}

// GitStashPop pops the stash.
func GitStashPop(path string) error {
	root, err := FindGitRoot(path)
	if err != nil {
		return err
	}

	_, err = RunGitCommandWithDir(root, "stash", "pop")
	return err
}

// =============================================================================
// Git Ignore Functions
// =============================================================================

// IsGitIgnored checks if a path is gitignored.
func IsGitIgnored(path, checkPath string) (bool, error) {
	root, err := FindGitRoot(path)
	if err != nil {
		return false, err
	}

	// Run git check-ignore
	cmd := exec.Command("git", "check-ignore", "-q", checkPath)
	cmd.Dir = root

	err = cmd.Run()
	if err != nil {
		// Exit code 1 means not ignored
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return false, nil
			}
		}
		return false, err
	}

	// Exit code 0 means ignored
	return true, nil
}

// GetGitIgnoredPatterns gets patterns from .gitignore.
func GetGitIgnoredPatterns(path string) ([]string, error) {
	gitignorePath := filepath.Join(path, ".gitignore")
	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		return nil, err
	}

	var patterns []string
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}

	return patterns, nil
}
