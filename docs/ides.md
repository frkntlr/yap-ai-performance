# IDE yapılandırması

`yap install` (veya `--only=config`) aşağıdaki hedefleri yazar. MCP sunucuları her yerde aynıdır:

```json
{
  "mcpServers": {
    "CodeGraphContext": { "command": "<yap-path>", "args": ["proxy", "cgc"] },
    "Graphify":         { "command": "<yap-path>", "args": ["proxy", "graphify"] }
  }
}
```

`<yap-path>` örneği: Linux/macOS `~/.local/bin/yap`, Windows `%LOCALAPPDATA%\Programs\yap\yap.exe`.

Kurulumdan sonra IDE’yi yeniden başlatın veya MCP panelinden yenileyin. Proje scope dosyaları (`./.mcp.json` vb.) ilk kullanımda onay isteyebilir.

---

## Cursor

| Tür | Path |
|-----|------|
| Global MCP | `~/.cursor/mcp.json` |
| Proje MCP | `<repo>/.cursor/mcp.json` |
| Global skill | `~/.cursor/skills/yap/SKILL.md` |
| Global rule | `~/.cursor/rules/yap-context.mdc` (`alwaysApply`) |
| Proje skill/rule | `<repo>/.cursor/skills/...`, `.cursor/rules/...` |

Doğrulama: `yap status` → `Config: Cursor MCP`, `Skill: Cursor /yap`.

---

## Zed

| Tür | Path |
|-----|------|
| Global | Linux/macOS `~/.config/zed/settings.json` · Windows `%APPDATA%\Zed\settings.json` |
| Proje | `<repo>/.zed/settings.json` |

`context_servers` altına `yap-cgc` ve `yap-graphify` yazılır. Dosya yoksa oluşturulur.

Doğrulama: `Config: Zed context_servers`.

---

## Antigravity / Gemini

| Tür | Path |
|-----|------|
| Global MCP | `~/.gemini/config/mcp_config.json` |
| Workspace MCP | `<repo>/.agents/mcp_config.json` |
| Global skill | `~/.gemini/config/skills/yap/SKILL.md` |
| Workspace skill | `<repo>/.agents/skills/yap/SKILL.md` |
| Agents skill (home) | `~/.agents/skills/yap/SKILL.md` |

Antigravity IDE: Agent paneli → MCP Servers → gerekirse Refresh.

Doğrulama: `Config: Gemini/Antigravity MCP`, `Skill: Gemini /yap`.

---

## Claude Code

| Tür | Path | Not |
|-----|------|-----|
| User MCP | `~/.claude.json` | Yalnızca `mcpServers` **merge** edilir; oauth/cache silinmez |
| Proje MCP | `<repo>/.mcp.json` | Team/git için; onay gerekebilir |
| Personal skill | `~/.claude/skills/yap/SKILL.md` | |
| Proje skill | `<repo>/.claude/skills/yap/SKILL.md` | |

**Yanlış yer:** `~/.claude/settings.json` MCP yüklemez — kullanmayın.

Doğrulama: `Config: Claude Code user MCP`, `Config: project .mcp.json`, `Skill: Claude Code /yap`.

---

## Claude Desktop / Cline

Dosya **zaten varsa** güncellenir; yoksa oluşturulmaz.

- Linux: `~/.config/Claude/claude_desktop_config.json`
- macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
- Windows: `%APPDATA%\Claude\claude_desktop_config.json`
- Cline: Cursor/VS Code `globalStorage/.../cline_mcp_settings.json`

---

## graphify platform hook’ları

Config adımında (graphify PATH’teyse):

```text
graphify install --platform cursor|gemini|agents|antigravity|claude
```

Desteklenmeyen platform uyarısı güvenli şekilde yok sayılır.

---

## Sorun giderme

1. `yap status` çıktısındaki kırmızı satırın `Fix` ipucunu uygulayın.  
2. IDE’yi tamamen kapatıp açın.  
3. Proxy log: `~/.yap/logs/yap-YYYY-MM-DD.log`  
4. Claude Code proje MCP onayı: oturumda `/mcp` veya trust dialog.
