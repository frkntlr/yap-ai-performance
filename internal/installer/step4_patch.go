package installer

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/frkntlr/yap-ai-performance/internal/detector"
)

// Step4Patch applies necessary runtime patches to CodeGraphContext codebase.
func Step4Patch(p *detector.Platform) error {
	venvDir := filepath.Join(p.HomeDir, ".local", "share", "pipx", "venvs", "codegraphcontext")

	if _, err := os.Stat(venvDir); os.IsNotExist(err) {
		return fmt.Errorf("codegraphcontext pipx venv not found at %s. Please install tools first", venvDir)
	}

	var serverPy string
	var kuzuPy string
	var pyFiles []string

	// Find python files in venv
	err := filepath.Walk(venvDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(path, ".py") {
			pyFiles = append(pyFiles, path)
			if strings.HasSuffix(path, filepath.Join("codegraphcontext", "server.py")) {
				serverPy = path
			}
			if strings.HasSuffix(path, filepath.Join("codegraphcontext", "core", "database_kuzu.py")) {
				kuzuPy = path
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk venv directory: %v", err)
	}

	// 1. Patch server.py
	if serverPy != "" {
		fmt.Printf("Patching server.py: %s\n", serverPy)
		if err := patchServerPy(serverPy); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("server.py not found in venv")
	}

	// 2. Patch Console() to Console(stderr=True) in all .py files
	patchedConsoleCount := 0
	for _, pyFile := range pyFiles {
		data, err := ioutil.ReadFile(pyFile)
		if err != nil {
			continue
		}
		content := string(data)
		if strings.Contains(content, "Console()") {
			newContent := strings.ReplaceAll(content, "Console()", "Console(stderr=True)")
			if err := ioutil.WriteFile(pyFile, []byte(newContent), 0644); err == nil {
				patchedConsoleCount++
			}
		}
	}
	fmt.Printf("Patched Console() to Console(stderr=True) in %d files.\n", patchedConsoleCount)

	// 3. Patch database_kuzu.py
	if kuzuPy != "" {
		fmt.Printf("Patching database_kuzu.py: %s\n", kuzuPy)
		if err := patchDatabaseKuzuPy(kuzuPy); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("database_kuzu.py not found in venv")
	}

	return nil
}

func patchServerPy(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(data)
	patched := false

	// Target 1: db_path override
	target1 := "self.db_manager = get_database_manager(db_path=ctx.db_path)"
	replacement1 := "db_path = os.getenv(\"CGC_RUNTIME_DB_PATH\") or ctx.db_path\n            self.db_manager = get_database_manager(db_path=db_path)"
	if strings.Contains(content, target1) {
		content = strings.ReplaceAll(content, target1, replacement1)
		patched = true
	}

	// Target 2: protocol version negotiation
	target2 := `"protocolVersion": "2025-03-26"`
	replacement2 := `"protocolVersion": params.get("protocolVersion", "2024-11-05")`
	if strings.Contains(content, target2) {
		content = strings.ReplaceAll(content, target2, replacement2)
		patched = true
	}

	// Target 3: empty instructions to prevent LLM crashes
	target3 := `"instructions": LLM_SYSTEM_PROMPT`
	replacement3 := `"instructions": ""`
	if strings.Contains(content, target3) {
		content = strings.ReplaceAll(content, target3, replacement3)
		patched = true
	}

	if patched {
		err = ioutil.WriteFile(path, []byte(content), 0644)
		if err != nil {
			return err
		}
		fmt.Println("server.py patches applied successfully!")
	} else {
		fmt.Println("server.py patches already applied or targets not found.")
	}

	return nil
}

func patchDatabaseKuzuPy(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(data)
	patched := false

	// Target 1: Initialize _is_read_only in __init__
	target1 := "self.db_path = new_db_path"
	replacement1 := "self.db_path = new_db_path\n        self._is_read_only = False"
	if strings.Contains(content, target1) && !strings.Contains(content, "self._is_read_only = False") {
		content = strings.ReplaceAll(content, target1, replacement1)
		patched = true
	}

	// Target 2: Try-except read-only fallback
	dbPattern := regexp.MustCompile(`(\s+)self\._db = kuzu\.Database\(self\.db_path\)`)
	dbReplacement := `$1try:
$1    self._db = kuzu.Database(self.db_path)
$1    self._is_read_only = False
$1except RuntimeError as re:
$1    if "lock" in str(re).lower():
$1        info_logger("KùzuDB is locked. Opening in read_only mode.")
$1        self._db = kuzu.Database(self.db_path, read_only=True)
$1        self._is_read_only = True
$1    else:
$1        raise`

	if dbPattern.MatchString(content) && !strings.Contains(content, "except RuntimeError as re:") {
		content = dbPattern.ReplaceAllString(content, dbReplacement)
		patched = true
	}

	// Target 3: Skip schema init if read-only
	// Matches schema initialization block
	schemaPattern := regexp.MustCompile(`(\s+)# Use one connection from the pool to initialise schema\r?\n\s+temp_conn = self\._pool\.get\(\)\r?\n\s+try:\r?\n\s+self\._conn = temp_conn[^\n]*\r?\n\s+self\._initialize_schema\(\)\r?\n\s+self\._conn = None\r?\n\s+finally:\r?\n\s+self\._pool\.put\(temp_conn\)`)
	
	if schemaPattern.MatchString(content) && !strings.Contains(content, "Skipping schema init") {
		content = schemaPattern.ReplaceAllStringFunc(content, func(match string) string {
			// Extract indentation
			lines := strings.Split(match, "\n")
			indent := ""
			if len(lines) > 0 {
				indent = lines[0][:len(lines[0])-len(strings.TrimSpace(lines[0]))]
			}
			
			// Indent the original block
			indentedBlock := ""
			for _, line := range lines {
				if strings.TrimSpace(line) != "" {
					indentedBlock += indent + "    " + line + "\n"
				} else {
					indentedBlock += "\n"
				}
			}
			
			return fmt.Sprintf("%sif not getattr(self, '_is_read_only', False):\n%s%selse:\n%s    info_logger(\"Skipping schema init due to read-only mode\")", 
				indent, indentedBlock, indent, indent)
		})
		patched = true
	}

	if patched {
		err = ioutil.WriteFile(path, []byte(content), 0644)
		if err != nil {
			return err
		}
		fmt.Println("database_kuzu.py patched successfully!")
	} else {
		fmt.Println("database_kuzu.py patches already applied or targets not found.")
	}

	return nil
}
