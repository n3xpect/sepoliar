# sepoliar

Automated Sepolia testnet faucet claimer. Claims ETH every 24 hours using saved Google sessions across multiple wallet addresses, and reports balances via Telegram.

## Features

- Browser automation via Playwright (headless Chromium)
- Claims **0.05 Sepolia ETH** per wallet across multiple accounts
- On-chain balance checks via Sepolia RPC
- Telegram bot notifications on each claim cycle
- Docker and Railway deployment support

## Setup

### 1. Capture Google session

Open a real browser, sign in to Google, and save the session:

```sh
go run . --capture
```

Session is saved to `data/account/` so subsequent runs can reuse it. Repeat this step for each Google account you want to use.

### 2. Run the claimer

```sh
go run . --no-capture
```

Runs the claim loop indefinitely, sleeping ~24h 1m between cycles.

### 3. Check balances

```sh
go run . --balance
```

Prints current balances for the configured wallets and exits.

## Environment Variables

| Variable | Required | Description |
|---|---|---|
| `WALLET_ADDRESSES` | ✅ | Wallet address for claiming ETH |
| `TELEGRAM_BOT_TOKEN` | — | Telegram bot token (both token and chat ID must be set to enable) |
| `TELEGRAM_CHAT_ID` | — | Telegram chat/user ID |
| `LOG_LEVEL` | — | Log level: `debug`, `info`, `warn`, `error` (default: `info`) |
| `ENABLED_TOKENS` | `ETH` | Comma-separated list of tokens to claim: `ETH`, `PYUSD` |
| `TZ` | — | Timezone for log timestamps (e.g. `Europe/Istanbul`) |

Copy `.env.example` to `.env` and fill in the required values.

## Telegram Commands

| Command | Description |
|---|---|
| `/start` | Shows welcome message |
| `/balance` | Returns current wallet balances |

## Deploy

### CLI

```sh
# Build binary
make -f deploy/Makefile build

# Show help
./sepoliar --help

# Save Google session (run once)
./sepoliar --capture

# Start the claim loop
./sepoliar --no-capture

# Check current wallet balances
./sepoliar --balance
```

### Docker

```sh
# Build image
make -f deploy/Makefile docker-build

# Start container
# Note: compose-up automatically sets correct ownership (UID 1000)
# and permissions on data/account/ before starting the container.
make -f deploy/Makefile compose-up

# Stop container
make -f deploy/Makefile compose-down
```

### Railway

```sh
# Push environment variables
make -f deploy/Makefile railway-env-set

# List current variables
make -f deploy/Makefile railway-env-list

# Deploy
make -f deploy/Makefile railway-up
```
