package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanGoProject(t *testing.T) {
	tmpDir := t.TempDir()

	goModContent := `module github.com/test/my-go-project

go 1.22

require (
	github.com/spf13/cobra v1.8.0
	golang.org/x/sys v0.10.0
)
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644)
	if err != nil {
		t.Fatalf("failed to write dummy go.mod: %v", err)
	}

	info, err := Scan(tmpDir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if info.Language != "Go" {
		t.Errorf("expected Language to be 'Go', got %q", info.Language)
	}

	if info.ModuleName != "github.com/test/my-go-project" {
		t.Errorf("expected ModuleName to be 'github.com/test/my-go-project', got %q", info.ModuleName)
	}

	if len(info.Dependencies) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(info.Dependencies))
	}
}

func TestScanNodeProject(t *testing.T) {
	tmpDir := t.TempDir()

	packageJSONContent := `{
  "name": "my-node-app",
  "dependencies": {
    "express": "^4.18.2",
    "react": "^18.2.0"
  },
  "devDependencies": {
    "vite": "^4.4.5",
    "eslint": "^8.45.0"
  }
}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSONContent), 0644)
	if err != nil {
		t.Fatalf("failed to write dummy package.json: %v", err)
	}

	info, err := Scan(tmpDir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if info.Language != "JavaScript/TypeScript" {
		t.Errorf("expected Language to be 'JavaScript/TypeScript', got %q", info.Language)
	}

	if info.ModuleName != "my-node-app" {
		t.Errorf("expected ModuleName to be 'my-node-app', got %q", info.ModuleName)
	}

	if len(info.Frameworks) != 2 {
		t.Errorf("expected 2 frameworks (react, vite), got %d: %v", len(info.Frameworks), info.Frameworks)
	}
}

func TestScanExtensionFallback(t *testing.T) {
	tmpDir := t.TempDir()

	// Write 3 C++ files
	_ = os.WriteFile(filepath.Join(tmpDir, "main.cpp"), []byte("int main() {}"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "helper.cpp"), []byte(""), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "helper.h"), []byte(""), 0644)

	// Write 1 python file (less than cpp)
	_ = os.WriteFile(filepath.Join(tmpDir, "script.py"), []byte(""), 0644)

	info, err := Scan(tmpDir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if info.Language != "C++" {
		t.Errorf("expected Language to be 'C++', got %q", info.Language)
	}
}

func TestScanGameEngines(t *testing.T) {
	// 1. Godot
	godotDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(godotDir, "project.godot"), []byte(""), 0644)
	info, err := Scan(godotDir)
	if err != nil || info.Language != "GDScript (Godot)" {
		t.Errorf("Godot detection failed: %v, %v", err, info.Language)
	}

	// 2. Unreal Engine
	unrealDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(unrealDir, "MyGame.uproject"), []byte(""), 0644)
	info, err = Scan(unrealDir)
	if err != nil || info.Language != "C++ (Unreal Engine)" {
		t.Errorf("Unreal Engine detection failed: %v, %v", err, info.Language)
	}

	// 3. Unity
	unityDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(unityDir, "ProjectSettings"), 0755)
	_ = os.WriteFile(filepath.Join(unityDir, "ProjectSettings", "ProjectVersion.txt"), []byte(""), 0644)
	info, err = Scan(unityDir)
	if err != nil || info.Language != "C# (Unity)" {
		t.Errorf("Unity detection failed: %v, %v", err, info.Language)
	}
}
