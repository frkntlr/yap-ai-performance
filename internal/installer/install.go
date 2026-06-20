package installer

import (
	"fmt"
	"strings"

	"github.com/frkntlr/yap-ai-performance/internal/detector"
)

type InstallOptions struct {
	Only string // "deps", "tools", "patch", "config", or empty for all
}

// Run orchestrates the 6-step installer workflow.
func Run(p *detector.Platform, opts InstallOptions) error {
	steps := []struct {
		name string
		id   string
		run  func(*detector.Platform) error
	}{
		{"Checking and installing dependencies", "deps", Step2Deps},
		{"Installing CodeGraphContext and Graphify", "tools", Step3Tools},
		{"Applying patches to CodeGraphContext", "patch", Step4Patch},
		{"Updating MCP configuration files", "config", Step5Config},
		{"Verifying setup", "verify", Step6Verify},
	}

	filter := strings.ToLower(strings.TrimSpace(opts.Only))

	fmt.Printf("\n=== Starting Installation on %s (%s) ===\n", p.OS, p.PackageMgr)

	for i, step := range steps {
		// If "only" filter is active, skip other steps (except verify)
		if filter != "" && step.id != filter && step.id != "verify" {
			continue
		}

		fmt.Printf("\n[%d/5] %s...\n", i+1, step.name)
		if err := step.run(p); err != nil {
			fmt.Printf("✗ Step failed: %v\n", err)
			return err
		}
		fmt.Printf("✓ Step completed successfully\n")
	}

	fmt.Println("\n=== Installation Process Finished! ===")
	return nil
}
