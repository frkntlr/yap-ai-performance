package fileutil

import (
	"io"
	"os"
	"path/filepath"
)

// Exists checks if a file or directory exists.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// CopyFile copies a file from src to dst.
func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}

	return out.Sync()
}

// EnsureDir makes sure that a directory exists, creating it if necessary.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}
