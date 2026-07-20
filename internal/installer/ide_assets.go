package installer

import "embed"

//go:embed assets/cursor_yap_skill.md assets/cursor_yap_rule.mdc assets/gemini_yap_skill.md
var ideAssets embed.FS

func assetBytes(name string) ([]byte, error) {
	return ideAssets.ReadFile("assets/" + name)
}
