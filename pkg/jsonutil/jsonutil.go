package jsonutil

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
)

type MCPServer struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

type MCPConfig struct {
	MCPServers map[string]MCPServer `json:"mcpServers"`
}

// ReadOrCreate reads the MCP configuration file at the given path.
// If the file does not exist, it returns a new empty MCPConfig structure.
func ReadOrCreate(path string) (*MCPConfig, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &MCPConfig{
			MCPServers: make(map[string]MCPServer),
		}, nil
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config MCPConfig
	if err := json.Unmarshal(data, &config); err != nil {
		// If JSON is corrupted or invalid, return a new one to prevent blocking,
		// but backing up could be useful. For now we will overwrite or return error.
		if len(data) == 0 {
			return &MCPConfig{
				MCPServers: make(map[string]MCPServer),
			}, nil
		}
		return nil, err
	}

	if config.MCPServers == nil {
		config.MCPServers = make(map[string]MCPServer)
	}

	return &config, nil
}

// Write writes the MCPConfig structure to the file at the given path with nice indentation.
func Write(path string, config *MCPConfig) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, data, 0644)
}
