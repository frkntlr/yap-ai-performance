package installer

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/frkntlr/yap-ai-performance/internal/context"
	"github.com/frkntlr/yap-ai-performance/internal/detector"
	"github.com/frkntlr/yap-ai-performance/internal/dryrun"
	"github.com/frkntlr/yap-ai-performance/pkg/runner"
)

// Step2Deps checks and installs system-level dependencies.
func Step2Deps(p *detector.Platform, ctx *context.RunContext) error {
	// Check Git
	if !runner.Exists("git") {
		fmt.Println("git not found. Installing...")
		if err := installDependency(p, ctx, "git"); err != nil {
			return err
		}
	} else {
		fmt.Println("✓ git is already installed.")
	}

	// Check Python3
	pythonCmd := "python3"
	if p.OS == "windows" {
		pythonCmd = "python"
	}
	if !runner.Exists(pythonCmd) {
		fmt.Println("python not found. Installing...")
		if err := installDependency(p, ctx, "python"); err != nil {
			return err
		}
	} else {
		fmt.Println("✓ python is already installed.")
	}

	// Check pipx
	if !runner.Exists("pipx") {
		fmt.Println("pipx not found. Installing...")
		if err := installDependency(p, ctx, "pipx"); err != nil {
			return err
		}
	} else {
		fmt.Println("✓ pipx is already installed.")
	}

	// Ensure pipx path
	fmt.Println("Ensuring pipx paths are configured...")
	_ = runner.Run(ctx.DryRun, "pipx", "ensurepath", "--force")

	// Check uv
	if !runner.Exists("uv") {
		fmt.Println("uv not found. Installing...")
		if err := installDependency(p, ctx, "uv"); err != nil {
			return err
		}
	} else {
		fmt.Println("✓ uv is already installed.")
	}

	return nil
}

func installDependency(p *detector.Platform, ctx *context.RunContext, dep string) error {
	switch p.OS {
	case "windows":
		return installWinDep(ctx, dep)
	case "darwin":
		return installMacDep(ctx, dep)
	case "linux":
		return installLinuxDep(p, ctx, dep)
	}
	return fmt.Errorf("unsupported OS for auto-installation: %s", p.OS)
}

func installWinDep(ctx *context.RunContext, dep string) error {
	switch dep {
	case "git":
		return runner.Run(ctx.DryRun, "winget", "install", "--id", "Git.Git", "--silent", "--accept-package-agreements", "--accept-source-agreements")
	case "python":
		return runner.Run(ctx.DryRun, "winget", "install", "--id", "Python.Python.3.12", "--silent", "--accept-package-agreements", "--accept-source-agreements")
	case "pipx":
		// Install via python pip
		return runner.Run(ctx.DryRun, "python", "-m", "pip", "install", "--user", "pipx")
	case "uv":
		// Install via powershell installer script
		return runner.Run(ctx.DryRun, "powershell", "-ExecutionPolicy", "Bypass", "-Command", "irm https://astral.sh/uv/install.ps1 | iex")
	}
	return fmt.Errorf("unknown dependency: %s", dep)
}

func installMacDep(ctx *context.RunContext, dep string) error {
	if !runner.Exists("brew") {
		return fmt.Errorf("Homebrew (brew) is required on macOS but not found. Please install Homebrew first")
	}
	return runner.Run(ctx.DryRun, "brew", "install", dep)
}

func installLinuxDep(p *detector.Platform, ctx *context.RunContext, dep string) error {
	// Specialized uv installation via curl script
	if dep == "uv" {
		fmt.Println("Installing uv via official standalone script...")
		if ctx.DryRun {
			dryrun.PrintSimulation("'sh -c curl -LsSf https://astral.sh/uv/install.sh | sh' komutu çalıştırılacak")
		} else {
			cmd := exec.Command("sh", "-c", "curl -LsSf https://astral.sh/uv/install.sh | sh")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return err
			}
			// Source cargo env for current process path
			home, _ := os.UserHomeDir()
			os.Setenv("PATH", fmt.Sprintf("%s/.local/bin:%s/.cargo/bin:%s", home, home, os.Getenv("PATH")))
		}
		return nil
	}

	switch p.PackageMgr {
	case "pacman":
		var pkgName string
		switch dep {
		case "git":
			pkgName = "git"
		case "python":
			pkgName = "python"
		case "pipx":
			pkgName = "python-pipx"
		}
		fmt.Printf("Running: sudo pacman -S --needed --noconfirm %s\n", pkgName)
		err := runner.Run(ctx.DryRun, "sudo", "pacman", "-S", "--needed", "--noconfirm", pkgName)
		if err != nil {
			return fmt.Errorf("failed to install %s. Please run 'sudo pacman -S --needed %s' manually: %v", dep, pkgName, err)
		}
		return nil

	case "apt":
		var pkgName string
		switch dep {
		case "git":
			pkgName = "git"
		case "python":
			pkgName = "python3 python3-pip"
		case "pipx":
			pkgName = "pipx"
		}
		fmt.Printf("Running: sudo apt-get update && sudo apt-get install -y %s\n", pkgName)
		_ = runner.Run(ctx.DryRun, "sudo", "apt-get", "update")
		err := runner.Run(ctx.DryRun, "sudo", "apt-get", "install", "-y", pkgName)
		if err != nil {
			return fmt.Errorf("failed to install %s. Please run 'sudo apt-get install -y %s' manually: %v", dep, pkgName, err)
		}
		return nil

	default:
		return fmt.Errorf("unknown package manager. Please install %s manually", dep)
	}
}

