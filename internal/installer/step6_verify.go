package installer

import (
	"github.com/frkntlr/yap-ai-performance/internal/context"
	"github.com/frkntlr/yap-ai-performance/internal/detector"
	"github.com/frkntlr/yap-ai-performance/internal/status"
)

// Step6Verify runs the diagnostics suite to verify the installation succeeded.
func Step6Verify(p *detector.Platform, ctx *context.RunContext) error {
	results := status.RunStatus(p, ctx.Logger)
	status.PrintStatus(results)

	// Log all check results to the daily file logger
	for _, res := range results {
		if res.OK {
			ctx.Logger.Info("Diagnostic check passed", "name", res.Name, "detail", res.Detail)
		} else {
			ctx.Logger.Warn("Diagnostic check failed", "name", res.Name, "detail", res.Detail, "fix", res.Fix)
		}
	}

	return nil
}

