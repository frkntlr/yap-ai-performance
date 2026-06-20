package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// DualHandler routes log records to both a file (in JSON) and stderr (in text).
type DualHandler struct {
	fileHandler slog.Handler
	termHandler slog.Handler
}

// Enabled returns true if either handler handles the given log level.
func (h *DualHandler) Enabled(ctx context.Context, level slog.Level) bool {
	// File handler handles Info (-4 for Debug, 0 for Info, 4 for Warn, 8 for Error)
	// We want to support Debug level in file too, if needed, but Info is the default.
	return level >= slog.LevelInfo
}

// Handle writes the log record to file (INFO+) and stderr (WARN+).
func (h *DualHandler) Handle(ctx context.Context, r slog.Record) error {
	var err1, err2 error
	if r.Level >= slog.LevelInfo && h.fileHandler != nil {
		err1 = h.fileHandler.Handle(ctx, r)
	}
	if r.Level >= slog.LevelWarn && h.termHandler != nil {
		err2 = h.termHandler.Handle(ctx, r)
	}
	if err1 != nil {
		return err1
	}
	return err2
}

// WithAttrs returns a new handler with the given attributes.
func (h *DualHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	var fh, th slog.Handler
	if h.fileHandler != nil {
		fh = h.fileHandler.WithAttrs(attrs)
	}
	if h.termHandler != nil {
		th = h.termHandler.WithAttrs(attrs)
	}
	return &DualHandler{
		fileHandler: fh,
		termHandler: th,
	}
}

// WithGroup returns a new handler with the given group.
func (h *DualHandler) WithGroup(name string) slog.Handler {
	var fh, th slog.Handler
	if h.fileHandler != nil {
		fh = h.fileHandler.WithGroup(name)
	}
	if h.termHandler != nil {
		th = h.termHandler.WithGroup(name)
	}
	return &DualHandler{
		fileHandler: fh,
		termHandler: th,
	}
}

// Init initializes the logger.
// It creates a JSON log file under ~/.yap/logs/yap-YYYY-MM-DD.log
// and sets up a text handler for terminal output (only printing WARN/ERROR).
func Init(homeDir string) (*slog.Logger, error) {
	logDir := filepath.Join(homeDir, ".yap", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	dateStr := time.Now().Format("2006-01-02")
	logPath := filepath.Join(logDir, fmt.Sprintf("yap-%s.log", dateStr))

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	fileHandler := slog.NewJSONHandler(logFile, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	termHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	})

	handler := &DualHandler{
		fileHandler: fileHandler,
		termHandler: termHandler,
	}

	return slog.New(handler), nil
}
