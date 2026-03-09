# sepoliar

Sepolia testnet faucet'inden otomatik token talep aracı. Birden fazla hesap ve cüzdan adresiyle kaydedilmiş Google oturumları kullanarak her 24 saatte bir ETH talep eder; bakiyeleri Telegram üzerinden bildirir.

## Özellikler

- Playwright ile tarayıcı otomasyonu (başsız Chromium)
- Birden fazla hesap ile cüzdan başına **0.05 Sepolia ETH** talebi
- Sepolia RPC üzerinden zincir içi bakiye sorgulama
- Her claim döngüsünde Telegram bot bildirimi
- Docker ve Railway ile dağıtım desteği

## Kurulum

### 1. Google oturumunu kaydet

Gerçek bir tarayıcıda Google hesabına giriş yapıp oturumu kaydetmek için:

```sh
go run . --capture
```

Oturum `data/account/` dizinine kaydedilir; sonraki çalıştırmalar bu oturumu yeniden kullanır. Birden fazla Google hesabı kullanmak istiyorsanız bu adımı her hesap için tekrarlayın.

### 2. Claimer'ı başlat

```sh
go run . --no-capture
```

Döngü süresiz çalışır; her claim arasında ~24 saat 1 dakika bekler.

### 3. Bakiye sorgula

```sh
go run . --balance
```

Yapılandırılmış cüzdanların mevcut bakiyelerini yazdırıp çıkar.

## Ortam Değişkenleri

| Değişken | Zorunlu | Açıklama |
|---|---|---|
| `WALLET_ADDRESSES` | ✅ | ETH talebi için cüzdan adresleri |
| `TELEGRAM_BOT_TOKEN` | — | Telegram bot token'ı (etkinleştirmek için token ve chat ID birlikte girilmeli) |
| `TELEGRAM_CHAT_ID` | — | Telegram sohbet/kullanıcı ID'si |
| `LOG_LEVEL` | — | Log seviyesi: `debug`, `info`, `warn`, `error` (varsayılan: `info`) |
| `ENABLED_TOKENS` | `ETH` | Claim edilecek token listesi (virgülle ayrılmış): `ETH`, `PYUSD` |
| `TZ` | — | Log zaman damgaları için saat dilimi (örn. `Europe/Istanbul`) |

`.env.example` dosyasını `.env` olarak kopyalayıp gerekli değerleri doldurun.

## Telegram Komutları

| Komut | Açıklama |
|---|---|
| `/start` | Karşılama mesajını gösterir |
| `/balance` | Mevcut cüzdan bakiyelerini döndürür |

## Dağıtım

### CLI

```sh
# Binary'yi derle
make -f deploy/Makefile build

# Yardım mesajını göster
./sepoliar --help

# Google oturumunu kaydet (bir kez çalıştır)
./sepoliar --capture

# Claim döngüsünü başlat
./sepoliar --no-capture

# Mevcut cüzdan bakiyelerini kontrol et
./sepoliar --balance
```

### Docker

```sh
# İmaj oluştur
make -f deploy/Makefile docker-build

# Konteyneri başlat
# Not: compose-up, container başlamadan önce data/account/ dizininin
# sahipliğini (UID 1000) ve izinlerini otomatik olarak düzeltir.
make -f deploy/Makefile compose-up

# Konteyneri durdur
make -f deploy/Makefile compose-down
```

### Railway

```sh
# Ortam değişkenlerini aktar
make -f deploy/Makefile railway-env-set

# Mevcut değişkenleri listele
make -f deploy/Makefile railway-env-list

# Dağıt
make -f deploy/Makefile railway-up
```
