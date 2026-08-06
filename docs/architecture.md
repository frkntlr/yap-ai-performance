# Mimari (kısa)

## Install pipeline

```text
Detect(OS) → deps → tools (pipx/uv) → patch CGC → deploy yap binary
  → MCP targets (Cursor, Gemini, Claude Code merge, project scopes)
  → Zed context_servers
  → skills/rules
  → graphify --platform …
  → verify / status
```

Kaynak: `internal/installer/` (`install.go`, `step2`…`step6`, `step5_config.go`, `step5_ide.go`, `mcp_targets.go`).

## MCP proxy

IDE → `yap proxy cgc|graphify` → CodeGraphContext / Graphifyy subprocess.

- CGC: workspace `.codegraphcontext_db`, Windows/Linux binary path adayları (`internal/detector/paths.go`)
- Graphify: `graphify-out/graph.json` bulma + uv tool python

Kaynak: `internal/proxy/cgc.go`, `internal/proxy/graphify.go`.

## Claude Code güvenliği

`~/.claude.json` çok alanlı state dosyasıdır. `pkg/jsonutil.MergeMCPServers` yalnızca `mcpServers` anahtarını birleştirir; diğer alanlar korunur.

## Path çözümleme

Windows/Linux pipx, uv ve Scripts konumları `internal/detector/paths.go` içinde aday listesi olarak tutulur; ilk mevcut path seçilir.

## Bağlam

`yap context` → scanner + gitinfo → `pkg/promptbuilder`.
