//go:build !windows

package detector

import (
	"os"
	"path/filepath"
	"strings"
)

// EnsureLocalBinOnPATH is a no-op on Unix beyond ensuring the directory exists
// and prepending to the current process PATH when missing (shell profiles
// normally already include ~/.local/bin).
func EnsureLocalBinOnPATH(localBin string) error {
	if localBin == "" {
		return nil
	}
	clean := filepath.Clean(localBin)
	if err := os.MkdirAll(clean, 0755); err != nil {
		return err
	}
	cur := os.Getenv("PATH")
	if !unixPathListContains(cur, clean) {
		os.Setenv("PATH", clean+string(os.PathListSeparator)+cur)
	}
	return nil
}

func unixPathListContains(pathList, dir string) bool {
	want := filepath.Clean(dir)
	for _, part := range strings.Split(pathList, string(os.PathListSeparator)) {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if filepath.Clean(part) == want {
			return true
		}
	}
	return false
}
