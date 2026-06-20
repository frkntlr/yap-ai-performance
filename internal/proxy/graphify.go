package proxy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/frkntlr/yap-ai-performance/internal/logger"
)

type Cache map[string]string

func getCacheFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "graphify-mcp-cache.json")
}

func loadCache() Cache {
	cachePath := getCacheFilePath()
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return make(Cache)
	}

	data, err := ioutil.ReadFile(cachePath)
	if err != nil {
		return make(Cache)
	}

	var cache Cache
	if err := json.Unmarshal(data, &cache); err != nil {
		return make(Cache)
	}
	return cache
}

func saveCache(cache Cache) {
	cachePath := getCacheFilePath()
	_ = os.MkdirAll(filepath.Dir(cachePath), 0755)
	if data, err := json.MarshalIndent(cache, "", "  "); err == nil {
		_ = ioutil.WriteFile(cachePath, data, 0644)
	}
}

func findGraph(workspaceRoot string, loggerInst *slog.Logger) string {
	// 1. Priority: Workspace local graph
	if workspaceRoot != "" {
		candidate := filepath.Join(workspaceRoot, "graphify-out", "graph.json")
		if _, err := os.Stat(candidate); err == nil {
			cache := loadCache()
			cache[workspaceRoot] = candidate
			saveCache(cache)
			loggerInst.Info("Workspace local graph found", "path", candidate)
			return candidate
		}
	}

	// 2. Priority: Env var
	envPath := os.Getenv("GRAPHIFY_GRAPH_PATH")
	if envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			loggerInst.Info("Graph path from env var", "path", envPath)
			return envPath
		}
	}

	// 3. Priority: Cache check
	cache := loadCache()
	var validPaths []string
	for cachedDir, cachedFile := range cache {
		if _, err := os.Stat(cachedFile); err == nil {
			validPaths = append(validPaths, cachedFile)
		} else {
			delete(cache, cachedDir)
		}
	}
	saveCache(cache)

	if len(validPaths) > 0 {
		// Sort by modification time descending (newest first)
		sort.Slice(validPaths, func(i, j int) bool {
			fi, erri := os.Stat(validPaths[i])
			fj, errj := os.Stat(validPaths[j])
			if erri == nil && errj == nil {
				return fi.ModTime().After(fj.ModTime())
			}
			return false
		})
		fallback := validPaths[0]
		loggerInst.Info("No local graph found, using fallback from cache", "path", fallback)
		return fallback
	}

	loggerInst.Error("No graphify-out/graph.json found anywhere")
	return ""
}

// RunGraphifyProxy starts the Graphify proxy server.
func RunGraphifyProxy() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	// Initialize the custom dual logger
	loggerInst, err := logger.Init(homeDir)
	if err != nil {
		// Fallback to default slog on failure
		loggerInst = slog.Default()
	}

	// Open daily log file also for capturing subprocess Stderr raw output
	logDir := filepath.Join(homeDir, ".yap", "logs")
	_ = os.MkdirAll(logDir, 0755)
	dateStr := time.Now().Format("2006-01-02")
	logPath := filepath.Join(logDir, fmt.Sprintf("yap-%s.log", dateStr))
	subLogFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err == nil {
		defer subLogFile.Close()
	}

	loggerInst.Info("Graphify Proxy Wrapper started")

	stdinReader := bufio.NewReader(os.Stdin)
	headers, body, isFramed, err := readFirstMessage(stdinReader)
	if err != nil {
		loggerInst.Error("Error reading first message", "error", err)
	} else if len(body) > 0 {
		loggerInst.Debug("Intercepted first message", "body", string(body))
	}

	workspaceRoot := parseWorkspaceRoot(body)
	graphPath := findGraph(workspaceRoot, loggerInst)
	if graphPath == "" {
		loggerInst.Error("No graph found. Server cannot start.")
		return fmt.Errorf("no graph found")
	}

	var pythonBin string
	var cmdArgs []string

	if runtime.GOOS == "windows" {
		// On windows, find python from path or default winget locations
		pythonBin = "python"
		// If uv environment is used, it could be located in uv tools
		localAppData := os.Getenv("LOCALAPPDATA")
		uvPython := filepath.Join(localAppData, "Programs", "uv", "tools", "graphifyy", "Scripts", "python.exe")
		if _, err := os.Stat(uvPython); err == nil {
			pythonBin = uvPython
		}
		cmdArgs = []string{"-m", "graphify.serve", graphPath}
	} else {
		pythonBin = filepath.Join(homeDir, ".local", "share", "uv", "tools", "graphifyy", "bin", "python")
		if _, err := os.Stat(pythonBin); os.IsNotExist(err) {
			// Fallback
			pythonBin = "python3"
		}
		cmdArgs = []string{"-m", "graphify.serve", graphPath}
	}

	if len(os.Args) > 3 {
		cmdArgs = append(cmdArgs, os.Args[3:]...)
	}

	loggerInst.Info("Starting subprocess", "path", pythonBin, "args", cmdArgs)
	cmd := exec.Command(pythonBin, cmdArgs...)
	cmd.Dir = workspaceRoot
	if subLogFile != nil {
		cmd.Stderr = subLogFile
	} else {
		cmd.Stderr = os.Stderr
	}

	subStdin, err := cmd.StdinPipe()
	if err != nil {
		loggerInst.Error("Failed to get stdin pipe", "error", err)
		return err
	}

	subStdout, err := cmd.StdoutPipe()
	if err != nil {
		loggerInst.Error("Failed to get stdout pipe", "error", err)
		return err
	}

	if err := cmd.Start(); err != nil {
		loggerInst.Error("Failed to start subprocess", "error", err)
		return err
	}

	// Write intercepted first message back to subprocess stdin
	if isFramed {
		if len(headers) > 0 {
			_, _ = subStdin.Write(headers)
		}
		if len(body) > 0 {
			_, _ = subStdin.Write(body)
		}
	} else {
		if len(body) > 0 {
			if !bytes.HasSuffix(body, []byte("\n")) {
				body = append(body, '\n')
			}
			_, _ = subStdin.Write(body)
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// Stream rest of stdin to subprocess stdin
	go forward(stdinReader, subStdin, &wg)

	// Stream subprocess stdout to os.Stdout
	go forward(subStdout, os.Stdout, &wg)

	_ = cmd.Wait()
	wg.Wait()

	loggerInst.Info("Graphify Proxy Wrapper terminated")
	return nil
}
