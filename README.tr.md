# Sepoliar

## Nedir?

[Google Cloud Web3 Faucet](https://cloud.google.com/application/web3/faucet/ethereum/sepolia), Sepolia testnet için istek başına **0.05 ETH** veren ve Google hesabı gerektiren bir faucet'tir. Sepoliar, bu faucet'ten token talep işlemini otomatikleştirir: Playwright aracılığıyla şifreli Google oturumlarını kaydeder, birden fazla cüzdan adresinde claim döngüsünü çalıştırır ve sonuçları Telegram üzerinden bildirir.

## Özellikler

- Playwright ile tarayıcı otomasyonu (headless Chromium)
- Birden fazla hesap ile cüzdan başına **0.05 Sepolia ETH** talebi
- Sepolia RPC üzerinden zincir içi bakiye sorgulama
- Etherscan üzerinden son işlem zamanına dayalı başlangıç cooldown kontrolü
- Her claim döngüsünde Telegram bot bildirimi
- Telegram tetiklemeli başlatma: bot yapılandırıldıysa döngü yalnızca `/claim` komutu gönderildikten sonra başlar
- Şifreli oturum dosyaları (AES-256-GCM)
- Docker ile dağıtım desteği

## Kurulum

### 1. Google oturumunu kaydet

[cloud.google.com/application/web3/faucet/ethereum/sepolia](https://cloud.google.com/application/web3/faucet/ethereum/sepolia) adresindeki faucet bir Google hesabı gerektirir. Gerçek bir tarayıcıda Google hesabına giriş yapıp şifreli oturumu kaydetmek için:

```sh
go run . --capture
```

Şifreleme anahtarı girmeniz istenecektir. Oturum `data/account/` dizinine kaydedilir; sonraki çalıştırmalar bu oturumu yeniden kullanır. Birden fazla Google hesabı kullanmak istiyorsanız bu adımı her hesap için tekrarlayın.

### 2. Claimer'ı başlat

```sh
go run . --claim
```

Telegram yapılandırıldıysa uygulama, `/claim` komutu gönderilene kadar bekler.
Telegram yapılandırılmamışsa döngü hemen başlar.

> **Telegram yapılandırıldıysa:** Uygulama başlar ve `/claim` komutunu beklemeye girer. Telegram botuna `/claim` gönderince döngü başlar. Bot aşağıdaki yanıtları verir:
> - `Claim cycle starting...` — döngü başladı
> - Cooldown aktifse: `Cooldown active\nNext run: <tarih> - <saat>\nRemaining: <süre>\nWaiting...`
> Telegram yapılandırılmamışsa döngü beklenmeden hemen başlar.

Her döngü arasında ~24 saat 1 dakika beklenir. Bir sonraki çalışma zamanı, Etherscan üzerinden alınan son zincir içi işlem zamanına göre hesaplanır.

### 3. Bakiye sorgula

```sh
go run . --balance
```

Yapılandırılmış cüzdanların mevcut bakiyelerini yazdırıp çıkar. Şifreleme anahtarı gerektirmez.

### 4. Mevcut oturumları şifrele

```sh
go run . --encrypt
```

Şifrelenmemiş mevcut `.json` oturum dosyaları için tek seferlik geçiş adımı.

## CLI Bayrakları

| Bayrak | Kısa | Açıklama |
|---|---|---|
| `--capture` | `-C` | Tarayıcı açar, Google'a giriş yapılır ve şifreli oturum kaydedilir |
| `--claim` | `-c` | Kaydedilmiş oturumları kullanarak claim döngüsünü başlatır |
| `--balance` | `-b` | Mevcut cüzdan bakiyelerini yazdırıp çıkar |
| `--encrypt` | `-e` | Mevcut düz metin oturum dosyalarını şifreler |
| `--help` | `-h` | Yardım mesajını gösterir |

## Ortam Değişkenleri

| Değişken | Zorunlu | Açıklama |
|---|---|---|
| `LOG_LEVEL` | — | Log seviyesi: `debug`, `info`, `warn`, `error` (varsayılan: `info`) |
| `TZ` | — | Log zaman damgaları için saat dilimi (örn. `Europe/Istanbul`) |
| `TELEGRAM_BOT_TOKEN` | — | Telegram bot token'ı (etkinleştirmek için token ve chat ID birlikte girilmeli) |
| `TELEGRAM_CHAT_ID` | — | Telegram sohbet/kullanıcı ID'si |
| `ENABLED_TOKENS` | ✅ | Claim edilecek token listesi (virgülle ayrılmış): `ETH`, `PYUSD` |
| `WALLET_ADDRESSES` | ✅ | Claim için cüzdan adresleri (virgülle ayrılmış) |
| `ETHERSCAN_API_KEY` | ✅ | Son işlem zamanı sorgusu için Etherscan API anahtarı (cooldown hesabı) |
| `SEPOLIAR_ENCRYPTION_KEY` | Docker: ✅ | Oturum dosyaları şifreleme anahtarı; `--capture`, `--claim`, `--encrypt` için geçerlidir (Docker'da zorunlu — etkileşimli istem yok; CLI'da isteğe bağlı) |

`.env.example` dosyasını `.env` olarak kopyalayıp gerekli değerleri doldurun.

## Telegram Komutları

| Komut | Açıklama |
|---|---|
| `/claim` | Claim döngüsünü başlatır (yalnızca uygulama bekleme modundayken çalışır) |
| `/balance` | Mevcut cüzdan bakiyelerini döndürür |

> Telegram yapılandırıldığında `--claim` başlangıçta bloklar ve devam etmeden önce `/claim` komutunu bekler. Bir döngü zaten çalışırken `/claim` gönderilirse bot "Claim is already running." yanıtı verir.

### Telegram ile Claim Akışı

1. `./sepoliar --claim` çalıştır — uygulama `/claim` beklemeye girer
2. Telegram botuna `/claim` gönder
3. Bot `Claim cycle starting...` ile yanıt verir
4. Cooldown aktifse bot `Cooldown active / Next run / Remaining / Waiting...` mesajı gönderir ve döngü bekler
5. Cooldown dolunca claim işlemi gerçekleşir

## Dağıtım

### CLI

```sh
# Binary'yi derle
make build

# Yardım mesajını göster
./sepoliar --help

# Google oturumunu kaydet (hesap başına bir kez çalıştır)
./sepoliar --capture

# Claim döngüsünü başlat
./sepoliar --claim

# Mevcut cüzdan bakiyelerini kontrol et
./sepoliar --balance
```

### Docker

```sh
# İmaj oluştur
make docker-build

# Konteyneri başlat (detach modda başlar, ardından log akışı açılır)
make docker-up        # Linux
make docker-up-mac    # macOS

# Konteyneri durdur
make docker-down
```

