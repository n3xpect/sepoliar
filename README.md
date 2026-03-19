# sepoliar

## Overview

The [Google Cloud Web3 Faucet](https://cloud.google.com/application/web3/faucet/ethereum/sepolia) is a Sepolia testnet faucet that grants **0.05 ETH** and/or **100 PYUSD** per request and requires a Google account sign-in with a 24-hour cooldown. sepoliar automates the claim process: it saves encrypted Google sessions via Playwright, runs the claim cycle across multiple wallet addresses and token types, and reports results via Telegram.

## Features

- Browser automation via Playwright (headless Chromium)
- Claims **0.05 Sepolia ETH** and/or **100 Sepolia PYUSD** per wallet across multiple accounts
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
go run . --google-sign-in
```

You will be prompted for an encryption key. Session is saved to `data/account/` so subsequent runs can reuse it. Repeat this step for each Google account you want to use.

### 2. Run the claimer

```sh
go run . --claim
```

If Telegram is configured, the app waits for a `/claim` command before starting.
If Telegram is not configured, the claim loop starts immediately.

> **If Telegram is configured:** The app starts and waits for a `/claim` command. Once you send `/claim` to the bot, the cycle begins. Bot replies with one of the following:
> - `Claim cycle starting...` — no active cooldown, cycle has started
> - `Cooldown active\nNext run: <date> - <time>\nRemaining: <duration>\nWaiting...` — cooldown is active, cycle will not start until it expires
> If Telegram is not configured, the loop starts immediately without waiting.

Each cycle sleeps ~24h 1m between runs. The next wake-up time is calculated from the last on-chain transaction timestamp via Etherscan.

### 3. Check balances

```sh
go run . --balance
```

Prints current balances for the configured wallets and exits. Does not require an encryption key.

## CLI Flags

| Flag | Short | Description |
|---|---|---|
| `--google-sign-in` | `-g` | Opens a browser for Google sign-in and saves the encrypted session |
| `--claim` | `-c` | Starts the faucet claim loop using saved sessions |
| `--balance` | `-b` | Prints current wallet balances and exits |
| `--help` | `-h` | Show help message |

## Environment Variables

| Variable | Required | Description |
|---|---|---|
| `LOG_LEVEL` | — | Log level: `debug`, `info`, `warn`, `error` (default: `info`) |
| `TZ` | — | Timezone for log timestamps (e.g. `Europe/Istanbul`) |
| `TELEGRAM_BOT_TOKEN` | — | Telegram bot token (both token and chat ID must be set to enable) |
| `TELEGRAM_CHAT_ID` | — | Telegram chat/user ID |
| `WALLET_ADDRESSES` | ✅ | Comma-separated wallet addresses for claiming |
| `ETHERSCAN_API_KEY` | ✅ | Etherscan API key for last-tx lookup (cooldown calculation) |
| `SEPOLIAR_ENCRYPTION_KEY` | Docker: ✅ | Encryption key for session files; used by `--google-sign-in`, `--claim` (required in Docker — no interactive prompt; optional in CLI) |

Copy `.env.example` to `.env` and fill in the required values.

## Telegram Commands

| Command | Description |
|---|---|
| `/claim` | Starts the claim cycle (only works if the app is waiting for it) |
| `/balance` | Returns current wallet balances (only available after the first claim cycle completes) |

> When Telegram is configured, `--claim` blocks on startup and waits for `/claim` before proceeding. If a cooldown is active, `/claim` returns a cooldown message with remaining time instead of starting the cycle. If a cycle is already running, the bot replies "Claim is already running."

### Claim Flow with Telegram

1. Run `./sepoliar --claim` — the app starts and waits for `/claim`
2. Send `/claim` to the Telegram bot
3. If cooldown is active → bot replies with `Cooldown active / Next run / Remaining / Waiting...`; send `/claim` again after the cooldown to start
4. If no cooldown → bot replies with `Claim cycle starting...` and the cycle begins
5. Once the cycle completes, the app sleeps ~24h 1m until the next run

## Deploy

### CLI

```sh
# Build binary
make build

# Show help
./sepoliar --help

# Save Google session (run once per account)
./sepoliar --google-sign-in

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

