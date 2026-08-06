package installer

import (
	"os"
	"path/filepath"

	"github.com/frkntlr/yap-ai-performance/internal/detector"
)

// mcpTarget describes an MCP config file to update during Step5.
type mcpTarget struct {
	Path         string
	Label        string
	AlwaysCreate bool
	// MergeOnly updates mcpServers without rewriting the whole JSON document
	// (required for Claude Code ~/.claude.json).
	MergeOnly bool
}

// mcpTargets returns global and optional project MCP config targets.
func mcpTargets(p *detector.Platform, projectRoot string) []mcpTarget {
	var targets []mcpTarget

	targets = append(targets,
		mcpTarget{
			Path:         filepath.Join(p.HomeDir, ".cursor", "mcp.json"),
			Label:        "Cursor MCP",
			AlwaysCreate: true,
		},
		mcpTarget{
			Path:         filepath.Join(p.HomeDir, ".gemini", "config", "mcp_config.json"),
			Label:        "Gemini/Antigravity MCP",
			AlwaysCreate: true,
		},
		mcpTarget{
			Path:         filepath.Join(p.HomeDir, ".claude.json"),
			Label:        "Claude Code user MCP",
			AlwaysCreate: true,
			MergeOnly:    true,
		},
	)

	// Optional: Claude Desktop + Cline (update only if present)
	if p.OS == "windows" {
		appdata := envOrEmpty("APPDATA")
		localappdata := envOrEmpty("LOCALAPPDATA")
		targets = append(targets,
			mcpTarget{Path: filepath.Join(appdata, "Claude", "claude_desktop_config.json"), Label: "Claude Desktop"},
			mcpTarget{Path: filepath.Join(appdata, "Cursor", "User", "globalStorage", "saoudrizwan.claude-dev", "settings", "cline_mcp_settings.json"), Label: "Cline (Cursor)"},
			mcpTarget{Path: filepath.Join(localappdata, "Programs", "cursor", "resources", "app", "extensions", "saoudrizwan.claude-dev", "settings", "cline_mcp_settings.json"), Label: "Cline (Cursor app)"},
		)
	} else {
		targets = append(targets,
			mcpTarget{Path: filepath.Join(p.HomeDir, ".config", "Claude", "claude_desktop_config.json"), Label: "Claude Desktop"},
			mcpTarget{Path: filepath.Join(p.HomeDir, ".config", "Cursor", "User", "globalStorage", "saoudrizwan.claude-dev", "settings", "cline_mcp_settings.json"), Label: "Cline (Cursor)"},
			mcpTarget{Path: filepath.Join(p.HomeDir, ".config", "Code", "User", "globalStorage", "saoudrizwan.claude-dev", "settings", "cline_mcp_settings.json"), Label: "Cline (VS Code)"},
		)
		if p.OS == "darwin" {
			targets = append(targets, mcpTarget{
				Path:  filepath.Join(p.HomeDir, "Library", "Application Support", "Claude", "claude_desktop_config.json"),
				Label: "Claude Desktop (macOS)",
			})
		}
	}

	if projectRoot != "" {
		targets = append(targets,
			mcpTarget{
				Path:         filepath.Join(projectRoot, ".cursor", "mcp.json"),
				Label:        "project Cursor MCP",
				AlwaysCreate: true,
			},
			mcpTarget{
				Path:         filepath.Join(projectRoot, ".agents", "mcp_config.json"),
				Label:        "project Antigravity/Agents MCP",
				AlwaysCreate: true,
			},
			mcpTarget{
				Path:         filepath.Join(projectRoot, ".mcp.json"),
				Label:        "project Claude Code MCP",
				AlwaysCreate: true,
			},
		)
	}

	return targets
}

func envOrEmpty(key string) string {
	return os.Getenv(key)
}
