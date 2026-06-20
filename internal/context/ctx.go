package context

import (
	"log/slog"

	"github.com/frkntlr/yap-ai-performance/internal/backup"
)

// RunContext holds the execution context for installation steps,
// enabling dry-run, structured logging, and configuration backup.
type RunContext struct {
	DryRun bool
	Logger *slog.Logger
	Backup *backup.Manager
}
