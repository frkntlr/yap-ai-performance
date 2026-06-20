package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/frkntlr/yap-ai-performance/internal/backup"
	"github.com/frkntlr/yap-ai-performance/internal/config"
	"github.com/frkntlr/yap-ai-performance/internal/confirm"
	"github.com/frkntlr/yap-ai-performance/internal/detector"
	"github.com/frkntlr/yap-ai-performance/internal/env"
	"github.com/frkntlr/yap-ai-performance/internal/gitinfo"
	"github.com/frkntlr/yap-ai-performance/internal/installer"
	"github.com/frkntlr/yap-ai-performance/internal/logger"
	"github.com/frkntlr/yap-ai-performance/internal/proxy"
	"github.com/frkntlr/yap-ai-performance/internal/scanner"
	"github.com/frkntlr/yap-ai-performance/internal/status"
	"github.com/frkntlr/yap-ai-performance/pkg/promptbuilder"
	"github.com/spf13/cobra"
)

var (
	onlyFlag     string
	dryRunFlag   bool
	promptFlag   bool
	jsonFlag     bool
	withDiffFlag bool
	saveFlag     bool
	outFlag      string
)


func main() {
	cwd, err := os.Getwd()
	if err == nil {
		_ = env.Load(cwd)
	}

	p, err := detector.Detect()
	var home string
	if p != nil {
		home = p.HomeDir
	}
	cfg, _ := config.Load(home, cwd)
	if cfg == nil {
		cfg = config.NewDefault()
	}

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

	var contextCmd = &cobra.Command{
		Use:   "context",
		Short: "Bulunulan dizinin proje ve git bağlamını analiz eder",
		Long:  `Proje teknolojilerini, bağımlılıklarını ve git durumunu analiz ederek AI dostu bağlam üretir.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current working directory: %w", err)
			}

			projInfo, err := scanner.Scan(cwd)
			if err != nil {
				return fmt.Errorf("failed to scan project: %w", err)
			}

			gitInfo, err := gitinfo.Read(cwd, withDiffFlag)
			if err != nil {
				return fmt.Errorf("failed to read git status: %w", err)
			}

			if jsonFlag {
				type Combined struct {
					Project *scanner.ProjectInfo `json:"project"`
					Git     *gitinfo.GitInfo     `json:"git"`
				}
				combined := Combined{Project: projInfo, Git: gitInfo}
				data, err := json.MarshalIndent(combined, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal JSON: %w", err)
				}
				fmt.Println(string(data))
				return nil
			}

			promptCtx := promptbuilder.PromptContext{
				Project: projInfo,
				Git:     gitInfo,
			}
			promptStr := promptbuilder.Build(promptCtx)

			if promptFlag {
				if saveFlag || outFlag != "" {
					savePath := outFlag
					if savePath == "" {
						p, err := detector.Detect()
						if err != nil {
							return err
						}
						yapDir := filepath.Join(p.HomeDir, ".yap")
						if err := os.MkdirAll(yapDir, 0755); err != nil {
							return fmt.Errorf("failed to create directory %s: %w", yapDir, err)
						}
						savePath = filepath.Join(yapDir, "context.md")
					}
					if err := os.WriteFile(savePath, []byte(promptStr), 0644); err != nil {
						return fmt.Errorf("failed to write file %s: %w", savePath, err)
					}
					fmt.Printf("✓ Bağlam promptu başarıyla kaydedildi: %s\n", savePath)
					return nil
				}

				fmt.Print(promptStr)
				return nil
			}

			fmt.Println("╔══════════════════════════════════════════╗")
			fmt.Println("║   Yap AI — Project Context Analysis     ║")
			fmt.Println("╚══════════════════════════════════════════╝")
			fmt.Printf("📁 Dizin    : %s\n", cwd)
			fmt.Printf("🔤 Dil      : %s\n", projInfo.Language)
			if projInfo.ModuleName != "" {
				fmt.Printf("🗂  Modül   : %s\n", projInfo.ModuleName)
			}
			if len(projInfo.Frameworks) > 0 {
				fmt.Printf("📦 Araçlar  : %s\n", strings.Join(projInfo.Frameworks, ", "))
			}
			if len(projInfo.Dependencies) > 0 {
				depsToShow := projInfo.Dependencies
				if len(depsToShow) > 5 {
					depsToShow = append(depsToShow[:5], "...")
				}
				fmt.Printf("🔌 Bağımlı. : %s\n", strings.Join(depsToShow, ", "))
			}

			if gitInfo.IsRepo {
				fmt.Println()
				fmt.Printf("🌿 Branch   : %s\n", gitInfo.Branch)
				changeCount := len(gitInfo.ModifiedFiles) + len(gitInfo.UntrackedFiles) + len(gitInfo.StagedFiles)
				fmt.Printf("📝 Değişim  : %d dosya (Mod: %d, Yeni: %d, Staged: %d)\n",
					changeCount, len(gitInfo.ModifiedFiles), len(gitInfo.UntrackedFiles), len(gitInfo.StagedFiles))
				if gitInfo.LastCommit != "" {
					fmt.Printf("💬 Son Com. : \"%s\"\n", gitInfo.LastCommit)
				}
			}

			fmt.Println("\n[Sistem Promptu hazır — 'yap context --prompt' ile görüntüleyin]")
			return nil
		},
	}

	contextCmd.Flags().BoolVar(&promptFlag, "prompt", false, "AI modeline gönderilecek sistem promptunu üretir")
	contextCmd.Flags().BoolVar(&jsonFlag, "json", false, "Makine okunabilir ham JSON çıktısı verir")
	contextCmd.Flags().BoolVar(&withDiffFlag, "with-diff", false, "Git değişiklik detayını (diff) prompta ekler")
	contextCmd.Flags().BoolVar(&saveFlag, "save", false, "Prompt çıktısını varsayılan konuma (~/.yap/context.md) kaydeder")
	contextCmd.Flags().StringVar(&outFlag, "out", "", "Prompt çıktısını belirtilen dosya yoluna kaydeder")

	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(proxyCmd)
	rootCmd.AddCommand(rollbackCmd)
	rootCmd.AddCommand(contextCmd)

	// Dynamically register alias commands
	for name, promptTemplate := range cfg.Aliases {
		aliasName := name
		aliasPrompt := promptTemplate

		var aliasCmd = &cobra.Command{
			Use:   aliasName,
			Short: fmt.Sprintf("Özel kısayol komutu: %s", aliasName),
			RunE: func(cmd *cobra.Command, args []string) error {
				projInfo, err := scanner.Scan(cwd)
				if err != nil {
					return fmt.Errorf("failed to scan project: %w", err)
				}

				gitInfo, err := gitinfo.Read(cwd, withDiffFlag)
				if err != nil {
					return fmt.Errorf("failed to read git status: %w", err)
				}

				promptCtx := promptbuilder.PromptContext{
					Project: projInfo,
					Git:     gitInfo,
				}
				contextPrompt := promptbuilder.Build(promptCtx)
				fullPrompt := contextPrompt + aliasPrompt + "\n"

				if saveFlag || outFlag != "" {
					savePath := outFlag
					if savePath == "" {
						yapDir := filepath.Join(home, ".yap")
						if err := os.MkdirAll(yapDir, 0755); err != nil {
							return fmt.Errorf("failed to create directory %s: %w", yapDir, err)
						}
						savePath = filepath.Join(yapDir, "context.md")
					}
					if err := os.WriteFile(savePath, []byte(fullPrompt), 0644); err != nil {
						return fmt.Errorf("failed to write file %s: %w", savePath, err)
					}
					fmt.Printf("✓ Kısayol promptu başarıyla kaydedildi: %s\n", savePath)
					return nil
				}

				fmt.Print(fullPrompt)
				return nil
			},
		}

		aliasCmd.Flags().BoolVar(&withDiffFlag, "with-diff", false, "Git değişiklik detayını (diff) prompta ekler")
		aliasCmd.Flags().BoolVar(&saveFlag, "save", false, "Prompt çıktısını varsayılan konuma (~/.yap/context.md) kaydeder")
		aliasCmd.Flags().StringVar(&outFlag, "out", "", "Prompt çıktısını belirtilen dosya yoluna kaydeder")

		rootCmd.AddCommand(aliasCmd)
	}

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
