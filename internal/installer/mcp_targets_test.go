package installer

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/frkntlr/yap-ai-performance/internal/detector"
)

func TestMCPTargetsAlwaysCreateCore(t *testing.T) {
	home := t.TempDir()
	p := &detector.Platform{OS: "linux", HomeDir: home, LocalBin: filepath.Join(home, ".local", "bin")}
	root := filepath.Join(home, "proj")
	targets := mcpTargets(p, root)

	need := map[string]bool{
		filepath.Join(home, ".cursor", "mcp.json"):                  false,
		filepath.Join(home, ".gemini", "config", "mcp_config.json"): false,
		filepath.Join(home, ".claude.json"):                         false,
		filepath.Join(root, ".cursor", "mcp.json"):                  false,
		filepath.Join(root, ".agents", "mcp_config.json"):           false,
		filepath.Join(root, ".mcp.json"):                            false,
	}
	var claudeMerge bool
	for _, tg := range targets {
		if _, ok := need[tg.Path]; ok {
			need[tg.Path] = true
			if !tg.AlwaysCreate {
				t.Errorf("%s should AlwaysCreate", tg.Path)
			}
		}
		if strings.HasSuffix(tg.Path, ".claude.json") && tg.MergeOnly {
			claudeMerge = true
		}
	}
	for path, found := range need {
		if !found {
			t.Errorf("missing target %s", path)
		}
	}
	if !claudeMerge {
		t.Fatal("Claude Code target must use MergeOnly")
	}
}

func TestMCPTargetsWindowsOptional(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("env APPDATA already set on windows hosts")
	}
	home := t.TempDir()
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	t.Setenv("LOCALAPPDATA", filepath.Join(home, "AppData", "Local"))
	p := &detector.Platform{OS: "windows", HomeDir: home}
	targets := mcpTargets(p, "")
	foundDesktop := false
	for _, tg := range targets {
		if strings.Contains(tg.Path, "claude_desktop_config.json") {
			foundDesktop = true
			if tg.AlwaysCreate {
				t.Error("Claude Desktop should not AlwaysCreate")
			}
		}
	}
	if !foundDesktop {
		t.Fatal("expected Claude Desktop target on windows")
	}
}
