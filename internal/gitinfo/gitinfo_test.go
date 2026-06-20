package gitinfo

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestReadGitInfo(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git: %v", err)
	}

	// Configure local user info for dummy commits
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	_ = cmd.Run()

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	_ = cmd.Run()

	// Check on empty repo
	info, err := Read(tmpDir, false)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if !info.IsRepo {
		t.Errorf("expected IsRepo to be true")
	}

	// Create a file and commit it
	filePath := filepath.Join(tmpDir, "file.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	cmd = exec.Command("git", "add", "file.txt")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to git add: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "First commit")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to git commit: %v", err)
	}

	// Verify info after commit
	info, err = Read(tmpDir, false)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if info.LastCommit != "First commit" {
		t.Errorf("expected LastCommit 'First commit', got %q", info.LastCommit)
	}

	// Make a modification
	if err := os.WriteFile(filePath, []byte("hello world"), 0644); err != nil {
		t.Fatalf("failed to write file modification: %v", err)
	}

	// Check status
	info, err = Read(tmpDir, false)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if len(info.ModifiedFiles) == 0 || info.ModifiedFiles[0] != "file.txt" {
		t.Errorf("expected file.txt to be modified, got: %v", info.ModifiedFiles)
	}
}
