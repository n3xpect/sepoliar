package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

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
		_, _ = fmt.Fprintf(os.Stderr, "  --capture     Opens a browser for Google sign-in and saves the session (email auto-detected)\n")
		_, _ = fmt.Fprintf(os.Stderr, "  --no-capture  Starts the faucet claim loop using saved sessions\n")
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
		fetcher := rpc.New(cfg.RPC.SepoliaRPCURL)

		type tokenResult struct {
			name string
			bal  string
			err  error
		}
		type walletResult struct {
			wallet string
			tokens []tokenResult
		}

		results := make([]walletResult, len(cfg.Wallets))
		var wg sync.WaitGroup

		for i, w := range cfg.Wallets {
			configs := buildConfigs(cfg, w.Address)
			tokens := make([]tokenResult, len(configs))
			wg.Add(len(configs))
			for j, c := range configs {
				go func(j int, c domain.ClaimConfig) {
					defer wg.Done()
					bal, err := fetcher.GetBalance(ctx, c)
					tokens[j] = tokenResult{name: c.TokenName, bal: bal, err: err}
				}(j, c)
			}
			results[i] = walletResult{wallet: w.Address, tokens: tokens}
		}

		wg.Wait()

		for i, wr := range results {
			if i > 0 {
				fmt.Println()
			}
			fmt.Printf("Wallet: %s\n", wr.wallet)
			for _, t := range wr.tokens {
				if t.err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "  %s: error: %v\n", t.name, t.err)
					continue
				}
				fmt.Printf("  %-6s %s\n", t.name+":", t.bal)
			}
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
		accountFiles, err := readAccountFiles()
		if err != nil {
			lg.Fatal(ctx, "Could not read account files", logger.Err(err))
		}
		if len(accountFiles) == 0 {
			lg.Fatal(ctx, "No account files found in data/account/. Run --capture first.")
		}

		wallets := config.LoadWallets(accountFiles)

		if (cfg.Telegram.BotToken == "") != (cfg.Telegram.ChatID == "") {
			lg.Warn(ctx, "Only one of TELEGRAM_BOT_TOKEN / TELEGRAM_CHAT_ID is set — Telegram disabled.")
		}

		entries := make([]app.AccountEntry, len(accountFiles))
		for i, authFile := range accountFiles {
			entries[i] = app.AccountEntry{
				Name:    wallets[i].Name,
				Claimer: browser.New(pw, lg, authFile),
				Wallet:  wallets[i].Address,
				Configs: buildConfigs(cfg, wallets[i].Address),
			}
		}

		notifier := telegram.New(cfg.Telegram.BotToken, cfg.Telegram.ChatID, lg)
		balanceFetcher := rpc.New(cfg.RPC.SepoliaRPCURL)
		uc := usecase.New(lg, balanceFetcher)
		a := app.New(entries, notifier, uc, lg)
		a.Run(ctx)

	default:
		flag.Usage()
		os.Exit(0)
	}
}

func buildConfigs(cfg *config.Config, walletAddress string) []domain.ClaimConfig {
	all := []domain.ClaimConfig{
		{
			FaucetURL:     faucetURLETH,
			WalletAddress: walletAddress,
			ButtonText:    "Get 0.05 Sepolia ETH",
			TokenName:     "ETH",
			TokenDecimals: 18,
		},
		{
			FaucetURL:     faucetURLPYUSD,
			WalletAddress: walletAddress,
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

func readAccountFiles() ([]string, error) {
	entries, err := os.ReadDir("data/account")
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			files = append(files, filepath.Join("data/account", e.Name()))
		}
	}
	return files, nil
}
