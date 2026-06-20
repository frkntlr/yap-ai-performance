package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/frkntlr/yap-ai-performance/internal/backup"
	"github.com/frkntlr/yap-ai-performance/internal/confirm"
	"github.com/frkntlr/yap-ai-performance/internal/detector"
	"github.com/frkntlr/yap-ai-performance/internal/installer"
	"github.com/frkntlr/yap-ai-performance/internal/logger"
	"github.com/frkntlr/yap-ai-performance/internal/proxy"
	"github.com/frkntlr/yap-ai-performance/internal/status"
	"github.com/spf13/cobra"
)

var (
	onlyFlag   string
	dryRunFlag bool
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "yap",
		Short: "Yap AI Performance CLI is a robust cross-platform management tool for MCP servers",
		Long:  `A fast and reliable command line interface written in Go to install, configure, update and proxy MCP servers (CodeGraphContext & Graphify) across Windows, Linux and macOS.`,
	}

	// Persistent flags (available to all commands)
	rootCmd.PersistentFlags().BoolVar(&dryRunFlag, "dry-run", false, "Simulate actions without making changes")

	var installCmd = &cobra.Command{
		Use:   "install",
		Short: "Install and configure dependencies, tools and MCP settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := detector.Detect()
			if err != nil {
				return err
			}
			opts := installer.InstallOptions{
				Only:   onlyFlag,
				DryRun: dryRunFlag,
			}
			return installer.Run(p, opts)
		},
	}
	installCmd.Flags().StringVar(&onlyFlag, "only", "", "Specify a single component to install (deps, tools, patch, config)")

	var updateCmd = &cobra.Command{
		Use:   "update",
		Short: "Pull updates and redeploy services",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("update command stub. Features will be implemented in Phase 3.")
		},
	}

	var statusCmd = &cobra.Command{
		Use:   "status",
		Short: "Perform active health check diagnostics on all installed components",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := detector.Detect()
			if err != nil {
				return err
			}
			// Initialize logger to enable diagnostic check logging
			logInst, _ := logger.Init(p.HomeDir)
			results := status.RunStatus(p, logInst)
			status.PrintStatus(results)
			return nil
		},
	}

	var proxyCmd = &cobra.Command{
		Use:   "proxy [service]",
		Short: "Run the native Go MCP proxy for a service",
		Long:  `Run a transparent, workspace-aware JSON-RPC proxy for codegraphcontext (cgc) or graphify.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			service := args[0]
			switch service {
			case "cgc":
				return proxy.RunCGCProxy()
			case "graphify":
				return proxy.RunGraphifyProxy()
			default:
				return fmt.Errorf("unknown proxy service: %s. Supported: cgc, graphify", service)
			}
		},
	}

	var rollbackCmd = &cobra.Command{
		Use:   "rollback",
		Short: "Restore configuration files to their last backup state",
		RunE:  runRollback,
	}

	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(proxyCmd)
	rootCmd.AddCommand(rollbackCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runRollback(cmd *cobra.Command, args []string) error {
	p, err := detector.Detect()
	if err != nil {
		return err
	}

	backupMgr := backup.NewManager(p.HomeDir)
	latestBackups, err := backupMgr.ListAllLatestBackups()
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	if len(latestBackups) == 0 {
		fmt.Println("Geri yüklenecek herhangi bir yedek bulunamadı (No backups found).")
		return nil
	}

	fmt.Println("Geri yüklenecek dosyalar (Files to restore):")
	for _, entry := range latestBackups {
		prettyPath := entry.OriginalPath
		if strings.HasPrefix(prettyPath, p.HomeDir) {
			prettyPath = "~" + strings.TrimPrefix(prettyPath, p.HomeDir)
		}
		fmt.Printf("  %s  (Yedek: %s)\n", prettyPath, entry.Timestamp.Format("2006-01-02 15:04:05"))
	}

	fmt.Println()
	approved, err := confirm.AskYesNo("Devam etmek istiyor musunuz? (Do you want to proceed?)")
	if err != nil {
		return err
	}

	if !approved {
		fmt.Println("Geri yükleme işlemi iptal edildi (Rollback cancelled).")
		return nil
	}

	for _, entry := range latestBackups {
		err := backupMgr.RestoreLatest(entry.OriginalPath)
		if err != nil {
			fmt.Printf("✗ %s geri yüklenemedi: %v\n", filepath.Base(entry.OriginalPath), err)
		} else {
			fmt.Printf("✓ %s geri yüklendi.\n", filepath.Base(entry.OriginalPath))
		}
	}

	return nil
}
