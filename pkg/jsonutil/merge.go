package jsonutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/frkntlr/yap-ai-performance/internal/dryrun"
)

// MergeMCPServers updates only the mcpServers object in a JSON file,
// preserving all other top-level keys. Safe for Claude Code ~/.claude.json.
func MergeMCPServers(dryRun bool, path string, servers map[string]MCPServer) error {
	if dryRun {
		dryrun.PrintSimulation(fmt.Sprintf("%s içinde mcpServers birleştirilecek", path))
		return nil
	}

	root := map[string]interface{}{}
	if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &root); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
	}

	existingRaw, _ := root["mcpServers"].(map[string]interface{})
	if existingRaw == nil {
		existingRaw = map[string]interface{}{}
	}

	for name, srv := range servers {
		entry := map[string]interface{}{
			"command": srv.Command,
			"args":    srv.Args,
		}
		if len(srv.Env) > 0 {
			env := map[string]interface{}{}
			for k, v := range srv.Env {
				env[k] = v
			}
			entry["env"] = env
		}
		existingRaw[name] = entry
	}
	root["mcpServers"] = existingRaw

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0644)
}

// ReadMCPServersFromFile extracts mcpServers from a JSON file that may contain
// other top-level keys (e.g. ~/.claude.json). Returns empty map if missing.
func ReadMCPServersFromFile(path string) (map[string]MCPServer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]MCPServer{}, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return map[string]MCPServer{}, nil
	}
	var root map[string]interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	raw, _ := root["mcpServers"].(map[string]interface{})
	out := make(map[string]MCPServer)
	for name, v := range raw {
		m, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		srv := MCPServer{}
		if c, ok := m["command"].(string); ok {
			srv.Command = c
		}
		if args, ok := m["args"].([]interface{}); ok {
			for _, a := range args {
				if s, ok := a.(string); ok {
					srv.Args = append(srv.Args, s)
				}
			}
		}
		out[name] = srv
	}
	return out, nil
}
