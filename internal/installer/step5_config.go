package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/frkntlr/yap-ai-performance/internal/confirm"
	"github.com/frkntlr/yap-ai-performance/internal/context"
	"github.com/frkntlr/yap-ai-performance/internal/detector"
	"github.com/frkntlr/yap-ai-performance/internal/dryrun"
	"github.com/frkntlr/yap-ai-performance/pkg/fileutil"
	"github.com/frkntlr/yap-ai-performance/pkg/jsonutil"
)

// Step5Config copies the running binary to the local bin directory and updates MCP configuration files.
func Step5Config(p *detector.Platform, ctx *context.RunContext) error {
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
	ctx.Logger.Info("Deploying yap binary", "src", execPath, "dst", yapDestPath)

	if ctx.DryRun {
		dryrun.PrintSimulation(fmt.Sprintf("%s dosyasına binary kopyalanacak", yapDestPath))
	} else {
		if err := fileutil.CopyFile(execPath, yapDestPath); err != nil {
			fmt.Printf("Warning: Failed to copy executable to destination: %v. Continuing config updates...\n", err)
			ctx.Logger.Warn("Failed to copy executable to target path", "error", err)
		} else {
			if p.OS != "windows" {
				_ = os.Chmod(yapDestPath, 0755)
			}
			ctx.Logger.Info("Yap binary deployed successfully", "path", yapDestPath)
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
		ctx.Logger.Info("Updating MCP config", "path", cfgPath)

		cfg, err := jsonutil.ReadOrCreate(cfgPath)
		if err != nil {
			fmt.Printf("Warning: failed to read config %s: %v\n", cfgPath, err)
			ctx.Logger.Warn("Failed to read/create config file", "path", cfgPath, "error", err)
			continue
		}

		// Check if we are overwriting existing keys
		var keysToOverwrite []string
		if _, exists := cfg.MCPServers["CodeGraphContext"]; exists {
			keysToOverwrite = append(keysToOverwrite, "CodeGraphContext")
		}
		if _, exists := cfg.MCPServers["Graphify"]; exists {
			keysToOverwrite = append(keysToOverwrite, "Graphify")
		}

		if len(keysToOverwrite) > 0 && !ctx.DryRun {
			promptMsg := fmt.Sprintf("Mevcut %s ayarları ezilecek. Devam etmek istiyor musunuz?", strings.Join(keysToOverwrite, " ve "))
			approved, err := confirm.AskYesNo(promptMsg)
			if err != nil {
				ctx.Logger.Error("Confirmation prompt error", "error", err)
				return fmt.Errorf("onay alınırken hata oluştu: %w", err)
			}
			if !approved {
				fmt.Println("İşlem kullanıcı tarafından iptal edildi.")
				ctx.Logger.Info("Config update cancelled by user", "path", cfgPath)
				continue
			}
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

		// Backup before writing (only if not dry-running and file exists)
		if !ctx.DryRun {
			if _, err := os.Stat(cfgPath); err == nil {
				if err := ctx.Backup.Backup(cfgPath); err != nil {
					ctx.Logger.Warn("Yedekleme başarısız, devam ediliyor...", "path", cfgPath, "error", err)
					fmt.Printf("Warning: yedekleme başarısız (%s): %v\n", cfgPath, err)
				} else {
					ctx.Logger.Info("Backup created successfully", "path", cfgPath)
				}
			}
		}

		if err := jsonutil.Write(ctx.DryRun, cfgPath, cfg); err != nil {
			fmt.Printf("Warning: failed to write config %s: %v\n", cfgPath, err)
			ctx.Logger.Warn("Failed to write config file", "path", cfgPath, "error", err)
		} else {
			if ctx.DryRun {
				ctx.Logger.Info("Config write simulated", "path", cfgPath)
			} else {
				fmt.Printf("✓ Successfully updated: %s\n", cfgPath)
				ctx.Logger.Info("Config updated successfully", "path", cfgPath)
			}
		}
	}

	return nil
}

