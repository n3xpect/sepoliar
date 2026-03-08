package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/playwright-community/playwright-go"

	"sepoliar/internal/app"
	"sepoliar/internal/domain"
	"sepoliar/internal/infra/browser"
	"sepoliar/internal/infra/rpc"
	"sepoliar/internal/infra/telegram"
	"sepoliar/internal/usecase"
	"sepoliar/pkg/config"
	"sepoliar/pkg/logger"
)

const (
	faucetURLETH         = "https://cloud.google.com/application/web3/faucet/ethereum/sepolia"
	faucetURLPYUSD       = "https://cloud.google.com/application/web3/faucet/ethereum/sepolia/pyusd"
	pyusdContractAddress = "0xCaC524BcA292aaade2DF8A05cC58F0a65B1B3bB9"
)

func main() {
	capture := flag.Bool("capture", false, "")
	noCapture := flag.Bool("no-capture", false, "")
	balance := flag.Bool("balance", false, "")

	flag.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, "Usage: sepoliar [--capture | --no-capture | --balance]\n\n")
		_, _ = fmt.Fprintf(os.Stderr, "  --capture     Saves Google session by opening a browser for sign-in\n")
		_, _ = fmt.Fprintf(os.Stderr, "  --no-capture  Starts the faucet claim loop using the saved session\n")
		_, _ = fmt.Fprintf(os.Stderr, "  --balance     Prints current wallet balances and exits\n")
		_, _ = fmt.Fprintf(os.Stderr, "  --help        Show this help message\n")
	}

	flag.Parse()

	if len(os.Args) == 1 {
		flag.Usage()
		os.Exit(0)
	}

	cfg := config.Load()
	lg := logger.NewLog(cfg.LogLevel)
	ctx := context.Background()

	if *balance {
		configs := buildConfigs(cfg)
		fetcher := rpc.New(cfg.RPC.SepoliaRPCURL)
		type result struct {
			name string
			bal  string
			err  error
		}
		results := make([]result, len(configs))
		for i, c := range configs {
			bal, err := fetcher.GetBalance(ctx, c)
			results[i] = result{name: c.TokenName, bal: bal, err: err}
		}
		for _, r := range results {
			if r.err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "%s: error: %v\n", r.name, r.err)
				continue
			}
			fmt.Printf("%-6s %s\n", r.name+":", r.bal)
		}
		return
	}

	pw, err := playwright.Run()
	if err != nil {
		lg.Fatal(ctx, "Could not start playwright", logger.Err(err))
	}
	defer func() { _ = pw.Stop() }()

	switch {
	case *capture:
		capturer := browser.NewSessionCapturer(pw, lg)
		if err := capturer.CaptureSession(ctx); err != nil {
			lg.Fatal(ctx, "Could not capture session", logger.Err(err))
		}

	case *noCapture:
		configs := buildConfigs(cfg)
		if (cfg.Telegram.BotToken == "") != (cfg.Telegram.ChatID == "") {
			lg.Warn(ctx, "Only one of TELEGRAM_BOT_TOKEN / TELEGRAM_CHAT_ID is set — Telegram disabled.")
		}
		claimer := browser.New(pw, lg)
		notifier := telegram.New(cfg.Telegram.BotToken, cfg.Telegram.ChatID, lg)
		balanceFetcher := rpc.New(cfg.RPC.SepoliaRPCURL)
		uc := usecase.New(lg, balanceFetcher)
		a := app.New(configs, claimer, notifier, uc, lg)
		a.Run(ctx)

	default:
		flag.Usage()
		os.Exit(0)
	}
}

func buildConfigs(cfg *config.Config) []domain.ClaimConfig {
	all := []domain.ClaimConfig{
		{
			FaucetURL:     faucetURLETH,
			WalletAddress: cfg.Wallet.ETH,
			ButtonText:    "Get 0.05 Sepolia ETH",
			TokenName:     "ETH",
			TokenDecimals: 18,
		},
		{
			FaucetURL:     faucetURLPYUSD,
			WalletAddress: cfg.Wallet.ETH,
			ButtonText:    "Get 100 Sepolia PYUSD",
			TokenName:     "PYUSD",
			TokenAddress:  pyusdContractAddress,
			TokenDecimals: 6,
		},
	}

	enabled := make(map[string]bool)
	for _, t := range strings.Split(cfg.EnabledTokens, ",") {
		enabled[strings.TrimSpace(strings.ToUpper(t))] = true
	}

	var result []domain.ClaimConfig
	for _, c := range all {
		if enabled[c.TokenName] {
			result = append(result, c)
		}
	}
	return result
}
