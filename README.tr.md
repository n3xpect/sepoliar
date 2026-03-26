# Sepoliar

## Nedir?

[Google Cloud Web3 Faucet](https://cloud.google.com/application/web3/faucet/ethereum/sepolia), Sepolia testnet için istek başına **0.05 ETH** ve/veya **100 PYUSD** veren ve Google hesabı gerektiren bir faucet'tir. Sepoliar, bu faucet'ten token talep işlemini otomatikleştirir: Playwright aracılığıyla şifreli Google oturumlarını kaydeder, birden fazla cüzdan adresi ve token türü için claim döngüsünü çalıştırır ve sonuçları Telegram üzerinden bildirir.

## Özellikler

- Playwright ile tarayıcı otomasyonu (headless Chromium)
- Birden fazla hesap ile cüzdan başına **0.05 Sepolia ETH** ve/veya **100 Sepolia PYUSD** talebi
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
go run . --google-sign-in
```

Şifreleme anahtarı girmeniz istenecektir. Oturum `data/account/` dizinine kaydedilir; sonraki çalıştırmalar bu oturumu yeniden kullanır. Birden fazla Google hesabı kullanmak istiyorsanız bu adımı her hesap için tekrarlayın.

### 2. Claimer'ı başlat

```sh
go run . --claim
```

Telegram yapılandırıldıysa uygulama, `/claim` komutu gönderilene kadar bekler.
Telegram yapılandırılmamışsa döngü hemen başlar.

> **Telegram yapılandırıldıysa:** Uygulama başlar ve `/claim` komutunu beklemeye girer. Telegram botuna `/claim` gönderince bot şu yanıtlardan birini verir:
> - `Claim cycle starting...` — cooldown yok, döngü başladı
> - `Cooldown active\nNext run: <tarih> - <saat>\nRemaining: <süre>\nWaiting...` — cooldown aktif, süresi dolana kadar döngü başlamaz
> Telegram yapılandırılmamışsa döngü beklenmeden hemen başlar.

Her döngü arasında ~24 saat 1 dakika beklenir. Bir sonraki çalışma zamanı, Etherscan üzerinden alınan son zincir içi işlem zamanına göre hesaplanır.

### 3. Bakiye sorgula

```sh
go run . --balance
```

Yapılandırılmış cüzdanların mevcut bakiyelerini yazdırıp çıkar. Şifreleme anahtarı gerektirmez.

## CLI Bayrakları

| Bayrak | Kısa | Açıklama |
|---|---|---|
| `--google-sign-in` | `-g` | Tarayıcı açar, Google'a giriş yapılır ve şifreli oturum kaydedilir |
| `--claim` | `-c` | Kaydedilmiş oturumları kullanarak claim döngüsünü başlatır |
| `--balance` | `-b` | Mevcut cüzdan bakiyelerini yazdırıp çıkar |
| `--help` | `-h` | Yardım mesajını gösterir |

## Ortam Değişkenleri

| Değişken | Zorunlu | Açıklama |
|---|---|---|
| `LOG_LEVEL` | — | Log seviyesi: `debug`, `info`, `warn`, `error` (varsayılan: `info`) |
| `TZ` | — | Log zaman damgaları için saat dilimi (örn. `Europe/Istanbul`) |
| `TELEGRAM_BOT_TOKEN` | — | Telegram bot token'ı (etkinleştirmek için token ve chat ID birlikte girilmeli) |
| `TELEGRAM_CHAT_ID` | — | Telegram sohbet/kullanıcı ID'si |
| `WALLET_ADDRESSES` | ✅ | Claim için cüzdan adresleri (virgülle ayrılmış) |
| `ETHERSCAN_API_KEY` | ✅ | Son işlem zamanı sorgusu için Etherscan API anahtarı (cooldown hesabı) |
| `SEPOLIAR_ENCRYPTION_KEY` | Docker: ✅ | Oturum dosyaları şifreleme anahtarı; `--google-sign-in`, `--claim` için geçerlidir (`make docker-up` sırasında interaktif olarak sorulur; CLI'da isteğe bağlı) |

`.env.example` dosyasını `.env` olarak kopyalayıp gerekli değerleri doldurun.

## Telegram Komutları

| Komut | Açıklama |
|---|---|
| `/claim` | Claim döngüsünü başlatır (yalnızca uygulama bekleme modundayken çalışır) |
| `/balance` | Mevcut cüzdan bakiyelerini döndürür (yalnızca ilk claim döngüsü tamamlandıktan sonra kullanılabilir) |
| `/info` | Hesap adlarını ve ilişkili cüzdan adreslerini döndürür |

> Telegram yapılandırıldığında `--claim` başlangıçta bloklar ve devam etmeden önce `/claim` komutunu bekler. Cooldown aktifse `/claim`, kalan süreyle birlikte cooldown mesajı döndürür; döngü başlamaz. Claim döngüsü devam ediyorsa bot "Claim is in progress. Please wait..." yanıtı verir. Tüm hesaplar eş zamanlı olarak claim edilir.

### Telegram ile Claim Akışı

1. `./sepoliar --claim` çalıştır — uygulama `/claim` beklemeye girer
2. Telegram botuna `/claim` gönder
3. Cooldown aktifse → bot `Cooldown active / Next run / Remaining / Waiting...` ile yanıt verir; cooldown bittikten sonra tekrar `/claim` gönder
4. Cooldown yoksa → bot `Claim cycle starting...` ile yanıt verir ve döngü başlar
5. Döngü tamamlanınca uygulama ~24s 1dk uyur

## Dağıtım

### CLI

```sh
# Binary'yi derle
make build

# Yardım mesajını göster
./sepoliar --help

# Google oturumunu kaydet (hesap başına bir kez çalıştır)
./sepoliar --google-sign-in

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

