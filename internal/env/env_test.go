package env

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnv(t *testing.T) {
	tmpDir := t.TempDir()

	envContent := `
# A comment line
YAP_TEST_VAR=my-val
YAP_ANOTHER_VAR="another value"
`
	err := os.WriteFile(filepath.Join(tmpDir, ".env"), []byte(envContent), 0644)
	if err != nil {
		t.Fatalf("failed to write dummy .env: %v", err)
	}

	err = Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if os.Getenv("YAP_TEST_VAR") != "my-val" {
		t.Errorf("expected YAP_TEST_VAR='my-val', got %q", os.Getenv("YAP_TEST_VAR"))
	}

	if os.Getenv("YAP_ANOTHER_VAR") != "another value" {
		t.Errorf("expected YAP_ANOTHER_VAR='another value', got %q", os.Getenv("YAP_ANOTHER_VAR"))
	}
}
