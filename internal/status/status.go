package status

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
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
func RunStatus(p *detector.Platform) []CheckResult {
	var results []CheckResult

	// 1. Tool Checks
	results = append(results, checkTools(p)...)

	// 2. Config Checks
	results = append(results, checkConfigs(p)...)

	// 3. Patch Checks
	results = append(results, checkPatches(p)...)

	// 4. Liveness Checks (only if tools are present)
	results = append(results, checkLiveness(p)...)

	return results
}

func checkTools(p *detector.Platform) []CheckResult {
	var results []CheckResult

	// Check codegraphcontext
	cgcOK := false
	cgcDetail := "codegraphcontext binary not found on PATH"
	cgcPath := filepath.Join(p.HomeDir, ".local", "bin", "codegraphcontext")
	if p.OS == "windows" {
		cgcPath = filepath.Join(p.LocalBin, "codegraphcontext.exe") // or standard python location
	}
	if _, err := os.Stat(cgcPath); err == nil {
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

	// Check graphifyy[mcp] (via uv tool list or python)
	graphifyOK := false
	graphifyDetail := "graphifyy not found or not installed in uv/python"
	if uvOK {
		out, err := runner.RunAndCapture("uv", "tool", "list")
		if err == nil && strings.Contains(out, "graphifyy") {
			graphifyOK = true
			graphifyDetail = "Installed as uv tool"
		}
	}
	if !graphifyOK {
		// Check python site-packages or just running it
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

func checkConfigs(p *detector.Platform) []CheckResult {
	var results []CheckResult
	var configPaths []string

	if p.OS == "windows" {
		appdata := os.Getenv("APPDATA")
		localappdata := os.Getenv("LOCALAPPDATA")
		configPaths = []string{
			filepath.Join(p.HomeDir, ".gemini", "config", "mcp_config.json"),
			filepath.Join(appdata, "Claude", "claude_desktop_config.json"),
			filepath.Join(appdata, "Cursor", "User", "globalStorage", "saoudrizwan.claude-dev", "settings", "cline_mcp_settings.json"),
			filepath.Join(localappdata, "Programs", "cursor", "resources", "app", "extensions", "saoudrizwan.claude-dev", "settings", "cline_mcp_settings.json"),
		}
	} else {
		configPaths = []string{
			filepath.Join(p.HomeDir, ".gemini", "config", "mcp_config.json"),
			filepath.Join(p.HomeDir, ".config", "Cursor", "User", "globalStorage", "saoudrizwan.claude-dev", "settings", "cline_mcp_settings.json"),
			filepath.Join(p.HomeDir, ".config", "Code", "User", "globalStorage", "saoudrizwan.claude-dev", "settings", "cline_mcp_settings.json"),
			filepath.Join(p.HomeDir, ".config", "Claude", "claude_desktop_config.json"),
		}
	}

	for _, path := range configPaths {
		isGemini := strings.Contains(path, "gemini")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if isGemini {
				results = append(results, CheckResult{
					Name:   fmt.Sprintf("Config: %s", filepath.Base(path)),
					OK:     false,
					Detail: "Gemini config does not exist but is required",
					Fix:    "yap install --only=config",
				})
			}
			continue
		}

		cfg, err := jsonutil.ReadOrCreate(path)
		if err != nil {
			results = append(results, CheckResult{
				Name:   fmt.Sprintf("Config: %s", filepath.Base(path)),
				OK:     false,
				Detail: fmt.Sprintf("Error reading config: %v", err),
				Fix:    "yap install --only=config",
			})
			continue
		}

		cgcServer, cgcExist := cfg.MCPServers["CodeGraphContext"]
		graphifyServer, graphifyExist := cfg.MCPServers["Graphify"]

		cgcValid := cgcExist && strings.Contains(cgcServer.Command, "yap") && len(cgcServer.Args) > 0 && cgcServer.Args[0] == "proxy" && cgcServer.Args[1] == "cgc"
		graphifyValid := graphifyExist && strings.Contains(graphifyServer.Command, "yap") && len(graphifyServer.Args) > 0 && graphifyServer.Args[0] == "proxy" && graphifyServer.Args[1] == "graphify"

		if cgcValid && graphifyValid {
			results = append(results, CheckResult{
				Name:   fmt.Sprintf("Config: %s", filepath.Base(path)),
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
				Name:   fmt.Sprintf("Config: %s", filepath.Base(path)),
				OK:     false,
				Detail: strings.Join(details, ", "),
				Fix:    "yap install --only=config",
			})
		}
	}

	return results
}

func checkPatches(p *detector.Platform) []CheckResult {
	var results []CheckResult

	// Try to find the site-packages codegraphcontext path
	venvDir := filepath.Join(p.HomeDir, ".local", "share", "pipx", "venvs", "codegraphcontext")
	if p.OS == "windows" {
		venvDir = filepath.Join(p.HomeDir, ".local", "share", "pipx", "venvs", "codegraphcontext")
	}

	if _, err := os.Stat(venvDir); os.IsNotExist(err) {
		results = append(results, CheckResult{
			Name:   "Patches",
			OK:     false,
			Detail: "codegraphcontext pipx venv not found. Cannot check patches.",
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

func checkLiveness(p *detector.Platform) []CheckResult {
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
	results = append(results, testProxyLiveness(yapBin, "cgc"))

	// Test Graphify proxy liveness (only if a graph exists somewhere in cache/workspace)
	results = append(results, testProxyLiveness(yapBin, "graphify"))

	return results
}

func testProxyLiveness(yapBin, service string) CheckResult {
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
