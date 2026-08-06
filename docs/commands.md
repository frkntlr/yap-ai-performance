# Komut referansı

## `yap install`

Sistem bağımlılıkları, araçlar, yamalar, binary ve IDE config kurulumu.

```bash
yap install
yap install --dry-run
yap install --only=deps|tools|patch|config|verify
```

## `yap status`

Araçlar, pipx yamaları, MCP config’ler, skills, proxy liveness.

```bash
yap status
```

## `yap context`

Çalışma dizinindeki proje + git özeti.

```bash
yap context
yap context --prompt
yap context --prompt --with-diff
yap context --prompt --save
yap context --json
```

## `yap rollback`

Son yedeklenen config dosyalarını geri yükler.

```bash
yap rollback
```

## `yap proxy`

IDE’lerin çağırdığı MCP proxy (normalde elle çalıştırılmaz).

```bash
yap proxy cgc
yap proxy graphify
```

## Aliases (`.yaprc`)

```bash
yap code-review --with-diff
yap explain
```

Alias metinleri `yap context` çıktısıyla birleştirilebilir; tanım `~/.yaprc` veya `./.yaprc` içinde.

## Ortam

- `.env` içindeki `YAP_*` değişkenleri yüklenir.
- Loglar: `~/.yap/logs/`
