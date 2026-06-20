package backup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type BackupEntry struct {
	OriginalPath string
	BackupPath   string
	Timestamp    time.Time
}

type Manager struct {
	BackupDir string // ~/.yap/backups/
}

// NewManager creates a new Backup Manager instance.
func NewManager(homeDir string) *Manager {
	return &Manager{
		BackupDir: filepath.Join(homeDir, ".yap", "backups"),
	}
}

// encodePath encodes a full filesystem path into a safe filename.
func encodePath(p string) string {
	r := strings.ReplaceAll(p, ":", "_colon_")
	r = strings.ReplaceAll(r, "\\", "_backslash_")
	r = strings.ReplaceAll(r, "/", "_slash_")
	return r
}

// decodePath decodes a safe filename back into a full filesystem path.
func decodePath(s string) string {
	r := strings.ReplaceAll(s, "_slash_", "/")
	r = strings.ReplaceAll(r, "_backslash_", "\\")
	r = strings.ReplaceAll(r, "_colon_", ":")
	return r
}

// Backup copies the file to the backup directory.
// Target file: filePath (e.g. /home/user/.gemini/config/mcp_config.json)
// Backup file: ~/.yap/backups/_slash_home_slash_user_slash_.gemini_slash_config_slash_mcp_config.json.20260620153000.bak
func (m *Manager) Backup(filePath string) error {
	// Clean and get absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("file to backup does not exist: %s", absPath)
	}

	if err := os.MkdirAll(m.BackupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	srcFile, err := os.Open(absPath)
	if err != nil {
		return fmt.Errorf("failed to open source file for backup: %w", err)
	}
	defer srcFile.Close()

	timestamp := time.Now().Format("20060102150405")
	encodedName := encodePath(absPath)
	backupFileName := fmt.Sprintf("%s.%s.bak", encodedName, timestamp)
	backupPath := filepath.Join(m.BackupDir, backupFileName)

	dstFile, err := os.OpenFile(backupPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy data to backup file: %w", err)
	}

	return dstFile.Sync()
}

// ListBackups returns all backup entries for a given original file path.
func (m *Manager) ListBackups(originalPath string) ([]BackupEntry, error) {
	if _, err := os.Stat(m.BackupDir); os.IsNotExist(err) {
		return nil, nil
	}

	absPath, err := filepath.Abs(originalPath)
	if err != nil {
		absPath = originalPath
	}

	files, err := os.ReadDir(m.BackupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	encodedTarget := encodePath(absPath)
	var entries []BackupEntry

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		// Matches format: encodedTarget.YYYYMMDDHHMMSS.bak
		if strings.HasPrefix(name, encodedTarget+".") && strings.HasSuffix(name, ".bak") {
			parts := strings.Split(name, ".")
			if len(parts) >= 3 {
				tsStr := parts[len(parts)-2]
				t, err := time.Parse("20060102150405", tsStr)
				if err == nil {
					entries = append(entries, BackupEntry{
						OriginalPath: absPath,
						BackupPath:   filepath.Join(m.BackupDir, name),
						Timestamp:    t,
					})
				}
			}
		}
	}

	// Sort backups from newest to oldest
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.After(entries[j].Timestamp)
	})

	return entries, nil
}

// RestoreLatest restores the newest backup for a given file to its original location.
func (m *Manager) RestoreLatest(originalPath string) error {
	backups, err := m.ListBackups(originalPath)
	if err != nil {
		return err
	}
	if len(backups) == 0 {
		return fmt.Errorf("no backups found for: %s", originalPath)
	}

	latest := backups[0]

	// Ensure the destination directory exists
	if err := os.MkdirAll(filepath.Dir(originalPath), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	srcFile, err := os.Open(latest.BackupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(originalPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open target file for restoration: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to restore file content: %w", err)
	}

	return dstFile.Sync()
}

// ListAllLatestBackups returns a list of unique original files and their latest backup entries.
func (m *Manager) ListAllLatestBackups() ([]BackupEntry, error) {
	if _, err := os.Stat(m.BackupDir); os.IsNotExist(err) {
		return nil, nil
	}

	files, err := os.ReadDir(m.BackupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	latestMap := make(map[string]BackupEntry)

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		if !strings.HasSuffix(name, ".bak") {
			continue
		}

		parts := strings.Split(name, ".")
		if len(parts) >= 3 {
			tsStr := parts[len(parts)-2]
			t, err := time.Parse("20060102150405", tsStr)
			if err == nil {
				// Reconstruct the encoded path by joining all parts except timestamp and bak
				encodedPathStr := strings.Join(parts[:len(parts)-2], ".")
				origPath := decodePath(encodedPathStr)

				entry := BackupEntry{
					OriginalPath: origPath,
					BackupPath:   filepath.Join(m.BackupDir, name),
					Timestamp:    t,
				}

				if existing, exists := latestMap[origPath]; !exists || t.After(existing.Timestamp) {
					latestMap[origPath] = entry
				}
			}
		}
	}

	var result []BackupEntry
	for _, entry := range latestMap {
		result = append(result, entry)
	}

	// Sort alphabetically by original path
	sort.Slice(result, func(i, j int) bool {
		return result[i].OriginalPath < result[j].OriginalPath
	})

	return result, nil
}
