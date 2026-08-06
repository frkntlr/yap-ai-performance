package detector

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFirstExisting(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	if err := os.WriteFile(b, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := FirstExisting([]string{a, b}); got != b {
		t.Fatalf("got %q want %q", got, b)
	}
	if got := FirstExisting([]string{a, filepath.Join(dir, "missing")}); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestPipxVenvDirsWindows(t *testing.T) {
	home := t.TempDir()
	la := filepath.Join(home, "AppData", "Local")
	t.Setenv("LOCALAPPDATA", la)

	p := &Platform{OS: "windows", HomeDir: home, LocalBin: filepath.Join(la, "Programs", "yap")}
	dirs := PipxVenvDirs(p)
	want := []string{
		filepath.Join(home, ".local", "share", "pipx", "venvs", "codegraphcontext"),
		filepath.Join(la, "pipx", "venvs", "codegraphcontext"),
		filepath.Join(la, "pipx", "pipx", "venvs", "codegraphcontext"),
		filepath.Join(home, ".local", "pipx", "venvs", "codegraphcontext"),
	}
	if len(dirs) != len(want) {
		t.Fatalf("len=%d want %d: %v", len(dirs), len(want), dirs)
	}
	for i := range want {
		if dirs[i] != want[i] {
			t.Errorf("[%d] got %q want %q", i, dirs[i], want[i])
		}
	}

	// Create one and ensure FindPipxVenv resolves it.
	target := want[1]
	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatal(err)
	}
	if got := FindPipxVenv(p); got != target {
		t.Fatalf("FindPipxVenv=%q want %q", got, target)
	}
}

func TestPipxVenvDirsLinux(t *testing.T) {
	home := t.TempDir()
	p := &Platform{OS: "linux", HomeDir: home, LocalBin: filepath.Join(home, ".local", "bin")}
	dirs := PipxVenvDirs(p)
	if len(dirs) != 1 {
		t.Fatalf("expected 1 linux candidate, got %v", dirs)
	}
	want := filepath.Join(home, ".local", "share", "pipx", "venvs", "codegraphcontext")
	if dirs[0] != want {
		t.Fatalf("got %q want %q", dirs[0], want)
	}
}

func TestGraphifyPythonBinsWindows(t *testing.T) {
	home := t.TempDir()
	la := filepath.Join(home, "AppData", "Local")
	t.Setenv("LOCALAPPDATA", la)
	p := &Platform{OS: "windows", HomeDir: home}
	bins := GraphifyPythonBins(p)
	if len(bins) < 3 {
		t.Fatalf("expected multiple windows python candidates, got %v", bins)
	}
	// Prefer user .local share path first (matches install.ps1 / uv default).
	wantFirst := filepath.Join(home, ".local", "share", "uv", "tools", "graphifyy", "Scripts", "python.exe")
	if bins[0] != wantFirst {
		t.Fatalf("first candidate %q want %q", bins[0], wantFirst)
	}
}

func TestCodegraphcontextBinsWindowsExe(t *testing.T) {
	home := t.TempDir()
	la := filepath.Join(home, "AppData", "Local")
	t.Setenv("LOCALAPPDATA", la)
	p := &Platform{OS: "windows", HomeDir: home, LocalBin: filepath.Join(la, "Programs", "yap")}
	bins := CodegraphcontextBins(p)
	foundExe := false
	for _, b := range bins {
		if filepath.Ext(b) == ".exe" {
			foundExe = true
			break
		}
	}
	if !foundExe {
		t.Fatalf("expected .exe candidates, got %v", bins)
	}
}

func TestGraphifyBinsIncludesLocalBin(t *testing.T) {
	home := t.TempDir()
	local := filepath.Join(home, "Programs", "yap")
	p := &Platform{OS: "windows", HomeDir: home, LocalBin: local}
	bins := GraphifyBins(p)
	want := filepath.Join(local, "graphify.exe")
	found := false
	for _, b := range bins {
		if b == want {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("LocalBin candidate missing: %v", bins)
	}
}
