package proxy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// readFirstMessage reads the first JSON-RPC message from r.
// It supports both raw JSON-RPC (delimited by matching braces) and LSP-framed messages.
func readFirstMessage(r *bufio.Reader) (headers []byte, body []byte, isFramed bool, err error) {
	// Read until first non-whitespace character
	var firstChar byte
	for {
		b, err := r.ReadByte()
		if err != nil {
			return nil, nil, false, err
		}
		if b != '\r' && b != '\n' && b != ' ' && b != '\t' {
			firstChar = b
			break
		}
	}

	if firstChar == '{' {
		// Unframed raw JSON - count braces
		var content bytes.Buffer
		content.WriteByte(firstChar)
		braceCount := 1
		inString := false
		escaped := false

		for braceCount > 0 {
			b, err := r.ReadByte()
			if err != nil {
				return nil, nil, false, err
			}
			content.WriteByte(b)

			if inString {
				if escaped {
					escaped = false
				} else if b == '\\' {
					escaped = true
				} else if b == '"' {
					inString = false
				}
			} else {
				if b == '"' {
					inString = true
				} else if b == '{' {
					braceCount++
				} else if b == '}' {
					braceCount--
				}
			}
		}
		return nil, content.Bytes(), false, nil
	}

	// Framed LSP message (Content-Length: X...)
	// We read line by line until we find the empty line separating headers and body
	var headerBuf bytes.Buffer
	headerBuf.WriteByte(firstChar)

	// Read rest of the first line
	firstLineRest, err := r.ReadBytes('\n')
	if err != nil {
		return nil, nil, false, err
	}
	headerBuf.Write(firstLineRest)

	// Content-Length parsing
	var contentLength int64
	lineStr := string(headerBuf.Bytes())
	if strings.HasPrefix(strings.ToLower(lineStr), "content-length:") {
		parts := strings.Split(lineStr, ":")
		if len(parts) > 1 {
			val := strings.TrimSpace(parts[1])
			if parsedVal, err := strconv.ParseInt(val, 10, 64); err == nil {
				contentLength = parsedVal
			}
		}
	}

	// Read remaining headers
	for {
		line, err := r.ReadBytes('\n')
		if err != nil {
			return nil, nil, false, err
		}
		headerBuf.Write(line)

		lineStr := string(line)
		if strings.HasPrefix(strings.ToLower(lineStr), "content-length:") {
			parts := strings.Split(lineStr, ":")
			if len(parts) > 1 {
				val := strings.TrimSpace(parts[1])
				if parsedVal, err := strconv.ParseInt(val, 10, 64); err == nil {
					contentLength = parsedVal
				}
			}
		}

		if lineStr == "\r\n" || lineStr == "\n" {
			break
		}
	}

	if contentLength <= 0 {
		return headerBuf.Bytes(), nil, true, fmt.Errorf("invalid content-length: %d", contentLength)
	}

	// Read exact body
	body = make([]byte, contentLength)
	_, err = io.ReadFull(r, body)
	if err != nil {
		return headerBuf.Bytes(), nil, true, err
	}

	return headerBuf.Bytes(), body, true, nil
}

// parseWorkspaceRoot parses the initialize request body to extract the workspace root directory.
func parseWorkspaceRoot(body []byte) string {
	var req struct {
		Method string `json:"method"`
		Params struct {
			RootPath string `json:"rootPath"`
			RootURI  string `json:"rootUri"`
		} `json:"params"`
	}

	if err := json.Unmarshal(body, &req); err != nil {
		return ""
	}

	if req.Method != "initialize" {
		return ""
	}

	if req.Params.RootPath != "" {
		return req.Params.RootPath
	}

	if req.Params.RootURI != "" {
		parsed, err := url.Parse(req.Params.RootURI)
		if err == nil && parsed.Scheme == "file" {
			return parsed.Path
		}
	}

	return ""
}

// forward pipes data from src to dst.
func forward(src io.Reader, dst io.Writer, wg *sync.WaitGroup) {
	defer wg.Done()
	_, _ = io.Copy(dst, src)
	// Try to close output if it implements Close
	if c, ok := dst.(io.WriteCloser); ok {
		_ = c.Close()
	}
}

// RunCGCProxy starts the CodeGraphContext proxy server.
func RunCGCProxy() error {
	logDir := filepath.Join(os.Getenv("HOME"), ".cache")
	_ = os.MkdirAll(logDir, 0755)
	logFile, err := os.OpenFile(filepath.Join(logDir, "codegraphcontext-mcp.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		defer logFile.Close()
		log.SetOutput(logFile)
	}

	log.Println("CGC Proxy Wrapper started.")

	stdinReader := bufio.NewReader(os.Stdin)
	headers, body, isFramed, err := readFirstMessage(stdinReader)
	if err != nil {
		log.Printf("Error reading first message: %v\n", err)
	} else if len(body) > 0 {
		log.Printf("Intercepted first message: %s\n", string(body))
	}

	workspaceRoot := parseWorkspaceRoot(body)
	var dbPath string
	env := os.Environ()

	if workspaceRoot != "" {
		dbPath = filepath.Join(workspaceRoot, ".codegraphcontext_db")
		env = append(env, fmt.Sprintf("CGC_RUNTIME_DB_PATH=%s", dbPath))
		log.Printf("Dynamic workspace root detected: %s\n", workspaceRoot)
		log.Printf("Setting database path to: %s\n", dbPath)
	} else {
		dbPath = ".codegraphcontext_db"
		env = append(env, fmt.Sprintf("CGC_RUNTIME_DB_PATH=%s", dbPath))
		log.Println("No workspace root detected. Using relative fallback.")
	}

	// Locate codegraphcontext binary. Usually in ~/.local/bin or on PATH.
	cgcPath := filepath.Join(os.Getenv("HOME"), ".local", "bin", "codegraphcontext")
	if _, err := os.Stat(cgcPath); os.IsNotExist(err) {
		// Try system path
		if path, err := exec.LookPath("codegraphcontext"); err == nil {
			cgcPath = path
		}
	}

	log.Printf("Starting subprocess: %s mcp start\n", cgcPath)
	cmdArgs := []string{"mcp", "start"}
	if len(os.Args) > 3 {
		cmdArgs = append(cmdArgs, os.Args[3:]...)
	}

	cmd := exec.Command(cgcPath, cmdArgs...)
	cmd.Dir = workspaceRoot
	cmd.Env = env
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

	log.Println("CGC Proxy Wrapper terminated.")
	return nil
}
