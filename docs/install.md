# Kurulum

## Gereksinimler

- **Git**, **Python 3**, **pipx**, **uv** (yoksa `yap install` kurmaya çalışır)
- Go (yalnızca kaynaktan derleyecekseniz)

## Linux

```bash
bash install.sh
# veya
go build -ldflags="-s -w" -o dist/yap-linux-amd64 ./cmd/yap
./dist/yap-linux-amd64 install
```

Binary varsayılan konumu: `~/.local/bin/yap`

## macOS

```bash
bash install_mac.sh
```

Homebrew (`brew`) önerilir; `yap install` bağımlılıkları brew üzerinden de kurabilir.

## Windows

```powershell
# 1) Binary
go build -ldflags="-s -w" -o dist\yap-windows-amd64.exe .\cmd\yap
# veya release dist\yap-windows-amd64.exe kullanın

# 2) Bootstrap → yap install
powershell -ExecutionPolicy Bypass -File install.ps1

# Simülasyon
powershell -ExecutionPolicy Bypass -File install.ps1 -DryRun
```

Binary: `%LOCALAPPDATA%\Programs\yap\yap.exe` (kullanıcı PATH’ine eklenir). Yeni terminal açın.

## `yap install` adımları

1. Bağımlılıklar (git, python, pipx, uv)  
2. CodeGraphContext (pipx) + Graphifyy (uv)  
3. CGC runtime yamaları  
4. Binary deploy + MCP/skill/Zed + graphify platform hook’ları  
5. Doğrulama  

### Sık kullanılan bayraklar

```bash
yap install --dry-run          # hiçbir şeyi yazmadan simüle et
yap install --only=deps
yap install --only=tools
yap install --only=patch
yap install --only=config      # MCP + skills + binary (IDE yenilemesi sonrası yeterli)
yap install --only=verify
```

## Kurulum sonrası

```bash
yap status
```

Kırmızı kalan IDE config satırları için tekrar: `yap install --only=config`, ardından IDE’yi yeniden başlatın.

Ayrıntılı IDE path’leri: [ides.md](ides.md)
