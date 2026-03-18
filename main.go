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

type cmd struct {
	cfg *config.Config
	lg  logger.Logger
	ctx context.Context
}

func main() {
	capture := flag.Bool("capture", false, "")
	start := flag.Bool("start", false, "")
	balance := flag.Bool("balance", false, "")

	flag.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, "Usage: sepoliar [--capture | --start | --balance]\n\n")
		_, _ = fmt.Fprintf(os.Stderr, "  --capture     Opens a browser for Google sign-in and saves the session (email auto-detected)\n")
		_, _ = fmt.Fprintf(os.Stderr, "  --start       Starts the faucet claim loop using saved sessions\n")
		_, _ = fmt.Fprintf(os.Stderr, "  --balance     Prints current wallet balances and exits\n")
		_, _ = fmt.Fprintf(os.Stderr, "  --help        Show this help message\n")
	}

	flag.Parse()

	if len(os.Args) == 1 {
		flag.Usage()
		os.Exit(0)
	}

	cfg := config.Load()
	c := &cmd{cfg: cfg, lg: logger.NewLog(cfg.LogLevel), ctx: context.Background()}

	switch {
	case *balance:
		c.balance()
	case *capture:
		c.capture()
	case *start:
		c.claim()
	default:
		flag.Usage()
		os.Exit(0)
	}
}

func (c *cmd) balance() {
	fetcher := rpc.New(c.cfg.RPC.SepoliaRPCURL)

	type tokenResult struct {
		name string
		bal  string
		err  error
	}
	type walletResult struct {
		wallet string
		tokens []tokenResult
	}

	results := make([]walletResult, len(c.cfg.Wallets))
	var wg sync.WaitGroup

	for i, w := range c.cfg.Wallets {
		configs := c.buildConfigs(w.Address)
		tokens := make([]tokenResult, len(configs))
		wg.Add(len(configs))
		for j, cfg := range configs {
			go func(j int, cfg domain.ClaimConfig) {
				defer wg.Done()
				bal, err := fetcher.GetBalance(c.ctx, cfg)
				tokens[j] = tokenResult{name: cfg.TokenName, bal: bal, err: err}
			}(j, cfg)
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
}
func (c *cmd) capture() {
	pw, err := playwright.Run()
	if err != nil {
		c.lg.Fatal(c.ctx, "Could not start playwright", logger.Err(err))
	}
	defer func() { _ = pw.Stop() }()

	capturer := browser.NewSessionCapturer(pw, c.lg)
	if err := capturer.CaptureSession(c.ctx); err != nil {
		c.lg.Fatal(c.ctx, "Could not capture session", logger.Err(err))
	}
}
func (c *cmd) claim() {
	accountFiles, err := readAccountFiles()
	if err != nil {
		c.lg.Fatal(c.ctx, "Could not read account files", logger.Err(err))
	}
	if len(accountFiles) == 0 {
		c.lg.Fatal(c.ctx, "No account files found in data/account/. Run --capture first.")
	}

	pw, err := playwright.Run()
	if err != nil {
		c.lg.Fatal(c.ctx, "Could not start playwright", logger.Err(err))
	}
	defer func() { _ = pw.Stop() }()

	wallets := config.LoadWallets(accountFiles)

	if (c.cfg.Telegram.BotToken == "") != (c.cfg.Telegram.ChatID == "") {
		c.lg.Warn(c.ctx, "Only one of TELEGRAM_BOT_TOKEN / TELEGRAM_CHAT_ID is set — Telegram disabled.")
	}

	entries := make([]app.AccountEntry, len(accountFiles))
	for i, authFile := range accountFiles {
		entries[i] = app.AccountEntry{
			Name:    wallets[i].Name,
			Claimer: browser.New(pw, c.lg, authFile),
			Wallet:  wallets[i].Address,
			Configs: c.buildConfigs(wallets[i].Address),
		}
	}

	notifier := telegram.New(c.cfg.Telegram.BotToken, c.cfg.Telegram.ChatID, c.lg)
	balanceFetcher := rpc.New(c.cfg.RPC.SepoliaRPCURL)
	uc := usecase.New(c.lg, balanceFetcher)
	a := app.New(entries, notifier, uc, c.lg)
	a.Run(c.ctx)
}
func (c *cmd) buildConfigs(walletAddress string) []domain.ClaimConfig {
	all := []domain.ClaimConfig{
		{
			FaucetURL:     c.cfg.FaucetURLETH,
			WalletAddress: walletAddress,
			ButtonText:    "Get 0.05 Sepolia ETH",
			TokenName:     "ETH",
			TokenDecimals: 18,
		},
		{
			FaucetURL:     c.cfg.FaucetURLPYUSD,
			WalletAddress: walletAddress,
			ButtonText:    "Get 100 Sepolia PYUSD",
			TokenName:     "PYUSD",
			TokenAddress:  c.cfg.PyUSDContractAddress,
			TokenDecimals: 6,
		},
	}

	enabled := make(map[string]bool)
	for _, t := range strings.Split(c.cfg.EnabledTokens, ",") {
		enabled[strings.TrimSpace(strings.ToUpper(t))] = true
	}

	var result []domain.ClaimConfig
	for _, cfg := range all {
		if enabled[cfg.TokenName] {
			result = append(result, cfg)
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
