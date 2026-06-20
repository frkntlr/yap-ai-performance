# Yap AI Performance — MCP Server Kurulum & Optimizasyon Aracı

> Optimized MCP (Model Context Protocol) Server kurulum, yama ve yapılandırma scripti.  
> Arch/CachyOS ve Ubuntu/Debian tabanlı sistemleri destekler.

---

## 🚀 Özellikler

- **Otomatik OS Algılama** — Arch (pacman) ve Debian (apt) tabanlı dağıtımları otomatik tanır
- **Bağımlılık Yönetimi** — `pipx`, `uv`, `python`, `git` otomatik kurulur
- **CodeGraphContext** — pipx üzerinden kurulum + kritik yamalar otomatik uygulanır
  - CGC_RUNTIME_DB_PATH veritabanı izolasyonu
  - MCP Protokol Versiyonu müzakeresi
  - stdout temizleme (ANSI kaçış kodları)
- **Graphify MCP** — `graphifyy[mcp]` uv tool olarak kurulur
- **Gemini CLI Entegrasyonu** — MCP sunucu konfigürasyonları otomatik oluşturulur
- **systemd Servisi** — Arka planda kalıcı çalışma desteği

---

## 📋 Gereksinimler

- Linux (Arch/CachyOS veya Ubuntu/Debian)
- `bash` 4.0+
- İnternet bağlantısı
- `sudo` yetkisi (bağımlılık kurulumu için)

---

## ⚡ Hızlı Başlangıç

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

## 🔧 Ne Yapar?

| Adım | İşlem |
|------|-------|
| 1/7  | OS ve paket yöneticisi algılama |
| 2/7  | pipx, uv, python, git kurulumu |
| 3/7  | CodeGraphContext ve Graphify kurulumu |
| 4/7  | CodeGraphContext yamaları uygulanır |
| 5/7  | Gemini CLI MCP konfigürasyonu |
| 6/7  | MCP sunucuları test edilir |
| 7/7  | systemd servisi kurulur (opsiyonel) |

---

## 📁 Proje Yapısı

```
.
├── install.sh     # Ana kurulum ve yapılandırma scripti
├── test.go        # Go JSON unmarshal edge-case testi
└── README.md
```

---

## 🛠 Geliştirme

Katkıda bulunmak için fork'layıp PR açabilirsiniz.  
Sorunlar için [Issues](https://github.com/frkntlr/yap-ai-performance/issues) bölümünü kullanın.

---

## 📄 Lisans

MIT License — © 2026 frkntlr
