package gitinfo

import (
	"strings"

	"github.com/frkntlr/yap-ai-performance/pkg/runner"
)

// GitInfo contains git repository status and difference data.
type GitInfo struct {
	IsRepo         bool     `json:"is_repo"`
	Branch         string   `json:"branch"`
	ModifiedFiles  []string `json:"modified_files"`
	UntrackedFiles []string `json:"untracked_files"`
	StagedFiles    []string `json:"staged_files"`
	DiffStat       string   `json:"diff_stat"`
	FullDiff       string   `json:"full_diff,omitempty"`
	LastCommit     string   `json:"last_commit"`
}

// Read inspects a git repository at the specified directory and collects metadata.
func Read(dir string, fetchFullDiff bool) (*GitInfo, error) {
	info := &GitInfo{
		IsRepo:         false,
		ModifiedFiles:  []string{},
		UntrackedFiles: []string{},
		StagedFiles:    []string{},
	}

	// 1. Check if it's a git repo
	out, err := runner.RunInDirAndCapture(dir, "git", "rev-parse", "--is-inside-work-tree")
	if err != nil || strings.TrimSpace(out) != "true" {
		return info, nil
	}
	info.IsRepo = true

	// 2. Fetch current branch name
	branchOut, err := runner.RunInDirAndCapture(dir, "git", "branch", "--show-current")
	if err == nil {
		info.Branch = strings.TrimSpace(branchOut)
	}
	if info.Branch == "" {
		branchOut, err = runner.RunInDirAndCapture(dir, "git", "rev-parse", "--abbrev-ref", "HEAD")
		if err == nil {
			info.Branch = strings.TrimSpace(branchOut)
		}
	}

	// 3. Fetch changed files
	statusOut, err := runner.RunInDirAndCapture(dir, "git", "status", "--porcelain")
	if err == nil {
		lines := strings.Split(statusOut, "\n")
		for _, line := range lines {
			if len(line) < 4 {
				continue
			}
			statusType := line[:2]
			filePath := strings.TrimSpace(line[3:])

			if strings.Contains(statusType, "M") {
				info.ModifiedFiles = append(info.ModifiedFiles, filePath)
			} else if strings.Contains(statusType, "?") {
				info.UntrackedFiles = append(info.UntrackedFiles, filePath)
			} else if strings.Contains(statusType, "A") {
				info.StagedFiles = append(info.StagedFiles, filePath)
			}
		}
	}

	// 4. Fetch diff stat
	diffStatOut, err := runner.RunInDirAndCapture(dir, "git", "diff", "--stat", "HEAD")
	if err == nil {
		info.DiffStat = strings.TrimSpace(diffStatOut)
	}

	// 5. Fetch full diff if explicitly requested
	if fetchFullDiff {
		fullDiffOut, err := runner.RunInDirAndCapture(dir, "git", "diff", "HEAD")
		if err == nil {
			info.FullDiff = strings.TrimSpace(fullDiffOut)
		}
	}

	// 6. Fetch last commit subject line
	lastCommitOut, err := runner.RunInDirAndCapture(dir, "git", "log", "-1", "--pretty=format:%s")
	if err == nil {
		info.LastCommit = strings.TrimSpace(lastCommitOut)
	}

	return info, nil
}
