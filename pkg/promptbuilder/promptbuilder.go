package promptbuilder

import (
	"fmt"
	"strings"

	"github.com/frkntlr/yap-ai-performance/internal/gitinfo"
	"github.com/frkntlr/yap-ai-performance/internal/scanner"
)

// PromptContext holds the context for generating the dynamic system prompt.
type PromptContext struct {
	Project *scanner.ProjectInfo
	Git     *gitinfo.GitInfo
}

// Build generates a formatted AI context prompt based on the scanner and git information.
func Build(ctx PromptContext) string {
	var sb strings.Builder

	sb.WriteString("[SİSTEM BAĞLAMI]\n")
	sb.WriteString(fmt.Sprintf("Proje Dili: %s\n", ctx.Project.Language))

	if len(ctx.Project.Dependencies) > 0 {
		sb.WriteString(fmt.Sprintf("Ana Bağımlılıklar: %s\n", strings.Join(ctx.Project.Dependencies, ", ")))
	}

	if len(ctx.Project.Frameworks) > 0 {
		sb.WriteString(fmt.Sprintf("Çatı/Araçlar: %s\n", strings.Join(ctx.Project.Frameworks, ", ")))
	}

	if ctx.Project.ModuleName != "" {
		sb.WriteString(fmt.Sprintf("Modül Adı: %s\n", ctx.Project.ModuleName))
	}

	if ctx.Project.HasDockerfile {
		sb.WriteString("Dockerfile: Mevcut\n")
	}
	sb.WriteString("\n")

	if ctx.Git.IsRepo {
		sb.WriteString(fmt.Sprintf("[DEĞİŞEN DOSYALAR (git diff --stat - Branch: %s)]\n", ctx.Git.Branch))
		if ctx.Git.DiffStat != "" {
			sb.WriteString(ctx.Git.DiffStat)
			sb.WriteString("\n")
		} else {
			sb.WriteString("Değişiklik yok.\n")
		}
		sb.WriteString("\n")

		if ctx.Git.FullDiff != "" {
			sb.WriteString("[DETAYLI GİT DİFF]\n")
			sb.WriteString(ctx.Git.FullDiff)
			sb.WriteString("\n\n")
		}
	}

	sb.WriteString("[KULLANICI TALEBİ]\n")

	return sb.String()
}
