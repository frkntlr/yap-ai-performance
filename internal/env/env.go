package env

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Load reads a .env file in the specified directory and loads key-value pairs into system environment variables.
// If the file does not exist, it returns nil.
func Load(dir string) error {
	path := filepath.Join(dir, ".env")
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		// Trim optional surrounding quotes
		val = strings.Trim(val, `"'`)

		if key != "" {
			os.Setenv(key, val)
		}
	}

	return scanner.Err()
}
