package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/frkntlr/yap-ai-performance/internal/context"
	"github.com/frkntlr/yap-ai-performance/internal/detector"
	"github.com/frkntlr/yap-ai-performance/internal/dryrun"
	"github.com/frkntlr/yap-ai-performance/pkg/jsonutil"
)

// installIDESkillsAndRules deploys /yap skills, always-apply rules, project MCP,
// and graphify platform hooks so IDEs pick up the stack automatically.
func installIDESkillsAndRules(p *detector.Platform, ctx *context.RunContext, yapDestPath string) error {
	fmt.Println("Installing IDE skills & rules (/yap)...")

	cursorSkill, err := assetBytes("cursor_yap_skill.md")
	if err != nil {
		return fmt.Errorf("embed cursor skill: %w", err)
	}
	cursorRule, err := assetBytes("cursor_yap_rule.mdc")
	if err != nil {
		return fmt.Errorf("embed cursor rule: %w", err)
	}
	geminiSkill, err := assetBytes("gemini_yap_skill.md")
	if err != nil {
		return fmt.Errorf("embed gemini skill: %w", err)
	}

	targets := []struct {
		path string
		data []byte
		desc string
	}{
		// Global Cursor skill → /yap everywhere
		{filepath.Join(p.HomeDir, ".cursor", "skills", "yap", "SKILL.md"), cursorSkill, "Cursor global /yap skill"},
		// Global Cursor rule (best-effort; project rule is authoritative)
		{filepath.Join(p.HomeDir, ".cursor", "rules", "yap-context.mdc"), cursorRule, "Cursor global yap rule"},
		// Gemini / Antigravity-style skill
		{filepath.Join(p.HomeDir, ".gemini", "config", "skills", "yap", "SKILL.md"), geminiSkill, "Gemini /yap skill"},
		// Agents-compatible skill path (Codex/OpenCode ecosystem)
		{filepath.Join(p.HomeDir, ".agents", "skills", "yap", "SKILL.md"), cursorSkill, "Agents /yap skill"},
	}

	for _, t := range targets {
		if err := writeManagedFile(ctx, t.path, t.data, t.desc); err != nil {
			fmt.Printf("Warning: %s: %v\n", t.desc, err)
			ctx.Logger.Warn("Failed to write IDE asset", "desc", t.desc, "path", t.path, "error", err)
		}
	}

	// Project-local Cursor scaffold when cwd looks like a repo/workspace
	if root := detectProjectRoot(); root != "" {
		projTargets := []struct {
			path string
			data []byte
			desc string
		}{
			{filepath.Join(root, ".cursor", "skills", "yap", "SKILL.md"), cursorSkill, "project Cursor /yap skill"},
			{filepath.Join(root, ".cursor", "rules", "yap-context.mdc"), cursorRule, "project Cursor yap rule"},
		}
		for _, t := range projTargets {
			if err := writeManagedFile(ctx, t.path, t.data, t.desc); err != nil {
				fmt.Printf("Warning: %s: %v\n", t.desc, err)
				ctx.Logger.Warn("Failed to write project IDE asset", "desc", t.desc, "path", t.path, "error", err)
			}
		}
		if err := writeProjectCursorMCP(ctx, root, yapDestPath); err != nil {
			fmt.Printf("Warning: project Cursor MCP: %v\n", err)
			ctx.Logger.Warn("Failed to write project Cursor MCP", "error", err)
		}
	} else {
		ctx.Logger.Info("No project root detected; skipped project .cursor scaffold")
	}

	runGraphifyPlatformInstalls(ctx)
	return nil
}

func writeManagedFile(ctx *context.RunContext, path string, data []byte, desc string) error {
	if ctx.DryRun {
		dryrun.PrintSimulation(fmt.Sprintf("%s yazılacak (%s)", path, desc))
		return nil
	}

	if _, err := os.Stat(path); err == nil {
		if err := ctx.Backup.Backup(path); err != nil {
			ctx.Logger.Warn("Backup failed before overwrite", "path", path, "error", err)
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}
	fmt.Printf("✓ %s → %s\n", desc, path)
	ctx.Logger.Info("IDE asset written", "desc", desc, "path", path)
	return nil
}

func writeProjectCursorMCP(ctx *context.RunContext, root, yapDestPath string) error {
	path := filepath.Join(root, ".cursor", "mcp.json")
	cfg, err := jsonutil.ReadOrCreate(path)
	if err != nil {
		return err
	}
	cfg.MCPServers["CodeGraphContext"] = jsonutil.MCPServer{
		Command: yapDestPath,
		Args:    []string{"proxy", "cgc"},
	}
	cfg.MCPServers["Graphify"] = jsonutil.MCPServer{
		Command: yapDestPath,
		Args:    []string{"proxy", "graphify"},
	}
	if err := jsonutil.Write(ctx.DryRun, path, cfg); err != nil {
		return err
	}
	if !ctx.DryRun {
		fmt.Printf("✓ project Cursor MCP → %s\n", path)
	}
	return nil
}

func detectProjectRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	dir := cwd
	for {
		for _, marker := range []string{".git", "go.mod", "package.json", "Cargo.toml", "pyproject.toml", ".codegraphcontext_db"} {
			if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func runGraphifyPlatformInstalls(ctx *context.RunContext) {
	plat, _ := detector.Detect()
	graphifyPath, err := exec.LookPath("graphify")
	if err != nil {
		if plat != nil {
			if found := detector.FindGraphifyBin(plat); found != "" {
				graphifyPath = found
			}
		}
		if graphifyPath == "" {
			home, _ := os.UserHomeDir()
			candidate := filepath.Join(home, ".local", "bin", "graphify")
			if _, statErr := os.Stat(candidate); statErr == nil {
				graphifyPath = candidate
			} else {
				ctx.Logger.Info("graphify not on PATH; skipping platform skill install")
				return
			}
		}
	}

	// Install graphify's own IDE hooks where the binary supports them.
	platforms := []string{"cursor", "gemini", "agents", "antigravity"}
	for _, plat := range platforms {
		if ctx.DryRun {
			dryrun.PrintSimulation(fmt.Sprintf("graphify install --platform %s", plat))
			continue
		}
		cmd := exec.Command(graphifyPath, "install", "--platform", plat)
		out, err := cmd.CombinedOutput()
		msg := strings.TrimSpace(string(out))
		if err != nil {
			ctx.Logger.Warn("graphify platform install failed", "platform", plat, "error", err, "out", msg)
			fmt.Printf("Warning: graphify --platform %s: %v\n", plat, err)
			continue
		}
		fmt.Printf("✓ graphify platform: %s\n", plat)
		if msg != "" {
			ctx.Logger.Info("graphify install", "platform", plat, "out", msg)
		}
	}
}
