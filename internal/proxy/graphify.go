package proxy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
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

func findGraph(workspaceRoot string, logFile io.Writer) string {
	// 1. Priority: Workspace local graph
	if workspaceRoot != "" {
		candidate := filepath.Join(workspaceRoot, "graphify-out", "graph.json")
		if _, err := os.Stat(candidate); err == nil {
			cache := loadCache()
			cache[workspaceRoot] = candidate
			saveCache(cache)
			_, _ = fmt.Fprintf(logFile, "Workspace local graph found: %s\n", candidate)
			return candidate
		}
	}

	// 2. Priority: Env var
	envPath := os.Getenv("GRAPHIFY_GRAPH_PATH")
	if envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			_, _ = fmt.Fprintf(logFile, "Graph path from env var: %s\n", envPath)
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
		_, _ = fmt.Fprintf(logFile, "No local graph found. Using fallback from cache: %s\n", fallback)
		return fallback
	}

	_, _ = fmt.Fprintln(logFile, "Error: No graphify-out/graph.json found anywhere.")
	return ""
}

// RunGraphifyProxy starts the Graphify proxy server.
func RunGraphifyProxy() error {
	logDir := filepath.Join(os.Getenv("HOME"), ".cache")
	_ = os.MkdirAll(logDir, 0755)
	logFile, err := os.OpenFile(filepath.Join(logDir, "graphify-mcp.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		defer logFile.Close()
		log.SetOutput(logFile)
	}

	log.Println("Graphify Proxy Wrapper started.")

	stdinReader := bufio.NewReader(os.Stdin)
	headers, body, isFramed, err := readFirstMessage(stdinReader)
	if err != nil {
		log.Printf("Error reading first message: %v\n", err)
	} else if len(body) > 0 {
		log.Printf("Intercepted first message: %s\n", string(body))
	}

	workspaceRoot := parseWorkspaceRoot(body)
	graphPath := findGraph(workspaceRoot, logFile)
	if graphPath == "" {
		log.Println("No graph found. Server cannot start.")
		return fmt.Errorf("no graph found")
	}

	home, _ := os.UserHomeDir()
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
		pythonBin = filepath.Join(home, ".local", "share", "uv", "tools", "graphifyy", "bin", "python")
		if _, err := os.Stat(pythonBin); os.IsNotExist(err) {
			// Fallback
			pythonBin = "python3"
		}
		cmdArgs = []string{"-m", "graphify.serve", graphPath}
	}

	if len(os.Args) > 3 {
		cmdArgs = append(cmdArgs, os.Args[3:]...)
	}

	log.Printf("Starting subprocess: %s %v\n", pythonBin, cmdArgs)
	cmd := exec.Command(pythonBin, cmdArgs...)
	cmd.Dir = workspaceRoot
	cmd.Stderr = logFile

	subStdin, err := cmd.StdinPipe()
	if err != nil {
		log.Printf("Failed to get stdin pipe: %v\n", err)
		return err
	}

	subStdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("Failed to get stdout pipe: %v\n", err)
		return err
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start subprocess: %v\n", err)
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

	log.Println("Graphify Proxy Wrapper terminated.")
	return nil
}
