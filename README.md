# sepoliar

## Overview

The [Google Cloud Web3 Faucet](https://cloud.google.com/application/web3/faucet/ethereum/sepolia) is a Sepolia testnet faucet that grants **0.05 ETH** per request and requires a Google account sign-in with a 24-hour cooldown. sepoliar automates the claim process: it saves encrypted Google sessions via Playwright, runs the claim cycle across multiple wallet addresses, and reports results via Telegram.

## Features

- Browser automation via Playwright (headless Chromium)
- Claims **0.05 Sepolia ETH** per wallet across multiple accounts
- On-chain balance checks via Sepolia RPC
- Startup cooldown check via Etherscan — skips waiting if the interval hasn't elapsed
- Telegram bot notifications on each claim cycle
- Telegram-triggered claim start: when a bot is configured, the loop only begins after `/claim` is sent
- Encrypted session files (AES-256-GCM)
- Docker deployment support

## Setup

### 1. Capture Google session

The faucet at [cloud.google.com/application/web3/faucet/ethereum/sepolia](https://cloud.google.com/application/web3/faucet/ethereum/sepolia) requires a Google account. Open a real browser, sign in to Google, and save the encrypted session:

```sh
go run . --capture
```

You will be prompted for an encryption key. Session is saved to `data/account/` so subsequent runs can reuse it. Repeat this step for each Google account you want to use.

### 2. Run the claimer

```sh
go run . --claim
```

If Telegram is configured, the app waits for a `/claim` command before starting.
If Telegram is not configured, the claim loop starts immediately.

> **If Telegram is configured:** The app starts and waits for a `/claim` command. Once you send `/claim` to the bot, the cycle begins. The bot sends the following responses:
> - `Claim cycle starting...` — cycle has started
> - If cooldown is active: `Cooldown active\nNext run: <date> - <time>\nRemaining: <duration>\nWaiting...`
> If Telegram is not configured, the loop starts immediately without waiting.

Each cycle sleeps ~24h 1m between runs. The next wake-up time is calculated from the last on-chain transaction timestamp via Etherscan.

### 3. Check balances

```sh
go run . --balance
```

Prints current balances for the configured wallets and exits. Does not require an encryption key.

### 4. Encrypt existing plaintext sessions

```sh
go run . --encrypt
```

One-time migration for pre-existing unencrypted `.json` session files.

## CLI Flags

| Flag | Short | Description |
|---|---|---|
| `--capture` | `-C` | Opens a browser for Google sign-in and saves the encrypted session |
| `--claim` | `-c` | Starts the faucet claim loop using saved sessions |
| `--balance` | `-b` | Prints current wallet balances and exits |
| `--encrypt` | `-e` | Encrypts existing plaintext session files |
| `--help` | `-h` | Show help message |

## Environment Variables

| Variable | Required | Description |
|---|---|---|
| `LOG_LEVEL` | — | Log level: `debug`, `info`, `warn`, `error` (default: `info`) |
| `TZ` | — | Timezone for log timestamps (e.g. `Europe/Istanbul`) |
| `TELEGRAM_BOT_TOKEN` | — | Telegram bot token (both token and chat ID must be set to enable) |
| `TELEGRAM_CHAT_ID` | — | Telegram chat/user ID |
| `ENABLED_TOKENS` | ✅ | Comma-separated tokens to claim: `ETH`, `PYUSD` |
| `WALLET_ADDRESSES` | ✅ | Comma-separated wallet addresses for claiming |
| `ETHERSCAN_API_KEY` | ✅ | Etherscan API key for last-tx lookup (cooldown calculation) |
| `SEPOLIAR_ENCRYPTION_KEY` | Docker: ✅ | Encryption key for session files; used by `--capture`, `--claim`, `--encrypt` (required in Docker — no interactive prompt; optional in CLI) |

Copy `.env.example` to `.env` and fill in the required values.

## Telegram Commands

| Command | Description |
|---|---|
| `/claim` | Starts the claim cycle (only works if the app is waiting for it) |
| `/balance` | Returns current wallet balances |

> When Telegram is configured, `--claim` blocks on startup and waits for `/claim` before proceeding. If `/claim` is sent while a cycle is already running, the bot replies "Claim is already running."

### Claim Flow with Telegram

1. Run `./sepoliar --claim` — the app starts and waits for `/claim`
2. Send `/claim` to the Telegram bot
3. Bot replies with `Claim cycle starting...`
4. If cooldown is active, the bot sends `Cooldown active / Next run / Remaining / Waiting...` and the loop waits
5. Once the cooldown expires, the claim is executed

## Deploy

### CLI

```sh
# Build binary
make build

# Show help
./sepoliar --help

# Save Google session (run once per account)
./sepoliar --capture

# Start the claim loop
./sepoliar --claim

# Check current wallet balances
./sepoliar --balance
```

### Docker

```sh
# Build image
make docker-build

# Start container (detaches, then streams logs)
make docker-up        # Linux
make docker-up-mac    # macOS

# Stop container
make docker-down
```

