package installer

import (
	"fmt"
	"strings"

	"github.com/frkntlr/yap-ai-performance/internal/context"
	"github.com/frkntlr/yap-ai-performance/internal/detector"
	"github.com/frkntlr/yap-ai-performance/internal/logger"
	"github.com/frkntlr/yap-ai-performance/internal/backup"
)

type InstallOptions struct {
	Only   string // "deps", "tools", "patch", "config", or empty for all
	DryRun bool
}

// Run orchestrates the 6-step installer workflow.
func Run(p *detector.Platform, opts InstallOptions) error {
	// Initialize logger
	logInst, err := logger.Init(p.HomeDir)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	backupMgr := backup.NewManager(p.HomeDir)

	ctx := &context.RunContext{
		DryRun: opts.DryRun,
		Logger: logInst,
		Backup: backupMgr,
	}

	ctx.Logger.Info("Installation process started", "dryRun", opts.DryRun, "only", opts.Only)

	steps := []struct {
		name string
		id   string
		run  func(*detector.Platform, *context.RunContext) error
	}{
		{"Checking and installing dependencies", "deps", Step2Deps},
		{"Installing CodeGraphContext and Graphify", "tools", Step3Tools},
		{"Applying patches to CodeGraphContext", "patch", Step4Patch},
		{"Updating MCP configuration files", "config", Step5Config},
		{"Verifying setup", "verify", Step6Verify},
	}

	filter := strings.ToLower(strings.TrimSpace(opts.Only))

	fmt.Printf("\n=== Starting Installation on %s (%s) ===\n", p.OS, p.PackageMgr)
	if opts.DryRun {
		fmt.Println("--- DRY-RUN MODE ACTIVE: Changes will only be simulated ---")
	}

	for i, step := range steps {
		// If "only" filter is active, skip other steps (except verify)
		if filter != "" && step.id != filter && step.id != "verify" {
			continue
		}

		fmt.Printf("\n[%d/5] %s...\n", i+1, step.name)
		ctx.Logger.Info("Running step", "stepId", step.id, "stepName", step.name)

		if err := step.run(p, ctx); err != nil {
			fmt.Printf("✗ Step failed: %v\n", err)
			ctx.Logger.Error("Step failed", "stepId", step.id, "error", err)
			return err
		}
		fmt.Printf("✓ Step completed successfully\n")
		ctx.Logger.Info("Step completed", "stepId", step.id)
	}

	fmt.Println("\n=== Installation Process Finished! ===")
	ctx.Logger.Info("Installation process completed successfully")
	return nil
}

