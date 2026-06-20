# Yap AI Performance — MCP Server Kurulum & Optimizasyon Aracı

> Optimized MCP (Model Context Protocol) Server kurulum, yama ve yapılandırma aracı.  
> **Linux**, **Windows** ve **macOS** platformlarını destekler.

---

## 🚀 Özellikler

- **Çoklu Platform Desteği** — Linux (Arch/Ubuntu), Windows 10/11, macOS 12+ Monterey
- **Otomatik Bağımlılık Kurulumu** — `pipx`, `uv`, `python`, `git` otomatik kurulur
- **CodeGraphContext** — pipx üzerinden kurulum + kritik yamalar otomatik uygulanır
  - CGC_RUNTIME_DB_PATH veritabanı izolasyonu
  - MCP Protokol Versiyonu müzakeresi
  - stdout temizleme (ANSI kaçış kodları)
  - KuzuDB kilitlenme koruması (read-only fallback)
- **Graphify MCP** — `graphifyy[mcp]` uv tool olarak kurulur
- **MCP Client Entegrasyonu** — Gemini CLI, Claude Desktop, Cursor ve VS Code yapılandırmaları otomatik oluşturulur
- **Yap Skill** — Antigravity AI için yap skill global olarak kurulur

---

## ⚡ Hızlı Başlangıç

### 🐧 Linux (Arch / Ubuntu / Debian)

```bash
curl -fsSL https://raw.githubusercontent.com/frkntlr/yap-ai-performance/main/install.sh | bash
```

veya manuel:

```bash
git clone https://github.com/frkntlr/yap-ai-performance.git
cd yap-ai-performance
chmod +x install.sh
./install.sh
```

---

### 🍎 macOS (Monterey 12+)

```bash
curl -fsSL https://raw.githubusercontent.com/frkntlr/yap-ai-performance/main/install_mac.sh | bash
```

veya manuel:

```bash
git clone https://github.com/frkntlr/yap-ai-performance.git
cd yap-ai-performance
chmod +x install_mac.sh
./install_mac.sh
```

> **Not:** İlk çalıştırmada Homebrew ve Xcode Command Line Tools kurulabilir.  
> Apple Silicon (M1/M2/M3) ve Intel Mac'ler desteklenir.

---

### 🪟 Windows (PowerShell 5.1+)

PowerShell'i **Yönetici olarak** açın ve çalıştırın:

```powershell
# Önce execution policy'yi ayarlayın (tek seferlik)
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser

# Repoyu klonlayın ve scripti çalıştırın
git clone https://github.com/frkntlr/yap-ai-performance.git
cd yap-ai-performance
.\install.ps1
```

veya doğrudan indirip çalıştırmak için:

```powershell
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/frkntlr/yap-ai-performance/main/install.ps1" -OutFile "install.ps1"
powershell -ExecutionPolicy Bypass -File install.ps1
```

> **Not:** winget (Windows Package Manager) kurulu olması önerilir.  
> `winget` yoksa Python'u https://python.org adresinden manuel kurun.

---

## 🔧 Kurulum Adımları (Tüm Platformlar)

| Adım | İşlem | Linux | macOS | Windows |
|------|-------|:-----:|:-----:|:-------:|
| 1/7  | OS algılama | pacman/apt | sw_vers | WinVer |
| 2/7  | Bağımlılıklar | pacman/apt | Homebrew | winget/pip |
| 3/7  | CGC + Graphify kurulumu | ✓ | ✓ | ✓ |
| 4/7  | CGC yamaları | ✓ | ✓ | ✓ |
| 5/7  | Wrapper scriptler | `~/.local/bin/` | `~/.local/bin/` | `%USERPROFILE%\.local\bin\` |
| 6/7  | MCP config güncelleme | ✓ | `~/Library/...` | `%APPDATA%\...` |
| 7/7  | Doğrulama + yap skill | ✓ | ✓ | ✓ |

---

## 📁 Proje Yapısı

```
.
├── install.sh        # Linux kurulum scripti (Arch / Ubuntu / Debian)
├── install_mac.sh    # macOS kurulum scripti (Homebrew tabanlı)
├── install.ps1       # Windows kurulum scripti (PowerShell)
├── test.go           # Go JSON unmarshal edge-case testi
└── README.md
```

---

## 🔍 Desteklenen MCP Client'lar

| Client | Linux | macOS | Windows |
|--------|:-----:|:-----:|:-------:|
| Gemini CLI | ✓ | ✓ | ✓ |
| Claude Desktop | ✓ | ✓ | ✓ |
| Cursor (Cline) | ✓ | ✓ | ✓ |
| VS Code (Cline) | ✓ | ✓ | ✓ |

---

## 📋 Gereksinimler

| Platform | Gereksinim |
|----------|-----------|
| Linux | Bash 4.0+, sudo yetkisi |
| macOS | macOS 12+, Terminal, internet bağlantısı |
| Windows | PowerShell 5.1+, internet bağlantısı |

---

## 🛠 Geliştirme

Katkıda bulunmak için fork'layıp PR açabilirsiniz.  
Sorunlar için [Issues](https://github.com/frkntlr/yap-ai-performance/issues) bölümünü kullanın.

---

## 📄 Lisans

MIT License — © 2026 frkntlr
