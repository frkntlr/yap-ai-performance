package installer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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
		if err := deployBinary(execPath, yapDestPath, p.OS); err != nil {
			fmt.Printf("Warning: Failed to copy executable to destination: %v. Continuing config updates...\n", err)
			ctx.Logger.Warn("Failed to copy executable to target path", "error", err)
		} else {
			ctx.Logger.Info("Yap binary deployed successfully", "path", yapDestPath)
			if err := detector.EnsureLocalBinOnPATH(p.LocalBin); err != nil {
				fmt.Printf("Warning: Failed to add %s to PATH: %v\n", p.LocalBin, err)
				ctx.Logger.Warn("Failed to ensure LocalBin on PATH", "path", p.LocalBin, "error", err)
			} else if p.OS == "windows" {
				fmt.Printf("✓ Ensured LocalBin on user PATH: %s\n", p.LocalBin)
			}
		}
	}

	// 2. MCP configs (Cursor, Gemini/Antigravity, Claude Code, optional Desktop/Cline, project scopes)
	projectRoot := detectProjectRoot()
	desired := map[string]jsonutil.MCPServer{
		"CodeGraphContext": {Command: yapDestPath, Args: []string{"proxy", "cgc"}},
		"Graphify":         {Command: yapDestPath, Args: []string{"proxy", "graphify"}},
	}

	for _, t := range mcpTargets(p, projectRoot) {
		if t.Path == "" || t.Path == "." {
			continue
		}
		if _, err := os.Stat(t.Path); os.IsNotExist(err) && !t.AlwaysCreate {
			continue
		}

		fmt.Printf("Updating MCP config (%s): %s\n", t.Label, t.Path)
		ctx.Logger.Info("Updating MCP config", "label", t.Label, "path", t.Path)

		if t.MergeOnly {
			existing, err := jsonutil.ReadMCPServersFromFile(t.Path)
			if err != nil {
				fmt.Printf("Warning: failed to read %s: %v\n", t.Path, err)
				ctx.Logger.Warn("Failed to read merge MCP file", "path", t.Path, "error", err)
				continue
			}
			var keysToOverwrite []string
			for name, want := range desired {
				if existingSrv, ok := existing[name]; ok && !mcpServerEqual(existingSrv, want) {
					keysToOverwrite = append(keysToOverwrite, name)
				}
			}
			if len(keysToOverwrite) > 0 && !ctx.DryRun {
				promptMsg := fmt.Sprintf("Mevcut %s ayarları ezilecek (%s). Devam etmek istiyor musunuz?", strings.Join(keysToOverwrite, " ve "), t.Label)
				approved, err := confirm.AskYesNo(promptMsg)
				if err != nil {
					return fmt.Errorf("onay alınırken hata oluştu: %w", err)
				}
				if !approved {
					fmt.Println("İşlem kullanıcı tarafından iptal edildi.")
					continue
				}
			}
			if !ctx.DryRun {
				if _, err := os.Stat(t.Path); err == nil {
					_ = ctx.Backup.Backup(t.Path)
				}
			}
			if err := jsonutil.MergeMCPServers(ctx.DryRun, t.Path, desired); err != nil {
				fmt.Printf("Warning: failed to merge MCP into %s: %v\n", t.Path, err)
				ctx.Logger.Warn("Failed to merge MCP servers", "path", t.Path, "error", err)
				continue
			}
			if ctx.DryRun {
				ctx.Logger.Info("MCP merge simulated", "path", t.Path)
			} else {
				fmt.Printf("✓ Successfully updated: %s\n", t.Path)
			}
			continue
		}

		cfg, err := jsonutil.ReadOrCreate(t.Path)
		if err != nil {
			fmt.Printf("Warning: failed to read config %s: %v\n", t.Path, err)
			ctx.Logger.Warn("Failed to read/create config file", "path", t.Path, "error", err)
			continue
		}

		var keysToOverwrite []string
		if existing, exists := cfg.MCPServers["CodeGraphContext"]; exists && !mcpServerEqual(existing, desired["CodeGraphContext"]) {
			keysToOverwrite = append(keysToOverwrite, "CodeGraphContext")
		}
		if existing, exists := cfg.MCPServers["Graphify"]; exists && !mcpServerEqual(existing, desired["Graphify"]) {
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
				ctx.Logger.Info("Config update cancelled by user", "path", t.Path)
				continue
			}
		}

		cfg.MCPServers["CodeGraphContext"] = desired["CodeGraphContext"]
		cfg.MCPServers["Graphify"] = desired["Graphify"]

		if !ctx.DryRun {
			if _, err := os.Stat(t.Path); err == nil {
				if err := ctx.Backup.Backup(t.Path); err != nil {
					ctx.Logger.Warn("Yedekleme başarısız, devam ediliyor...", "path", t.Path, "error", err)
					fmt.Printf("Warning: yedekleme başarısız (%s): %v\n", t.Path, err)
				}
			}
		}

		if err := jsonutil.Write(ctx.DryRun, t.Path, cfg); err != nil {
			fmt.Printf("Warning: failed to write config %s: %v\n", t.Path, err)
			ctx.Logger.Warn("Failed to write config file", "path", t.Path, "error", err)
		} else if ctx.DryRun {
			ctx.Logger.Info("Config write simulated", "path", t.Path)
		} else {
			fmt.Printf("✓ Successfully updated: %s\n", t.Path)
			ctx.Logger.Info("Config updated successfully", "path", t.Path)
		}
	}

	// 3. Zed context_servers (create global if missing; also update project .zed if present)
	if err := ensureZedConfigs(p, ctx, yapDestPath, projectRoot); err != nil {
		fmt.Printf("Warning: failed to update Zed config: %v\n", err)
		ctx.Logger.Warn("Failed to update Zed config", "error", err)
	}

	// 4. Skills, rules, project scaffold, graphify IDE hooks
	if err := installIDESkillsAndRules(p, ctx, yapDestPath); err != nil {
		fmt.Printf("Warning: failed to install IDE skills/rules: %v\n", err)
		ctx.Logger.Warn("Failed to install IDE skills/rules", "error", err)
	}

	return nil
}

func deployBinary(src, dst, goos string) error {
	// Prefer atomic replace so a running MCP process does not block updates (ETXTBSY).
	tmp := dst + ".new"
	if err := fileutil.CopyFile(src, tmp); err != nil {
		// Fallback to direct copy
		if err2 := fileutil.CopyFile(src, dst); err2 != nil {
			return err2
		}
		if goos != "windows" {
			_ = os.Chmod(dst, 0755)
		}
		return nil
	}
	if goos != "windows" {
		_ = os.Chmod(tmp, 0755)
	}
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func mcpServerEqual(a, b jsonutil.MCPServer) bool {
	if a.Command != b.Command {
		return false
	}
	if len(a.Args) != len(b.Args) {
		return false
	}
	for i := range a.Args {
		if a.Args[i] != b.Args[i] {
			return false
		}
	}
	return true
}

func ensureZedConfigs(p *detector.Platform, ctx *context.RunContext, yapDestPath, projectRoot string) error {
	var globalPath string
	if p.OS == "windows" {
		globalPath = filepath.Join(os.Getenv("APPDATA"), "Zed", "settings.json")
	} else {
		globalPath = filepath.Join(p.HomeDir, ".config", "zed", "settings.json")
	}
	if err := updateZedConfigAt(ctx, globalPath, yapDestPath, true); err != nil {
		return err
	}
	if projectRoot != "" {
		projPath := filepath.Join(projectRoot, ".zed", "settings.json")
		// Update project Zed settings if the file already exists (or create for parity).
		if err := updateZedConfigAt(ctx, projPath, yapDestPath, true); err != nil {
			return err
		}
	}
	return nil
}

func updateZedConfigAt(ctx *context.RunContext, zedConfigPath, yapDestPath string, createIfMissing bool) error {
	_, err := os.Stat(zedConfigPath)
	missing := os.IsNotExist(err)
	if missing && !createIfMissing {
		return nil
	}
	if err != nil && !missing {
		return err
	}

	fmt.Printf("Updating Zed config: %s\n", zedConfigPath)
	ctx.Logger.Info("Updating Zed config", "path", zedConfigPath)

	settings := make(map[string]interface{})
	if !missing {
		data, err := ioutil.ReadFile(zedConfigPath)
		if err != nil {
			return fmt.Errorf("failed to read Zed config: %w", err)
		}
		if len(data) > 0 {
			cleanData := stripTrailingCommasAndComments(data)
			if err := json.Unmarshal(cleanData, &settings); err != nil {
				return fmt.Errorf("failed to parse Zed config: %w", err)
			}
		}
	}

	contextServersVal, exists := settings["context_servers"]
	var contextServers map[string]interface{}
	if exists {
		var ok bool
		contextServers, ok = contextServersVal.(map[string]interface{})
		if !ok {
			contextServers = make(map[string]interface{})
		}
	} else {
		contextServers = make(map[string]interface{})
	}

	var keysToOverwrite []string
	if existing, exists := contextServers["yap-cgc"]; exists {
		if m, ok := existing.(map[string]interface{}); !ok || !zedServerEqual(m, yapDestPath, "cgc") {
			keysToOverwrite = append(keysToOverwrite, "yap-cgc")
		}
	}
	if existing, exists := contextServers["yap-graphify"]; exists {
		if m, ok := existing.(map[string]interface{}); !ok || !zedServerEqual(m, yapDestPath, "graphify") {
			keysToOverwrite = append(keysToOverwrite, "yap-graphify")
		}
	}

	if len(keysToOverwrite) > 0 && !ctx.DryRun {
		promptMsg := fmt.Sprintf("Mevcut Zed %s ayarları ezilecek. Devam etmek istiyor musunuz?", strings.Join(keysToOverwrite, " ve "))
		approved, err := confirm.AskYesNo(promptMsg)
		if err != nil {
			return fmt.Errorf("onay alınırken hata oluştu: %w", err)
		}
		if !approved {
			fmt.Println("Zed ayarları güncellemesi kullanıcı tarafından iptal edildi.")
			ctx.Logger.Info("Zed config update cancelled by user")
			return nil
		}
	}

	contextServers["yap-cgc"] = map[string]interface{}{
		"command": yapDestPath,
		"args":    []string{"proxy", "cgc"},
	}
	contextServers["yap-graphify"] = map[string]interface{}{
		"command": yapDestPath,
		"args":    []string{"proxy", "graphify"},
	}
	settings["context_servers"] = contextServers

	if !ctx.DryRun && !missing {
		if err := ctx.Backup.Backup(zedConfigPath); err != nil {
			ctx.Logger.Warn("Zed yedekleme başarısız, devam ediliyor...", "path", zedConfigPath, "error", err)
		}
	}

	if ctx.DryRun {
		dryrun.PrintSimulation(fmt.Sprintf("%s güncellenecek (Zed)", zedConfigPath))
		return nil
	}

	newData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal Zed config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(zedConfigPath), 0755); err != nil {
		return fmt.Errorf("failed to create Zed config directory: %w", err)
	}
	if err := ioutil.WriteFile(zedConfigPath, newData, 0644); err != nil {
		return fmt.Errorf("failed to write Zed config: %w", err)
	}
	fmt.Printf("✓ Successfully updated Zed config: %s\n", zedConfigPath)
	ctx.Logger.Info("Zed config updated successfully", "path", zedConfigPath)
	return nil
}

func zedServerEqual(m map[string]interface{}, yapPath, proxy string) bool {
	cmd, _ := m["command"].(string)
	if cmd != yapPath {
		return false
	}
	args, ok := m["args"].([]interface{})
	if !ok || len(args) != 2 {
		return false
	}
	a0, _ := args[0].(string)
	a1, _ := args[1].(string)
	return a0 == "proxy" && a1 == proxy
}

func stripTrailingCommasAndComments(data []byte) []byte {
	var result []byte
	inString := false
	escaped := false
	inLineComment := false
	inBlockComment := false

	for i := 0; i < len(data); i++ {
		c := data[i]

		if inLineComment {
			if c == '\n' {
				inLineComment = false
				result = append(result, c)
			}
			continue
		}

		if inBlockComment {
			if c == '*' && i+1 < len(data) && data[i+1] == '/' {
				inBlockComment = false
				i++
			}
			continue
		}

		if inString {
			if escaped {
				escaped = false
			} else if c == '\\' {
				escaped = true
			} else if c == '"' {
				inString = false
			}
			result = append(result, c)
			continue
		}

		// Handle comments
		if c == '/' && i+1 < len(data) {
			if data[i+1] == '/' {
				inLineComment = true
				i++
				continue
			} else if data[i+1] == '*' {
				inBlockComment = true
				i++
				continue
			}
		}

		if c == '"' {
			inString = true
		}

		result = append(result, c)
	}

	var clean []byte
	inString = false
	escaped = false

	for i := 0; i < len(result); i++ {
		c := result[i]

		if inString {
			if escaped {
				escaped = false
			} else if c == '\\' {
				escaped = true
			} else if c == '"' {
				inString = false
			}
			clean = append(clean, c)
			continue
		}

		if c == '"' {
			inString = true
			clean = append(clean, c)
			continue
		}

		if c == ',' {
			isTrailing := false
			for j := i + 1; j < len(result); j++ {
				nc := result[j]
				if nc == ' ' || nc == '\t' || nc == '\n' || nc == '\r' {
					continue
				}
				if nc == '}' || nc == ']' {
					isTrailing = true
				}
				break
			}
			if isTrailing {
				continue
			}
		}

		clean = append(clean, c)
	}

	return clean
}
