package jsonutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMergeMCPServersPreservesOtherKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude.json")
	initial := map[string]interface{}{
		"userID":       "abc",
		"oauthAccount": map[string]interface{}{"email": "x@y.z"},
		"mcpServers": map[string]interface{}{
			"Other": map[string]interface{}{"command": "echo", "args": []string{"hi"}},
		},
	}
	raw, _ := json.MarshalIndent(initial, "", "  ")
	if err := os.WriteFile(path, raw, 0644); err != nil {
		t.Fatal(err)
	}

	servers := map[string]MCPServer{
		"CodeGraphContext": {Command: "/bin/yap", Args: []string{"proxy", "cgc"}},
		"Graphify":         {Command: "/bin/yap", Args: []string{"proxy", "graphify"}},
	}
	if err := MergeMCPServers(false, path, servers); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatal(err)
	}
	if root["userID"] != "abc" {
		t.Fatalf("userID lost: %v", root["userID"])
	}
	if _, ok := root["oauthAccount"]; !ok {
		t.Fatal("oauthAccount lost")
	}
	ms := root["mcpServers"].(map[string]interface{})
	if _, ok := ms["Other"]; !ok {
		t.Fatal("existing Other server lost")
	}
	cgc := ms["CodeGraphContext"].(map[string]interface{})
	if cgc["command"] != "/bin/yap" {
		t.Fatalf("cgc command=%v", cgc["command"])
	}

	got, err := ReadMCPServersFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got["Graphify"].Args[1] != "graphify" {
		t.Fatalf("Graphify args=%v", got["Graphify"].Args)
	}
}

func TestMergeMCPServersCreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", ".claude.json")
	servers := map[string]MCPServer{
		"CodeGraphContext": {Command: "yap", Args: []string{"proxy", "cgc"}},
	}
	if err := MergeMCPServers(false, path, servers); err != nil {
		t.Fatal(err)
	}
	got, err := ReadMCPServersFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got["CodeGraphContext"].Command != "yap" {
		t.Fatalf("got %+v", got)
	}
}
