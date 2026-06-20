package installer

import (
	"fmt"
	"strings"

	"github.com/frkntlr/yap-ai-performance/internal/detector"
	"github.com/frkntlr/yap-ai-performance/pkg/runner"
)

// Step3Tools installs CodeGraphContext and Graphifyy using pipx and uv.
func Step3Tools(p *detector.Platform) error {
	fmt.Println("Installing/Updating CodeGraphContext via pipx...")

	// Check if already installed in pipx
	pipxList, err := runner.RunAndCapture("pipx", "list")
	if err == nil && strings.Contains(pipxList, "codegraphcontext") {
		fmt.Println("codegraphcontext already installed. Upgrading...")
		// Try to upgrade, ignore error if already at latest
		_ = runner.Run("pipx", "upgrade", "codegraphcontext")
	} else {
		if err := runner.Run("pipx", "install", "codegraphcontext"); err != nil {
			return fmt.Errorf("failed to install codegraphcontext: %v", err)
		}
	}

	fmt.Println("Installing/Updating Graphifyy via uv...")
	// uv tool install --force will install or upgrade
	if err := runner.Run("uv", "tool", "install", "--force", "graphifyy[mcp]"); err != nil {
		// Fallback if uv tool install fails
		fmt.Println("uv tool install failed. Trying alternative install using pip...")
		pythonCmd := "python3"
		if p.OS == "windows" {
			pythonCmd = "python"
		}
		if err := runner.Run(pythonCmd, "-m", "pip", "install", "--user", "graphifyy[mcp]"); err != nil {
			return fmt.Errorf("failed to install graphifyy via fallback: %v", err)
		}
	}

	return nil
}
