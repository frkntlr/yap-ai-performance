package detector

import (
	"os"
	"path/filepath"
	"strings"
)

// FirstExisting returns the first path that exists on disk, or "" if none do.
func FirstExisting(candidates []string) string {
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

func localAppData(p *Platform) string {
	if v := os.Getenv("LOCALAPPDATA"); v != "" {
		return v
	}
	if p != nil && p.HomeDir != "" {
		return filepath.Join(p.HomeDir, "AppData", "Local")
	}
	return ""
}

func exeSuffix(p *Platform) string {
	if p != nil && p.OS == "windows" {
		return ".exe"
	}
	return ""
}

// PipxVenvDirs returns candidate pipx venv directories for codegraphcontext.
func PipxVenvDirs(p *Platform) []string {
	if p == nil {
		return nil
	}
	home := p.HomeDir
	name := "codegraphcontext"
	candidates := []string{
		filepath.Join(home, ".local", "share", "pipx", "venvs", name),
	}
	if p.OS == "windows" {
		la := localAppData(p)
		candidates = append(candidates,
			filepath.Join(la, "pipx", "venvs", name),
			filepath.Join(la, "pipx", "pipx", "venvs", name),
			filepath.Join(home, ".local", "pipx", "venvs", name),
		)
	}
	return uniqueNonEmpty(candidates)
}

// FindPipxVenv returns the first existing codegraphcontext pipx venv, or "".
func FindPipxVenv(p *Platform) string {
	return FirstExisting(PipxVenvDirs(p))
}

// CodegraphcontextBins returns candidate paths for the codegraphcontext CLI.
func CodegraphcontextBins(p *Platform) []string {
	if p == nil {
		return nil
	}
	home := p.HomeDir
	suf := exeSuffix(p)
	base := "codegraphcontext" + suf
	candidates := []string{
		filepath.Join(home, ".local", "bin", base),
	}
	if p.LocalBin != "" {
		candidates = append(candidates, filepath.Join(p.LocalBin, base))
	}
	if p.OS == "windows" {
		la := localAppData(p)
		candidates = append(candidates,
			filepath.Join(home, ".local", "bin", "codegraphcontext.exe"),
			filepath.Join(la, "Programs", "Python", "Python312", "Scripts", "codegraphcontext.exe"),
			filepath.Join(la, "Programs", "Python", "Python313", "Scripts", "codegraphcontext.exe"),
			filepath.Join(la, "pipx", "venvs", "codegraphcontext", "Scripts", "codegraphcontext.exe"),
			filepath.Join(la, "pipx", "pipx", "venvs", "codegraphcontext", "Scripts", "codegraphcontext.exe"),
			filepath.Join(home, ".local", "share", "pipx", "venvs", "codegraphcontext", "Scripts", "codegraphcontext.exe"),
		)
	} else {
		candidates = append(candidates,
			filepath.Join(home, ".local", "share", "pipx", "venvs", "codegraphcontext", "bin", "codegraphcontext"),
		)
	}
	return uniqueNonEmpty(candidates)
}

// FindCodegraphcontextBin returns the first existing CGC binary path, or "".
func FindCodegraphcontextBin(p *Platform) string {
	return FirstExisting(CodegraphcontextBins(p))
}

// GraphifyBins returns candidate paths for the graphify CLI.
func GraphifyBins(p *Platform) []string {
	if p == nil {
		return nil
	}
	home := p.HomeDir
	suf := exeSuffix(p)
	base := "graphify" + suf
	candidates := []string{
		filepath.Join(home, ".local", "bin", base),
	}
	if p.LocalBin != "" {
		candidates = append(candidates, filepath.Join(p.LocalBin, base))
	}
	if p.OS == "windows" {
		la := localAppData(p)
		candidates = append(candidates,
			filepath.Join(home, ".local", "bin", "graphify.exe"),
			filepath.Join(la, "uv", "tools", "graphifyy", "Scripts", "graphify.exe"),
			filepath.Join(home, ".local", "share", "uv", "tools", "graphifyy", "Scripts", "graphify.exe"),
			filepath.Join(la, "Programs", "uv", "tools", "graphifyy", "Scripts", "graphify.exe"),
		)
	} else {
		candidates = append(candidates,
			filepath.Join(home, ".local", "share", "uv", "tools", "graphifyy", "bin", "graphify"),
		)
	}
	return uniqueNonEmpty(candidates)
}

// FindGraphifyBin returns the first existing graphify CLI path, or "".
func FindGraphifyBin(p *Platform) string {
	return FirstExisting(GraphifyBins(p))
}

// GraphifyPythonBins returns candidate python interpreters for graphify.serve.
func GraphifyPythonBins(p *Platform) []string {
	if p == nil {
		return nil
	}
	home := p.HomeDir
	if p.OS == "windows" {
		la := localAppData(p)
		return uniqueNonEmpty([]string{
			filepath.Join(home, ".local", "share", "uv", "tools", "graphifyy", "Scripts", "python.exe"),
			filepath.Join(la, "uv", "tools", "graphifyy", "Scripts", "python.exe"),
			filepath.Join(la, "Programs", "uv", "tools", "graphifyy", "Scripts", "python.exe"),
			filepath.Join(home, ".local", "share", "uv", "tools", "graphifyy", "bin", "python.exe"),
		})
	}
	return uniqueNonEmpty([]string{
		filepath.Join(home, ".local", "share", "uv", "tools", "graphifyy", "bin", "python"),
	})
}

// FindGraphifyPython returns the first existing uv-tool python for graphify, or "".
func FindGraphifyPython(p *Platform) string {
	return FirstExisting(GraphifyPythonBins(p))
}

func uniqueNonEmpty(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// Normalize for dedup on Windows without changing returned path form.
		key := p
		if len(key) > 1 && (key[1] == ':' || strings.Contains(key, `\`) || strings.Contains(key, `/`)) {
			key = strings.ToLower(filepath.Clean(p))
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, p)
	}
	return out
}
