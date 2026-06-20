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
	if _, err := os.Stat(filepath.Join(dir, "project.godot")); err == nil {
		info.Language = "GDScript (Godot)"
	} else if hasUnrealProject(dir) {
		info.Language = "C++ (Unreal Engine)"
	} else if isUnityProject(dir) {
		info.Language = "C# (Unity)"
	} else if _, err := os.Stat(filepath.Join(dir, "package.json")); err == nil {
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
	} else if _, err := os.Stat(filepath.Join(dir, "composer.json")); err == nil {
		info.Language = "PHP"
		scanComposerJSON(filepath.Join(dir, "composer.json"), info)
	} else if _, err := os.Stat(filepath.Join(dir, "pubspec.yaml")); err == nil {
		info.Language = "Dart/Flutter"
		scanPubspecYAML(filepath.Join(dir, "pubspec.yaml"), info)
	} else if _, err := os.Stat(filepath.Join(dir, "pom.xml")); err == nil {
		info.Language = "Java (Maven)"
	} else if _, err := os.Stat(filepath.Join(dir, "build.gradle")); err == nil {
		info.Language = "Java (Gradle)"
	} else if _, err := os.Stat(filepath.Join(dir, "CMakeLists.txt")); err == nil {
		info.Language = "C++"
	} else if _, err := os.Stat(filepath.Join(dir, "Package.swift")); err == nil {
		info.Language = "Swift"
	} else if _, err := os.Stat(filepath.Join(dir, "Gemfile")); err == nil {
		info.Language = "Ruby"
	} else if _, err := os.Stat(filepath.Join(dir, "build.sbt")); err == nil {
		info.Language = "Scala"
	} else if _, err := os.Stat(filepath.Join(dir, "Makefile")); err == nil {
		info.Language = "Makefile/C"
	}

	// 3. Fallback: Extension based language detection if config files are not present
	if info.Language == "Unknown" {
		info.Language = detectLanguageByExtensions(dir)
	}

	return info, nil
}

func hasUnrealProject(dir string) bool {
	files, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".uproject") {
			return true
		}
	}
	return false
}

func isUnityProject(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, "ProjectSettings", "ProjectVersion.txt")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(dir, "Assets")); err == nil {
		if _, err := os.Stat(filepath.Join(dir, "ProjectSettings")); err == nil {
			return true
		}
	}
	return false
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

func scanComposerJSON(path string, info *ProjectInfo) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var comp struct {
		Name    string                 `json:"name"`
		Require map[string]interface{} `json:"require"`
	}
	if err := json.Unmarshal(data, &comp); err != nil {
		return
	}
	info.ModuleName = comp.Name
	count := 0
	for dep := range comp.Require {
		if dep == "php" {
			continue
		}
		if count < 15 {
			info.Dependencies = append(info.Dependencies, dep)
			count++
		}
	}
}

func scanPubspecYAML(path string, info *ProjectInfo) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inDependencies := false
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if strings.HasPrefix(trimmed, "name:") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				info.ModuleName = strings.TrimSpace(parts[1])
			}
		}

		if trimmed == "dependencies:" {
			inDependencies = true
			continue
		} else if inDependencies && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			inDependencies = false
		}

		if inDependencies {
			if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") {
				parts := strings.SplitN(trimmed, ":", 2)
				depName := strings.TrimSpace(parts[0])
				if depName != "flutter" && len(info.Dependencies) < 15 {
					info.Dependencies = append(info.Dependencies, depName)
				}
			}
		}
	}
}

func detectLanguageByExtensions(dir string) string {
	files, err := os.ReadDir(dir)
	if err != nil {
		return "Unknown"
	}

	extCounts := make(map[string]int)
	for _, f := range files {
		if f.IsDir() {
			name := f.Name()
			if name == ".git" || name == "node_modules" || name == "dist" || name == "bin" || name == "vendor" || name == "build" {
				continue
			}
			subFiles, err := os.ReadDir(filepath.Join(dir, name))
			if err == nil {
				for _, sf := range subFiles {
					if !sf.IsDir() {
						ext := filepath.Ext(sf.Name())
						if ext != "" {
							extCounts[ext]++
						}
					}
				}
			}
			continue
		}

		ext := filepath.Ext(f.Name())
		if ext != "" {
			extCounts[ext]++
		}
	}

	if len(extCounts) == 0 {
		return "Unknown"
	}

	maxExt := ""
	maxCount := 0
	for ext, count := range extCounts {
		if count > maxCount {
			maxCount = count
			maxExt = ext
		}
	}

	switch strings.ToLower(maxExt) {
	case ".abap":
		return "ABAP"
	case ".as":
		return "ActionScript"
	case ".adb", ".ads":
		return "Ada"
	case ".agda":
		return "Agda"
	case ".alg":
		return "Algol"
	case ".cls":
		return "Apex"
	case ".apl":
		return "APL"
	case ".asm", ".s":
		return "Assembly"
	case ".awk":
		return "Awk"
	case ".bal":
		return "Ballerina"
	case ".sh", ".bash":
		return "Shell Script"
	case ".bas":
		return "BASIC"
	case ".bat", ".cmd":
		return "Batch"
	case ".bb":
		return "BlitzBasic"
	case ".boo":
		return "Boo"
	case ".bf":
		return "Brainfuck"
	case ".c":
		return "C"
	case ".cs":
		return "C#"
	case ".cpp", ".cc", ".cxx", ".hpp", ".h":
		return "C++"
	case ".ceylon":
		return "Ceylon"
	case ".chpl":
		return "Chapel"
	case ".clw":
		return "Clarion"
	case ".clj", ".cljs", ".cljc", ".edn":
		return "Clojure"
	case ".cob", ".cbl":
		return "COBOL"
	case ".coffee":
		return "CoffeeScript"
	case ".cfm":
		return "ColdFusion"
	case ".lisp", ".lsp", ".cl":
		return "Lisp"
	case ".cr":
		return "Crystal"
	case ".css":
		return "CSS"
	case ".d":
		return "D"
	case ".dart":
		return "Dart"
	case ".pas", ".dfm", ".dpr":
		return "Delphi"
	case ".dylan":
		return "Dylan"
	case ".e":
		return "E"
	case ".ex", ".exs":
		return "Elixir"
	case ".elm":
		return "Elm"
	case ".erl", ".hrl":
		return "Erlang"
	case ".fs", ".fsi", ".fsx":
		return "F#"
	case ".fth", ".4th":
		return "Forth"
	case ".f90", ".f", ".for":
		return "Fortran"
	case ".prg":
		return "FoxPro"
	case ".gd":
		return "GDScript (.gd)"
	case ".glsl", ".vert", ".frag":
		return "GLSL"
	case ".gml":
		return "GML (GameMaker Language)"
	case ".go":
		return "Go (Golang)"
	case ".groovy", ".gvy":
		return "Groovy"
	case ".hack", ".hh":
		return "Hack"
	case ".hs", ".lhs":
		return "Haskell"
	case ".hx":
		return "Haxe"
	case ".hlsl", ".fx":
		return "HLSL"
	case ".html", ".htm":
		return "HTML"
	case ".idr", ".lidr":
		return "Idris"
	case ".inf", ".ni":
		return "Inform"
	case ".ijs":
		return "J"
	case ".java":
		return "Java"
	case ".js", ".jsx", ".mjs":
		return "JavaScript"
	case ".jinja", ".jinja2":
		return "Jinja"
	case ".jl":
		return "Julia"
	case ".kt", ".kts":
		return "Kotlin"
	case ".vi":
		return "LabVIEW"
	case ".tex", ".sty":
		return "LaTeX"
	case ".ls":
		return "LiveScript"
	case ".lgo":
		return "Logo"
	case ".lol":
		return "LOLCODE"
	case ".lua":
		return "Lua"
	case ".mb":
		return "Malbolge"
	case ".md", ".markdown":
		return "Markdown"
	case ".m":
		return "MATLAB / Objective-C"
	case ".nim":
		return "Nim"
	case ".nix":
		return "Nix"
	case ".mm":
		return "Objective-C++"
	case ".ml", ".mli":
		return "OCaml"
	case ".opencl":
		return "OpenCL"
	case ".pl":
		return "Perl / Prolog"
	case ".pls", ".plsql":
		return "PL/SQL"
	case ".ps":
		return "PostScript"
	case ".ps1", ".psm1":
		return "PowerShell"
	case ".pro":
		return "Prolog"
	case ".pd":
		return "Pure Data"
	case ".purs":
		return "PureScript"
	case ".py", ".ipynb":
		return "Python"
	case ".r", ".R":
		return "R"
	case ".rkt":
		return "Racket"
	case ".raku", ".pm6":
		return "Raku"
	case ".re":
		return "Reason"
	case ".red":
		return "Red"
	case ".rb":
		return "Ruby"
	case ".rs":
		return "Rust"
	case ".sas":
		return "SAS"
	case ".scala":
		return "Scala"
	case ".scm", ".ss":
		return "Scheme"
	case ".sb3", ".sb2":
		return "Scratch"
	case ".st":
		return "Smalltalk"
	case ".sol":
		return "Solidity"
	case ".sql":
		return "SQL"
	case ".swift":
		return "Swift"
	case ".tcl":
		return "Tcl"
	case ".ts", ".tsx":
		return "TypeScript"
	case ".v":
		return "V / Verilog"
	case ".vala":
		return "Vala"
	case ".vbs":
		return "VBScript"
	case ".vhd", ".vhdl":
		return "VHDL"
	case ".vb":
		return "Visual Basic"
	case ".vy":
		return "Vyper"
	case ".wasm", ".wat":
		return "WebAssembly (Wasm)"
	case ".ws":
		return "Whitespace"
	case ".wl", ".wls", ".nb":
		return "Wolfram Language"
	case ".xml":
		return "XML"
	case ".yaml", ".yml":
		return "YAML"
	case ".zig":
		return "Zig"
	case ".gdshader":
		return "Godot Shader"
	case ".shader":
		return "ShaderLab (Unity)"
	case ".uproject":
		return "Unreal Engine Project"
	}

	return "Unknown"
}
