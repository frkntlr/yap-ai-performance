package detector

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/frkntlr/yap-ai-performance/pkg/runner"
)

type Platform struct {
	OS         string // "linux", "darwin", "windows"
	PackageMgr string // "apt", "pacman", "brew", "winget", "unknown"
	HomeDir    string
	LocalBin   string // Path to install CLI wrappers / binaries locally
}

// Detect returns the current system platform details.
func Detect() (*Platform, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to detect user home directory: %v", err)
	}

	p := &Platform{
		OS:      runtime.GOOS,
		HomeDir: home,
	}

	switch p.OS {
	case "windows":
		p.PackageMgr = "winget"
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			localAppData = filepath.Join(home, "AppData", "Local")
		}
		p.LocalBin = filepath.Join(localAppData, "Programs", "yap")

	case "darwin":
		p.PackageMgr = "brew"
		p.LocalBin = filepath.Join(home, ".local", "bin")

	case "linux":
		p.LocalBin = filepath.Join(home, ".local", "bin")
		p.PackageMgr = detectLinuxPackageManager()

	default:
		p.PackageMgr = "unknown"
		p.LocalBin = filepath.Join(home, ".local", "bin")
	}

	return p, nil
}

func detectLinuxPackageManager() string {
	// First check /etc/os-release
	if data, err := ioutil.ReadFile("/etc/os-release"); err == nil {
		content := string(data)
		if strings.Contains(content, "ID=arch") || strings.Contains(content, "ID_LIKE=arch") || strings.Contains(content, "ID=cachyos") {
			return "pacman"
		}
		if strings.Contains(content, "ID=ubuntu") || strings.Contains(content, "ID_LIKE=debian") || strings.Contains(content, "ID=debian") {
			return "apt"
		}
	}

	// Fallback to searching paths
	if runner.Exists("pacman") {
		return "pacman"
	}
	if runner.Exists("apt-get") {
		return "apt"
	}

	return "unknown"
}
