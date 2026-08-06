//go:build windows

package detector

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// EnsureLocalBinOnPATH adds localBin to the persistent User PATH if missing.
// Also prepends it to the current process PATH for the remainder of this run.
func EnsureLocalBinOnPATH(localBin string) error {
	if localBin == "" {
		return nil
	}
	clean := filepath.Clean(localBin)
	if err := os.MkdirAll(clean, 0755); err != nil {
		return fmt.Errorf("create LocalBin: %w", err)
	}

	// Update current process PATH immediately.
	cur := os.Getenv("PATH")
	if !pathListContains(cur, clean) {
		os.Setenv("PATH", clean+string(os.PathListSeparator)+cur)
	}

	userPath, err := readUserPath()
	if err != nil {
		return err
	}
	if pathListContains(userPath, clean) {
		return nil
	}
	newPath := clean
	if userPath != "" {
		newPath = clean + string(os.PathListSeparator) + userPath
	}
	return writeUserPath(newPath)
}

func readUserPath() (string, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
		"[Environment]::GetEnvironmentVariable('Path','User')")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("read user PATH: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func writeUserPath(path string) error {
	// Use PowerShell SetEnvironmentVariable for a proper User-scoped update
	// (avoids setx's 1024-char truncation).
	ps := fmt.Sprintf(
		"[Environment]::SetEnvironmentVariable('Path', %s, 'User')",
		psSingleQuoted(path),
	)
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", ps)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("write user PATH: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func psSingleQuoted(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func pathListContains(pathList, dir string) bool {
	want := strings.ToLower(filepath.Clean(dir))
	for _, part := range strings.Split(pathList, string(os.PathListSeparator)) {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.ToLower(filepath.Clean(part)) == want {
			return true
		}
	}
	return false
}
