package promptbuilder

import (
	"strings"
	"testing"

	"github.com/frkntlr/yap-ai-performance/internal/gitinfo"
	"github.com/frkntlr/yap-ai-performance/internal/scanner"
)

func TestBuildPrompt(t *testing.T) {
	ctx := PromptContext{
		Project: &scanner.ProjectInfo{
			Language:     "Go",
			Dependencies: []string{"github.com/spf13/cobra"},
			ModuleName:   "github.com/frkntlr/yap-ai-performance",
		},
		Git: &gitinfo.GitInfo{
			IsRepo:     true,
			Branch:     "main",
			DiffStat:   "cmd/yap/main.go | 2 +-",
			LastCommit: "Initial commit",
		},
	}

	prompt := Build(ctx)

	if !strings.Contains(prompt, "[SİSTEM BAĞLAMI]") {
		t.Errorf("expected prompt to contain '[SİSTEM BAĞLAMI]'")
	}
	if !strings.Contains(prompt, "Proje Dili: Go") {
		t.Errorf("expected prompt to contain 'Proje Dili: Go'")
	}
	if !strings.Contains(prompt, "[DEĞİŞEN DOSYALAR (git diff --stat - Branch: main)]") {
		t.Errorf("expected prompt to contain '[DEĞİŞEN DOSYALAR (git diff --stat - Branch: main)]'")
	}
	if !strings.Contains(prompt, "cmd/yap/main.go | 2 +-") {
		t.Errorf("expected prompt to contain diff stat")
	}
}
