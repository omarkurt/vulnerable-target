# Docker Compose Provider

Bu paket, `vulnerable-target` projesi için Docker Compose provider'ını içerir. `compose-go` kütüphanesini kullanarak geliştirilmiş güvenli ve güçlü bir Docker Compose yönetimi sağlar.

## Özellikler

### 🔒 Güvenlik Özellikleri

- **Path Traversal Koruması**: Dosya yolları otomatik olarak temizlenir ve doğrulanır
- **Container Güvenlik Validasyonu**: 
  - Privileged container kontrolü
  - Host network mode kontrolü
  - Tehlikeli capability kontrolü (SYS_ADMIN, NET_ADMIN, SYS_PTRACE)
  - Hassas dizinlere volume mount kontrolü (/etc, /sys, /proc, /dev)
- **Güvenli Temp Dosya Yönetimi**: Compose dosyaları için güvenli geçici dosya oluşturma
- **Context Timeout**: Tüm operasyonlar için configurable timeout

### 🚀 Performans ve Yönetim

- **Project Caching**: Yüklenen projeler memory'de cache'lenir
- **Concurrent Safety**: Thread-safe operasyonlar için sync.RWMutex kullanımı
- **Docker API Entegrasyonu**: Docker client ile gelişmiş container yönetimi
- **Health Check**: Container'ların sağlık durumu kontrolü
- **Orphan Container Temizleme**: Eski container'ların otomatik temizlenmesi

### 📝 Compose-go Entegrasyonu

- **Tam Compose Spec Desteği**: compose-go v2 ile tam uyumluluk
- **YAML Parsing ve Validation**: Otomatik compose dosyası validasyonu
- **Environment Variable Desteği**: .env dosyaları ve custom environment variable'lar
- **Label Management**: Otomatik template labeling

## Kullanım

### Basit Kullanım

```go
// Default configuration ile DockerCompose oluştur
dc := dockercompose.NewDockerCompose()

// Template'i başlat
err := dc.Start(template)
if err != nil {
    log.Fatal(err)
}

// Template'i durdur
err = dc.Stop(template)
if err != nil {
    log.Fatal(err)
}
```

### Özel Konfigürasyon

```go
config := &dockercompose.Config{
    Timeout:       10 * time.Minute,
    RemoveVolumes: true,
    RemoveOrphans: true,
    Environment: map[string]string{
        "CUSTOM_ENV": "value",
    },
    WorkingDir: "/custom/path",
    Verbose:    true,
}

dc := dockercompose.NewDockerComposeWithConfig(config)
```

### Status Kontrolü

```go
status, err := dc.Status(template)
if err != nil {
    log.Fatal(err)
}

for service, serviceStatus := range status.Services {
    fmt.Printf("Service: %s, Running: %v, Healthy: %v\n", 
        service, serviceStatus.Running, serviceStatus.Healthy)
}
```

## API

### DockerCompose Struct

```go
type DockerCompose struct {
    projects map[string]*types.Project  // Cache'lenmiş projeler
    mu       sync.RWMutex              // Thread safety
    config   *Config                   // Provider konfigürasyonu
}
```

### Config Struct

```go
type Config struct {
    Timeout       time.Duration          // İşlem timeout'u
    RemoveVolumes bool                   // Stop'ta volume'leri sil
    RemoveOrphans bool                   // Orphan container'ları temizle
    Environment   map[string]string      // Custom environment variables
    WorkingDir    string                 // Çalışma dizini override
    Verbose       bool                   // Detaylı log çıktısı
}
```

### Provider Interface

```go
type Provider interface {
    Name() string
    Start(template *templates.Template) error
    Stop(template *templates.Template) error
}
```

### Ek Methodlar

- `Status(template *templates.Template) (*ProviderStatus, error)` - Service durumunu kontrol et
- `validateProject(project *types.Project) error` - Güvenlik validasyonu
- `loadProject(ctx context.Context, template *templates.Template) (*types.Project, error)` - Compose projesini yükle

## Docker Client Entegrasyonu

`docker_client.go` dosyası, Docker API ile doğrudan etkileşim için helper methodlar sağlar:

- Container yönetimi (list, start, stop, remove)
- Network yönetimi
- Health check monitoring
- Log streaming
- Image management

## Test

### Unit Tests

```bash
go test ./pkg/provider/dockercompose
```

### Integration Tests

Docker daemon'ın çalışıyor olması gerekir:

```bash
go test -v ./pkg/provider/dockercompose -run Integration
```

### Test Coverage

```bash
go test -cover ./pkg/provider/dockercompose
```

## Güvenlik Notları

1. **Path Validation**: Tüm dosya yolları `filepath.Clean()` ile temizlenir
2. **Container Isolation**: Tehlikeli capability ve mount'lar engellenir
3. **Resource Limits**: Timeout mekanizması ile resource tüketimi sınırlandırılır
4. **Label Management**: Tüm container'lar yönetilebilir şekilde etiketlenir

## Gelecek İyileştirmeler

- [ ] Docker API ile tam entegrasyon (exec yerine)
- [ ] Swarm mode desteği
- [ ] Resource limit configuration
- [ ] Custom health check definitions
- [ ] Service dependency management
- [ ] Rolling update support
- [ ] Backup/restore functionality

## Dependencies

- `github.com/compose-spec/compose-go/v2` - Compose spec parsing
- `github.com/docker/docker` - Docker API client
- `github.com/rs/zerolog` - Structured logging

## Lisans

Bu proje vulnerable-target projesinin bir parçasıdır.
