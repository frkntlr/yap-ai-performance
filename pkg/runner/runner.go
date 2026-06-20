package runner

import (
	"bytes"
	"os"
	"os/exec"
)

// Exists checks if a command is available on the system PATH.
func Exists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// Run executes a command and streams its output directly to stdout/stderr.
func Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RunInDir executes a command in a specific directory, streaming output to stdout/stderr.
func RunInDir(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RunAndCapture executes a command and captures its combined stdout and stderr output.
func RunAndCapture(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}

// RunInDirAndCapture executes a command in a specific directory and captures its combined output.
func RunInDirAndCapture(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}
