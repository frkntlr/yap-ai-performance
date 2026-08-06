package status

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/frkntlr/yap-ai-performance/internal/detector"
	"github.com/frkntlr/yap-ai-performance/pkg/jsonutil"
	"github.com/frkntlr/yap-ai-performance/pkg/runner"
)

type CheckResult struct {
	Name   string
	OK     bool
	Detail string
	Fix    string
}

// RunStatus performs active diagnostic checks and returns the results.
func RunStatus(p *detector.Platform, logger *slog.Logger) []CheckResult {
	var results []CheckResult

	if logger != nil {
		logger.Debug("Starting active diagnostics suite")
	}

	// 1. Tool Checks
	results = append(results, checkTools(p, logger)...)

	// 2. Config Checks
	results = append(results, checkConfigs(p, logger)...)

	// 2b. IDE skills / rules
	results = append(results, checkIDESkills(p, logger)...)

	// 3. Patch Checks
	results = append(results, checkPatches(p, logger)...)

	// 4. Liveness Checks (only if tools are present)
	results = append(results, checkLiveness(p, logger)...)

	if logger != nil {
		logger.Debug("Completed diagnostics suite", "totalChecks", len(results))
	}

	return results
}

func checkTools(p *detector.Platform, logger *slog.Logger) []CheckResult {
	if logger != nil {
		logger.Debug("Checking tools and dependencies status")
	}
	var results []CheckResult

	// Check codegraphcontext
	cgcOK := false
	cgcDetail := "codegraphcontext binary not found on PATH"
	if cgcPath := detector.FindCodegraphcontextBin(p); cgcPath != "" {
		cgcOK = true
		cgcDetail = fmt.Sprintf("Found at %s", cgcPath)
	} else if runner.Exists("codegraphcontext") {
		cgcOK = true
		if path, err := exec.LookPath("codegraphcontext"); err == nil {
			cgcDetail = fmt.Sprintf("Found on PATH: %s", path)
		}
	}
	results = append(results, CheckResult{
		Name:   "codegraphcontext",
		OK:     cgcOK,
		Detail: cgcDetail,
		Fix:    "yap install --only=tools",
	})

	// Check uv
	uvOK := runner.Exists("uv")
	uvDetail := "uv binary not found on PATH"
	if uvOK {
		if path, err := exec.LookPath("uv"); err == nil {
			uvDetail = fmt.Sprintf("Found on PATH: %s", path)
		}
	}
	results = append(results, CheckResult{
		Name:   "uv",
		OK:     uvOK,
		Detail: uvDetail,
		Fix:    "yap install --only=deps",
	})

	// Check graphify (binary on PATH / ~/.local/bin, uv tool, or python package)
	graphifyOK := false
	graphifyDetail := "graphifyy not found or not installed in uv/python"
	if graphifyBin := detector.FindGraphifyBin(p); graphifyBin != "" {
		graphifyOK = true
		graphifyDetail = fmt.Sprintf("Found at %s", graphifyBin)
	} else if runner.Exists("graphify") {
		graphifyOK = true
		if path, err := exec.LookPath("graphify"); err == nil {
			graphifyDetail = fmt.Sprintf("Found on PATH: %s", path)
		}
	}
	if !graphifyOK && uvOK {
		out, err := runner.RunAndCapture("uv", "tool", "list")
		if err == nil && strings.Contains(out, "graphifyy") {
			graphifyOK = true
			graphifyDetail = "Installed as uv tool"
		}
	}
	if !graphifyOK {
		pythonBin := "python3"
		if p.OS == "windows" {
			pythonBin = "python"
		}
		_, err := runner.RunAndCapture(pythonBin, "-c", "import graphify.serve")
		if err == nil {
			graphifyOK = true
			graphifyDetail = fmt.Sprintf("Found in %s packages", pythonBin)
		}
	}
	results = append(results, CheckResult{
		Name:   "graphifyy",
		OK:     graphifyOK,
		Detail: graphifyDetail,
		Fix:    "yap install --only=tools",
	})

	return results
}

func checkConfigs(p *detector.Platform, logger *slog.Logger) []CheckResult {
	if logger != nil {
		logger.Debug("Checking MCP configuration files status")
	}
	var results []CheckResult

	type cfgCheck struct {
		name     string
		path     string
		required bool
		merge    bool // read via Merge-safe helper (Claude Code ~/.claude.json)
	}

	checks := []cfgCheck{
		{"Config: Cursor MCP", filepath.Join(p.HomeDir, ".cursor", "mcp.json"), true, false},
		{"Config: Gemini/Antigravity MCP", filepath.Join(p.HomeDir, ".gemini", "config", "mcp_config.json"), true, false},
		{"Config: Claude Code user MCP", filepath.Join(p.HomeDir, ".claude.json"), true, true},
	}

	if p.OS == "windows" {
		appdata := os.Getenv("APPDATA")
		checks = append(checks,
			cfgCheck{"Config: Claude Desktop", filepath.Join(appdata, "Claude", "claude_desktop_config.json"), false, false},
		)
	} else {
		checks = append(checks,
			cfgCheck{"Config: Claude Desktop", filepath.Join(p.HomeDir, ".config", "Claude", "claude_desktop_config.json"), false, false},
		)
		if p.OS == "darwin" {
			checks = append(checks, cfgCheck{
				"Config: Claude Desktop (macOS)",
				filepath.Join(p.HomeDir, "Library", "Application Support", "Claude", "claude_desktop_config.json"),
				false, false,
			})
		}
	}

	// Project-scoped configs when cwd is inside a project
	if root := detectStatusProjectRoot(); root != "" {
		checks = append(checks,
			cfgCheck{"Config: project .mcp.json", filepath.Join(root, ".mcp.json"), true, false},
			cfgCheck{"Config: project .agents MCP", filepath.Join(root, ".agents", "mcp_config.json"), true, false},
			cfgCheck{"Config: project Cursor MCP", filepath.Join(root, ".cursor", "mcp.json"), true, false},
		)
	}

	for _, c := range checks {
		if _, err := os.Stat(c.path); os.IsNotExist(err) {
			if c.required {
				results = append(results, CheckResult{
					Name:   c.name,
					OK:     false,
					Detail: fmt.Sprintf("does not exist: %s", c.path),
					Fix:    "yap install --only=config",
				})
			}
			continue
		}

		var servers map[string]jsonutil.MCPServer
		var err error
		if c.merge {
			servers, err = jsonutil.ReadMCPServersFromFile(c.path)
		} else {
			cfg, e := jsonutil.ReadOrCreate(c.path)
			err = e
			if cfg != nil {
				servers = cfg.MCPServers
			}
		}
		if err != nil {
			results = append(results, CheckResult{
				Name:   c.name,
				OK:     false,
				Detail: fmt.Sprintf("Error reading config: %v", err),
				Fix:    "yap install --only=config",
			})
			continue
		}

		cgcServer, cgcExist := servers["CodeGraphContext"]
		graphifyServer, graphifyExist := servers["Graphify"]
		cgcValid := cgcExist && strings.Contains(cgcServer.Command, "yap") && len(cgcServer.Args) >= 2 && cgcServer.Args[0] == "proxy" && cgcServer.Args[1] == "cgc"
		graphifyValid := graphifyExist && strings.Contains(graphifyServer.Command, "yap") && len(graphifyServer.Args) >= 2 && graphifyServer.Args[0] == "proxy" && graphifyServer.Args[1] == "graphify"

		if cgcValid && graphifyValid {
			results = append(results, CheckResult{
				Name:   c.name,
				OK:     true,
				Detail: "CodeGraphContext & Graphify properly configured to use yap proxy",
			})
		} else {
			var details []string
			if !cgcValid {
				details = append(details, "CodeGraphContext missing or not pointing to 'yap proxy cgc'")
			}
			if !graphifyValid {
				details = append(details, "Graphify missing or not pointing to 'yap proxy graphify'")
			}
			results = append(results, CheckResult{
				Name:   c.name,
				OK:     false,
				Detail: strings.Join(details, ", "),
				Fix:    "yap install --only=config",
			})
		}
	}

	results = append(results, checkZedContextServers(p)...)
	return results
}

func checkZedContextServers(p *detector.Platform) []CheckResult {
	var path string
	if p.OS == "windows" {
		path = filepath.Join(os.Getenv("APPDATA"), "Zed", "settings.json")
	} else {
		path = filepath.Join(p.HomeDir, ".config", "zed", "settings.json")
	}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return []CheckResult{{
			Name:   "Config: Zed context_servers",
			OK:     false,
			Detail: fmt.Sprintf("missing: %s", path),
			Fix:    "yap install --only=config",
		}}
	}
	clean := data
	var settings map[string]interface{}
	if err := json.Unmarshal(clean, &settings); err != nil {
		// Zed may have comments; best-effort: treat as incomplete
		return []CheckResult{{
			Name:   "Config: Zed context_servers",
			OK:     false,
			Detail: "could not parse settings.json",
			Fix:    "yap install --only=config",
		}}
	}
	cs, _ := settings["context_servers"].(map[string]interface{})
	okCGC := zedEntryLooksLikeYap(cs, "yap-cgc", "cgc")
	okG := zedEntryLooksLikeYap(cs, "yap-graphify", "graphify")
	if okCGC && okG {
		return []CheckResult{{
			Name:   "Config: Zed context_servers",
			OK:     true,
			Detail: fmt.Sprintf("Found at %s", path),
		}}
	}
	return []CheckResult{{
		Name:   "Config: Zed context_servers",
		OK:     false,
		Detail: "yap-cgc / yap-graphify missing or misconfigured",
		Fix:    "yap install --only=config",
	}}
}

func zedEntryLooksLikeYap(cs map[string]interface{}, key, proxy string) bool {
	if cs == nil {
		return false
	}
	m, ok := cs[key].(map[string]interface{})
	if !ok {
		return false
	}
	cmd, _ := m["command"].(string)
	if !strings.Contains(cmd, "yap") {
		return false
	}
	args, ok := m["args"].([]interface{})
	if !ok || len(args) < 2 {
		return false
	}
	a0, _ := args[0].(string)
	a1, _ := args[1].(string)
	return a0 == "proxy" && a1 == proxy
}

func detectStatusProjectRoot() string {
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

func checkIDESkills(p *detector.Platform, logger *slog.Logger) []CheckResult {
	if logger != nil {
		logger.Debug("Checking IDE skills and rules")
	}
	var results []CheckResult

	type skillCheck struct {
		name string
		path string
		need string
	}
	checks := []skillCheck{
		{"Skill: Cursor /yap", filepath.Join(p.HomeDir, ".cursor", "skills", "yap", "SKILL.md"), "name: yap"},
		{"Rule: Cursor yap", filepath.Join(p.HomeDir, ".cursor", "rules", "yap-context.mdc"), "alwaysApply: true"},
		{"Skill: Gemini /yap", filepath.Join(p.HomeDir, ".gemini", "config", "skills", "yap", "SKILL.md"), "name: yap"},
		{"Skill: Claude Code /yap", filepath.Join(p.HomeDir, ".claude", "skills", "yap", "SKILL.md"), "name: yap"},
		{"Skill: Agents /yap", filepath.Join(p.HomeDir, ".agents", "skills", "yap", "SKILL.md"), "name: yap"},
	}
	if root := detectStatusProjectRoot(); root != "" {
		checks = append(checks,
			skillCheck{"Skill: project Claude /yap", filepath.Join(root, ".claude", "skills", "yap", "SKILL.md"), "name: yap"},
			skillCheck{"Skill: project Agents /yap", filepath.Join(root, ".agents", "skills", "yap", "SKILL.md"), "name: yap"},
			skillCheck{"Skill: project Cursor /yap", filepath.Join(root, ".cursor", "skills", "yap", "SKILL.md"), "name: yap"},
		)
	}

	for _, c := range checks {
		data, err := ioutil.ReadFile(c.path)
		if err != nil {
			results = append(results, CheckResult{
				Name:   c.name,
				OK:     false,
				Detail: fmt.Sprintf("missing: %s", c.path),
				Fix:    "yap install --only=config",
			})
			continue
		}
		if !strings.Contains(string(data), c.need) {
			results = append(results, CheckResult{
				Name:   c.name,
				OK:     false,
				Detail: "file exists but content looks incomplete",
				Fix:    "yap install --only=config",
			})
			continue
		}
		results = append(results, CheckResult{
			Name:   c.name,
			OK:     true,
			Detail: fmt.Sprintf("Found at %s", c.path),
		})
	}

	return results
}

func checkPatches(p *detector.Platform, logger *slog.Logger) []CheckResult {
	if logger != nil {
		logger.Debug("Checking CodeGraphContext Python patches status")
	}
	var results []CheckResult

	venvDir := detector.FindPipxVenv(p)
	if venvDir == "" {
		results = append(results, CheckResult{
			Name:   "Patches",
			OK:     false,
			Detail: fmt.Sprintf("codegraphcontext pipx venv not found (searched %v). Cannot check patches.", detector.PipxVenvDirs(p)),
			Fix:    "yap install --only=tools",
		})
		return results
	}

	// Glob matching for server.py and database_kuzu.py
	var serverPy string
	var kuzuPy string

	err := filepath.Walk(venvDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			if strings.HasSuffix(path, filepath.Join("codegraphcontext", "server.py")) {
				serverPy = path
			}
			if strings.HasSuffix(path, filepath.Join("codegraphcontext", "core", "database_kuzu.py")) {
				kuzuPy = path
			}
		}
		return nil
	})

	if err != nil || serverPy == "" || kuzuPy == "" {
		results = append(results, CheckResult{
			Name:   "Patches",
			OK:     false,
			Detail: "codegraphcontext source files (server.py or database_kuzu.py) not found in venv",
			Fix:    "yap install --only=patch",
		})
		return results
	}

	// Verify server.py patch
	serverOK := false
	if serverData, err := ioutil.ReadFile(serverPy); err == nil {
		content := string(serverData)
		if strings.Contains(content, "CGC_RUNTIME_DB_PATH") && strings.Contains(content, "protocolVersion") {
			serverOK = true
		}
	}

	// Verify database_kuzu.py patch
	kuzuOK := false
	if kuzuData, err := ioutil.ReadFile(kuzuPy); err == nil {
		content := string(kuzuData)
		if strings.Contains(content, "_is_read_only") && strings.Contains(content, "read_only=True") {
			kuzuOK = true
		}
	}

	if serverOK && kuzuOK {
		results = append(results, CheckResult{
			Name:   "Patches",
			OK:     true,
			Detail: "server.py and database_kuzu.py patches are active",
		})
	} else {
		var details []string
		if !serverOK {
			details = append(details, "server.py patch missing")
		}
		if !kuzuOK {
			details = append(details, "database_kuzu.py patch missing")
		}
		results = append(results, CheckResult{
			Name:   "Patches",
			OK:     false,
			Detail: strings.Join(details, ", "),
			Fix:    "yap install --only=patch",
		})
	}

	return results
}

func checkLiveness(p *detector.Platform, logger *slog.Logger) []CheckResult {
	if logger != nil {
		logger.Debug("Checking MCP proxy liveness status")
	}
	var results []CheckResult

	// Try to locate 'yap' binary to test proxy
	yapBin, err := exec.LookPath("yap")
	if err != nil {
		// Use current executable if not in path
		yapBin, err = os.Executable()
		if err != nil {
			yapBin = "./yap"
		}
	}

	// Test CodeGraphContext proxy liveness
	results = append(results, testProxyLiveness(yapBin, "cgc", logger))

	// Test Graphify proxy liveness (only if a graph exists somewhere in cache/workspace)
	results = append(results, testProxyLiveness(yapBin, "graphify", logger))

	return results
}

func testProxyLiveness(yapBin, service string, logger *slog.Logger) CheckResult {
	if logger != nil {
		logger.Debug("Testing proxy liveness", "service", service, "yapBin", yapBin)
	}
	cmd := exec.Command(yapBin, "proxy", service)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return CheckResult{
			Name:   fmt.Sprintf("Liveness: %s", service),
			OK:     false,
			Detail: fmt.Sprintf("Failed to open stdin pipe: %v", err),
			Fix:    "Verify tool installations",
		}
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return CheckResult{
			Name:   fmt.Sprintf("Liveness: %s", service),
			OK:     false,
			Detail: fmt.Sprintf("Failed to open stdout pipe: %v", err),
			Fix:    "Verify tool installations",
		}
	}

	if err := cmd.Start(); err != nil {
		return CheckResult{
			Name:   fmt.Sprintf("Liveness: %s", service),
			OK:     false,
			Detail: fmt.Sprintf("Failed to start proxy subprocess: %v", err),
			Fix:    "yap install --only=tools",
		}
	}

	// Send an initialize request JSON-RPC
	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"processId":1234,"clientInfo":{"name":"status-check","version":"1.0.0"}}}`

	// Format as LSP framed if necessary, but raw JSON is also supported by our proxy.
	// Write raw JSON.
	_, _ = io.WriteString(stdin, initReq+"\n")

	done := make(chan bool)
	var output []byte
	go func() {
		buf := make([]byte, 1024)
		n, err := stdout.Read(buf)
		if err == nil && n > 0 {
			output = buf[:n]
		}
		done <- true
	}()

	select {
	case <-done:
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		if len(output) > 0 && (bytes.Contains(output, []byte("result")) || bytes.Contains(output, []byte("capabilities")) || bytes.Contains(output, []byte("error"))) {
			return CheckResult{
				Name:   fmt.Sprintf("Liveness: %s", service),
				OK:     true,
				Detail: "Subprocess responded successfully to initialize request",
			}
		}
		return CheckResult{
			Name:   fmt.Sprintf("Liveness: %s", service),
			OK:     false,
			Detail: fmt.Sprintf("Invalid or empty response from subprocess: %s", string(output)),
			Fix:    "Check logs in ~/.cache/ for detail",
		}
	case <-time.After(3 * time.Second):
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		return CheckResult{
			Name:   fmt.Sprintf("Liveness: %s", service),
			OK:     false,
			Detail: "Timeout waiting for initialize response (3s)",
			Fix:    "Check tool installation or missing graph path for graphify",
		}
	}
}

// PrintStatus prints CheckResults nicely.
func PrintStatus(results []CheckResult) {
	fmt.Println("\n====================================================")
	fmt.Println("     Yap AI Performance Analyzer - Status Report")
	fmt.Println("====================================================")

	allOK := true
	for _, res := range results {
		statusStr := "\x1b[31m✗\x1b[0m"
		if res.OK {
			statusStr = "\x1b[32m✓\x1b[0m"
		} else {
			allOK = false
		}
		fmt.Printf(" %s  %-25s — %s\n", statusStr, res.Name, res.Detail)
		if !res.OK && res.Fix != "" {
			fmt.Printf("     \x1b[33m→ Düzelt:\x1b[0m %s\n", res.Fix)
		}
	}

	fmt.Println("\n----------------------------------------------------")
	if allOK {
		fmt.Println(" Genel Durum: \x1b[32m✓ SAĞLIKLI (Tüm kontroller başarılı)\x1b[0m")
	} else {
		fmt.Println(" Genel Durum: \x1b[33m⚠ KISMİ SORUN (Kontrollerden bazıları başarısız)\x1b[0m")
	}
	fmt.Println("====================================================")
}
