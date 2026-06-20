package scanner

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// ProjectInfo contains the detected project metadata.
type ProjectInfo struct {
	RootDir       string   `json:"root_dir"`
	Language      string   `json:"language"`       // e.g. "Go", "JavaScript/TypeScript", "Rust", "Python", "Java (Maven)", "Java (Gradle)", "Unknown"
	Frameworks    []string `json:"frameworks"`     // Filtered tools/frameworks
	Dependencies  []string `json:"dependencies"`   // First 15 production dependencies
	ModuleName    string   `json:"module_name"`    // package/module name
	HasDockerfile bool     `json:"has_dockerfile"`
	HasGit        bool     `json:"has_git"`
}

// Scan analyzes the specified directory and returns detected project technologies.
func Scan(dir string) (*ProjectInfo, error) {
	info := &ProjectInfo{
		RootDir:      dir,
		Language:     "Unknown",
		Frameworks:   []string{},
		Dependencies: []string{},
	}

	// 1. Check basic repository structure
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		info.HasGit = true
	}
	if _, err := os.Stat(filepath.Join(dir, "Dockerfile")); err == nil {
		info.HasDockerfile = true
	}

	// 2. Scan language-specific configuration files
	if _, err := os.Stat(filepath.Join(dir, "package.json")); err == nil {
		info.Language = "JavaScript/TypeScript"
		scanPackageJSON(filepath.Join(dir, "package.json"), info)
	} else if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		info.Language = "Go"
		scanGoMod(filepath.Join(dir, "go.mod"), info)
	} else if _, err := os.Stat(filepath.Join(dir, "Cargo.toml")); err == nil {
		info.Language = "Rust"
		scanCargoToml(filepath.Join(dir, "Cargo.toml"), info)
	} else if _, err := os.Stat(filepath.Join(dir, "requirements.txt")); err == nil {
		info.Language = "Python"
		scanRequirementsTxt(filepath.Join(dir, "requirements.txt"), info)
	} else if _, err := os.Stat(filepath.Join(dir, "pyproject.toml")); err == nil {
		info.Language = "Python"
		scanPyprojectToml(filepath.Join(dir, "pyproject.toml"), info)
	} else if _, err := os.Stat(filepath.Join(dir, "pom.xml")); err == nil {
		info.Language = "Java (Maven)"
	} else if _, err := os.Stat(filepath.Join(dir, "build.gradle")); err == nil {
		info.Language = "Java (Gradle)"
	}

	return info, nil
}

func scanPackageJSON(path string, info *ProjectInfo) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var pkg struct {
		Name            string                 `json:"name"`
		Dependencies    map[string]interface{} `json:"dependencies"`
		DevDependencies map[string]interface{} `json:"devDependencies"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return
	}

	info.ModuleName = pkg.Name

	// Framework/Tool signatures to search in devDependencies and dependencies
	signatures := map[string]bool{
		"react":       true,
		"tailwindcss": true,
		"vite":        true,
		"jest":        true,
		"cypress":     true,
		"playwright":  true,
		"vitest":      true,
		"next":        true,
		"nuxt":        true,
		"vue":         true,
		"angular":     true,
		"svelte":      true,
		"solid-js":    true,
		"remix":       true,
		"astro":       true,
	}

	count := 0
	for dep := range pkg.Dependencies {
		if signatures[dep] {
			info.Frameworks = append(info.Frameworks, dep)
		}
		if count < 15 {
			info.Dependencies = append(info.Dependencies, dep)
			count++
		}
	}

	for devDep := range pkg.DevDependencies {
		if signatures[devDep] {
			// Avoid duplicate frameworks
			found := false
			for _, f := range info.Frameworks {
				if f == devDep {
					found = true
					break
				}
			}
			if !found {
				info.Frameworks = append(info.Frameworks, devDep)
			}
		}
	}
}

func scanGoMod(path string, info *ProjectInfo) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			info.ModuleName = strings.TrimPrefix(line, "module ")
			info.ModuleName = strings.Trim(info.ModuleName, "\"'` ")
		} else if strings.HasPrefix(line, "require (") {
			for scanner.Scan() {
				subLine := strings.TrimSpace(scanner.Text())
				if subLine == ")" {
					break
				}
				parts := strings.Fields(subLine)
				if len(parts) > 0 {
					dep := parts[0]
					if len(info.Dependencies) < 15 {
						info.Dependencies = append(info.Dependencies, dep)
					}
				}
			}
		} else if strings.HasPrefix(line, "require ") {
			parts := strings.Fields(line)
			if len(parts) > 1 {
				dep := parts[1]
				if len(info.Dependencies) < 15 {
					info.Dependencies = append(info.Dependencies, dep)
				}
			}
		}
	}
}

func scanCargoToml(path string, info *ProjectInfo) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inPackage := false
	inDependencies := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") {
			inPackage = (line == "[package]")
			inDependencies = (line == "[dependencies]")
			continue
		}

		if inPackage {
			if strings.HasPrefix(line, "name") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					name := strings.TrimSpace(parts[1])
					info.ModuleName = strings.Trim(name, "\"'` ")
				}
			}
		}

		if inDependencies {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) > 0 {
				dep := strings.TrimSpace(parts[0])
				if len(info.Dependencies) < 15 {
					info.Dependencies = append(info.Dependencies, dep)
				}
			}
		}
	}
}

func scanRequirementsTxt(path string, info *ProjectInfo) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		dep := line
		for _, op := range []string{"==", ">=", "<=", "~=", ">", "<"} {
			if idx := strings.Index(line, op); idx != -1 {
				dep = strings.TrimSpace(line[:idx])
				break
			}
		}
		if len(info.Dependencies) < 15 {
			info.Dependencies = append(info.Dependencies, dep)
		}
	}
}

func scanPyprojectToml(path string, info *ProjectInfo) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inDependencies := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") {
			inDependencies = strings.Contains(line, "dependencies")
			continue
		}

		if inDependencies {
			if strings.Contains(line, "=") {
				parts := strings.SplitN(line, "=", 2)
				dep := strings.TrimSpace(parts[0])
				dep = strings.Trim(dep, "\"'`[] ")
				if dep != "" && len(info.Dependencies) < 15 {
					info.Dependencies = append(info.Dependencies, dep)
				}
			} else {
				dep := strings.Trim(line, "\"'`, ")
				if dep != "" && len(info.Dependencies) < 15 {
					info.Dependencies = append(info.Dependencies, dep)
				}
			}
		}
	}
}
