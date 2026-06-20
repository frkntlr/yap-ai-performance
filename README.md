# Yap AI Performance CLI

> **Linux**, **macOS** ve **Windows** platformları için Go (Golang) ile geliştirilmiş, sıfır dış bağımlılıklı, yüksek performanslı ve akıllı bir MCP (Model Context Protocol) Server yönetim, optimizasyon ve bağlam (context) CLI aracı.

---

## 🚀 Temel Özellikler

- **Tekil Yürütülebilir Dosya (Single Executable):** Go ile derlenen, herhangi bir harici bağımlılığa ihtiyaç duymayan tek bir binary dosyası ile kolay dağıtım.
- **Akıllı Bağlam Farkındalığı (`yap context`):** Çalıştırıldığı dizindeki projeyi (Go, JavaScript/TypeScript, Rust, Python, Java) tarar, git durumunu (`git diff --stat`) analiz eder ve yapay zeka modelleri için optimize edilmiş dinamik sistem promptu üretir.
- **Özelleştirme ve Kısayollar (Custom Aliases):** Global (`~/.yaprc`) veya yerel (`./.yaprc`) yapılandırma dosyaları üzerinden sık tekrarlanan uzun prompt görevlerini (`yap code-review`, `yap explain` vb.) bağlam duyarlı kısayol komutlarına bağlar.
- **Güvenlik ve Gözlemlenebilirlik (Safety & Logging):**
  - **Dry-Run Modu (`--dry-run`):** Yapılacak işlemleri gerçekten uygulamadan önce güvenli bir şekilde simüle eder.
  - **Otomatik Yedekleme ve Rollback (`yap rollback`):** Düzenlenen her ayar dosyasının otomatik yedeğini alır ve tek tuşla geri yüklenmesini sağlar.
  - **slog Günlükleme:** Arka plan proxy ve kurulum süreçlerini `~/.yap/logs/` altında günlük log dosyalarına JSON formatında detaylıca kaydeder.
- **Çevre Değişkenleri ve `.env` Desteği:** Çalışma dizinindeki `.env` dosyasını sıfır bağımlılıkla yükleyip `YAP_` ön ekli ayar değişkenlerini ezer.
- **Aktif Sistem Teşhisi (`yap status`):** Sistem gereksinimleri, yüklü paketler, CGC yama durumları ve aktif port/servis kontrollerini gerçekleştirir.
- **Go-Native JSON-RPC Proxy:** CodeGraphContext ve Graphify sunucularını yöneten, log dönen kararlı proxy katmanı.

---

## 💻 Kurulum ve Dağıtım

### Derleme (Build)
Uygulamayı yerel olarak derlemek için aşağıdaki komutları kullanabilirsiniz:

```bash
# Bağımlılıkları kontrol et ve statik analiz çalıştır
go vet ./...

# Birim testlerini çalıştır
make test

# Tüm platformlar için derleme yap (dist/ klasörü altına çıktılar oluşturulur)
make build-all
```

---

## 🛠 Kullanım ve Komutlar

### 1. Kurulum (`yap install`)
Sistem gereksinimlerini kontrol eder, `pipx` veya `uv` üzerinden CodeGraphContext ve Graphify araçlarını kurar ve gerekli KuzuDB/Protokol yamalarını uygular.

```bash
yap install
# Simülasyon modu
yap install --dry-run
# Yalnızca konfigürasyon adımı
yap install --only=config
```

### 2. Teşhis ve Sağlık Kontrolü (`yap status`)
Kurulu bileşenleri ve yama durumlarını aktif olarak test eder.

```bash
yap status
```

### 3. Geri Yükleme (`yap rollback`)
Yapılan konfigürasyon değişikliklerini otomatik olarak yedeklenen son kararlı sürüme geri döndürür.

```bash
yap rollback
```

### 4. Bağlam Analizi (`yap context`)
Çalışma dizinindeki proje yapısını ve Git değişikliklerini analiz eder.

```bash
# Özet raporu basar
yap context

# AI modeline beslenecek sistem promptunu stdout'a yazar
yap context --prompt

# Git diff detaylarını prompta dahil eder
yap context --prompt --with-diff

# Prompt çıktısını kaydeder (varsayılan konuma: ~/.yap/context.md)
yap context --prompt --save

# JSON formatında çıktı verir
yap context --json
```

### 5. Özel Kısayollar (Örn: `yap code-review`)
`.yaprc` dosyanızda tanımladığınız özel takma adlar (aliases) dinamik olarak birer alt komuta dönüşür:

```bash
yap code-review --with-diff
```

---

## ⚙️ Yapılandırma (`.yaprc`)

Kişisel ayarlarınızı, varsayılan modellerinizi ve takma adlarınızı `~/.yaprc` (global) veya proje dizinindeki `.yaprc` (yerel) dosyasında tutabilirsiniz:

```json
{
  "default_model": "gemini-1.5-flash",
  "log_level": "INFO",
  "aliases": {
    "code-review": "Lütfen aşağıdaki değişikliklerin zaman karmaşıklığını (Big O) hesapla ve bellek sızıntılarını analiz et.",
    "explain": "Bu kod parçasının ne yaptığını adım adım anlat."
  }
}
```

---

## 📁 Proje Dizin Yapısı

```
.
├── cmd/
│   └── yap/
│       └── main.go           # CLI Giriş Noktası ve Komut Yönetimi
├── internal/
│   ├── backup/               # Güvenli zaman damgalı yedekleme ve rollback modülü
│   ├── config/               # Katmanlı JSON tabanlı .yaprc yapılandırma yöneticisi
│   ├── confirm/              # bufio tabanlı kullanıcı onay arayüzü
│   ├── detector/             # İşletim sistemi ve paket yöneticisi tespiti
│   ├── dryrun/               # Değişiklik simülasyon çıktıları
│   ├── env/                  # Sıfır dış bağımlılıklı .env ayrıştırıcısı
│   ├── gitinfo/              # os/exec tabanlı git status/diff stat analizörü
│   ├── installer/            # 6 adımlı sırayla kurulum adımları
│   ├── logger/               # slog tabanlı dual (JSON/Terminal) loglama sistemi
│   ├── proxy/                # Go-native transparent JSON-RPC proxy
│   └── scanner/              # Proje teknoloji ve bağımlılık tarayıcısı
├── pkg/
│   ├── fileutil/             # Dosya kopyalama ve arama yardımcıları
│   ├── jsonutil/             # JSON okuma/yazma yardımcıları
│   ├── promptbuilder/        # Dinamik prompt şablon mimarisi
│   └── runner/               # Komut çalıştırma ve çıktı yakalama araçları
├── Makefile                  # Çoklu platform derleme komutları
├── go.mod
└── README.md
```

---

## 📄 Lisans

MIT License — © 2026 frkntlr
