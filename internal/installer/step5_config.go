package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/frkntlr/yap-ai-performance/internal/detector"
	"github.com/frkntlr/yap-ai-performance/pkg/fileutil"
	"github.com/frkntlr/yap-ai-performance/pkg/jsonutil"
)

// Step5Config copies the running binary to the local bin directory and updates MCP configuration files.
func Step5Config(p *detector.Platform) error {
	// 1. Copy current executable to local bin path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %v", err)
	}

	targetName := "yap"
	if p.OS == "windows" {
		targetName = "yap.exe"
	}
	yapDestPath := filepath.Join(p.LocalBin, targetName)

	fmt.Printf("Deploying 'yap' binary to: %s\n", yapDestPath)
	if err := fileutil.CopyFile(execPath, yapDestPath); err != nil {
		// If running under development (e.g. go run), copy might fail or be unexpected,
		// but we still try or warn.
		fmt.Printf("Warning: Failed to copy executable to destination: %v. Continuing config updates...\n", err)
		// We fallback to using the detected/expected path anyway.
	} else {
		if p.OS != "windows" {
			_ = os.Chmod(yapDestPath, 0755)
		}
	}

	// 2. Determine configuration paths to update
	var configs []string
	if p.OS == "windows" {
		appdata := os.Getenv("APPDATA")
		localappdata := os.Getenv("LOCALAPPDATA")
		configs = []string{
			filepath.Join(p.HomeDir, ".gemini", "config", "mcp_config.json"),
			filepath.Join(appdata, "Claude", "claude_desktop_config.json"),
			filepath.Join(appdata, "Cursor", "User", "globalStorage", "saoudrizwan.claude-dev", "settings", "cline_mcp_settings.json"),
			filepath.Join(localappdata, "Programs", "cursor", "resources", "app", "extensions", "saoudrizwan.claude-dev", "settings", "cline_mcp_settings.json"),
		}
	} else {
		configs = []string{
			filepath.Join(p.HomeDir, ".gemini", "config", "mcp_config.json"),
			filepath.Join(p.HomeDir, ".config", "Cursor", "User", "globalStorage", "saoudrizwan.claude-dev", "settings", "cline_mcp_settings.json"),
			filepath.Join(p.HomeDir, ".config", "Code", "User", "globalStorage", "saoudrizwan.claude-dev", "settings", "cline_mcp_settings.json"),
			filepath.Join(p.HomeDir, ".config", "Claude", "claude_desktop_config.json"),
		}
	}

	for _, cfgPath := range configs {
		isGemini := strings.Contains(cfgPath, "gemini")

		// If config doesn't exist and it's not Gemini, skip to avoid creating unnecessary configs
		if _, err := os.Stat(cfgPath); os.IsNotExist(err) && !isGemini {
			continue
		}

		fmt.Printf("Updating MCP config: %s\n", cfgPath)
		cfg, err := jsonutil.ReadOrCreate(cfgPath)
		if err != nil {
			fmt.Printf("Warning: failed to read config %s: %v\n", cfgPath, err)
			continue
		}

		// Update CodeGraphContext configuration
		cfg.MCPServers["CodeGraphContext"] = jsonutil.MCPServer{
			Command: yapDestPath,
			Args:    []string{"proxy", "cgc"},
		}

		// Update Graphify configuration
		cfg.MCPServers["Graphify"] = jsonutil.MCPServer{
			Command: yapDestPath,
			Args:    []string{"proxy", "graphify"},
		}

		if err := jsonutil.Write(cfgPath, cfg); err != nil {
			fmt.Printf("Warning: failed to write config %s: %v\n", cfgPath, err)
		} else {
			fmt.Printf("✓ Successfully updated: %s\n", cfgPath)
		}
	}

	return nil
}
