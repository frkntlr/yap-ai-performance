# Yap AI Performance

Linux, macOS ve Windows için Go ile yazılmış MCP (Model Context Protocol) yönetim, kurulum ve bağlam CLI aracı.

`yap install` tek komutla CodeGraphContext + Graphify araçlarını kurar, gerekli yamaları uygular, `yap` proxy binary’sini yerleştirir ve **Cursor, Zed, Antigravity/Gemini, Claude Code** (ve varsa Claude Desktop / Cline) için MCP + `/yap` skill yapılandırmasını yazar.

---

## Nedir?

- **Tek binary:** Harici runtime gerektirmez (CGC/Graphify Python araçları pipx/uv ile kurulur).
- **Go-native proxy:** `yap proxy cgc` ve `yap proxy graphify` — IDE’lerin konuştuğu MCP uçları.
- **`yap context`:** Proje tarama + git diff ile AI’ya hazır sistem promptu.
- **`yap status`:** Araçlar, yamalar, MCP config’ler, skill’ler ve proxy canlılık kontrolü.
- **Güvenlik:** `--dry-run`, otomatik yedekleme, `yap rollback`.

---

## Hızlı başlangıç

### Linux

```bash
bash install.sh
# veya
go build -o dist/yap-linux-amd64 ./cmd/yap
./dist/yap-linux-amd64 install
yap status
```

### macOS

```bash
bash install_mac.sh
# veya derlenmiş binary ile: yap install
```

### Windows

```powershell
# dist\yap-windows-amd64.exe hazır olsun veya Go ile derleyin
powershell -ExecutionPolicy Bypass -File install.ps1
yap status
```

Binary konumu: `%LOCALAPPDATA%\Programs\yap\yap.exe` (PATH’e eklenir).

Detay: [docs/install.md](docs/install.md)

---

## Desteklenen IDE’ler

| IDE | MCP | /yap skill | Not |
|-----|-----|------------|-----|
| **Cursor** | `~/.cursor/mcp.json` + proje `.cursor/mcp.json` | skill + always-apply rule | Agent için zorunlu global MCP |
| **Zed** | `context_servers` (global + proje `.zed/settings.json`) | — | Yoksa settings oluşturulur |
| **Antigravity / Gemini** | `~/.gemini/config/mcp_config.json` + proje `.agents/mcp_config.json` | `~/.gemini/config/skills` + `.agents/skills` | Workspace + global |
| **Claude Code** | `~/.claude.json` (`mcpServers` merge) + proje `.mcp.json` | `~/.claude/skills` + `.claude/skills` | State dosyası silinmez |
| **Claude Desktop** | `claude_desktop_config.json` | — | Yalnızca dosya varsa güncellenir |
| **Cline** | mevcut settings | — | Yalnızca dosya varsa |

Kurulumdan sonra ilgili IDE’yi **yeniden başlatın** (veya MCP listesini yenileyin). Ayrıntılar: [docs/ides.md](docs/ides.md)

---

## Komutlar

```bash
yap install                 # tam kurulum
yap install --dry-run       # simülasyon
yap install --only=config   # yalnızca MCP/skill/binary
yap status                  # sağlık kontrolü
yap context --prompt        # AI sistem promptu
yap rollback                # son config yedeğine dön
```

Tam referans: [docs/commands.md](docs/commands.md)

---

## `/yap` nedir?

Cursor / Claude Code / Agents skill’i olarak yüklenen orkestrasyon akışı:

1. `yap status` / `yap context` ile bootstrap  
2. Graphify + CodeGraphContext MCP araçlarıyla mimari, caller/callee, karmaşıklık analizi  
3. Kısa bulgu → kanıt → aksiyon  

Skill dosyası `yap install --only=config` ile global ve proje dizinlerine yazılır.

---

## Dokümantasyon

| Belge | İçerik |
|-------|--------|
| [docs/README.md](docs/README.md) | Doküman indeksi |
| [docs/install.md](docs/install.md) | Kurulum, dry-run, `--only` |
| [docs/ides.md](docs/ides.md) | IDE path’leri ve doğrulama |
| [docs/commands.md](docs/commands.md) | CLI referansı |
| [docs/architecture.md](docs/architecture.md) | Proxy + install pipeline |

---

## Yapılandırma (`.yaprc`)

Global `~/.yaprc` veya proje `.yaprc`:

```json
{
  "default_model": "gemini-1.5-flash",
  "log_level": "INFO",
  "aliases": {
    "code-review": "Değişikliklerin Big O ve bellek etkisini analiz et.",
    "explain": "Bu kodu adım adım açıkla."
  }
}
```

```bash
yap code-review --with-diff
```

---

## Dizin yapısı

```
cmd/yap/            CLI giriş
internal/installer/ Kurulum adımları (MCP, skills, patch)
internal/proxy/     CGC / Graphify proxy
internal/status/    yap status
internal/detector/  OS + path adayları
docs/               Kullanıcı dokümantasyonu
```

Mimari özeti: [docs/architecture.md](docs/architecture.md)

---

## Lisans

MIT License — © 2026 frkntlr
