package installer

import (
	"github.com/frkntlr/yap-ai-performance/internal/detector"
	"github.com/frkntlr/yap-ai-performance/internal/status"
)

// Step6Verify runs the diagnostics suite to verify the installation succeeded.
func Step6Verify(p *detector.Platform) error {
	results := status.RunStatus(p)
	status.PrintStatus(results)
	return nil
}
