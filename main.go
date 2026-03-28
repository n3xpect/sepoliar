package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/playwright-community/playwright-go"

	"sepoliar/internal/browser"
	"sepoliar/internal/model"
	"sepoliar/internal/rpc"
	"sepoliar/internal/service"
	"sepoliar/internal/telegram"
	"sepoliar/pkg/config"
	"sepoliar/pkg/crypto"
	"sepoliar/pkg/logger"
)

type cmd struct {
	cfg *config.Config
	log logger.Logger
	ctx context.Context
	key [32]byte
}

func main() {
	googleSignIn := flag.Bool("google-sign-in", false, "")
	flag.BoolVar(googleSignIn, "g", false, "")
	claim := flag.Bool("claim", false, "")
	flag.BoolVar(claim, "c", false, "")
	balance := flag.Bool("balance", false, "")
	flag.BoolVar(balance, "b", false, "")
	flag.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, "Usage: sepoliar [--google-sign-in | --claim | --balance]\n\n")
		_, _ = fmt.Fprintf(os.Stderr, "  --google-sign-in, -g  Opens a browser for Google sign-in and saves the encrypted session\n")
		_, _ = fmt.Fprintf(os.Stderr, "  --claim,          -c  Starts the faucet claim loop using saved sessions\n")
		_, _ = fmt.Fprintf(os.Stderr, "  --balance,        -b  Prints current wallet balances and exits\n")
		_, _ = fmt.Fprintf(os.Stderr, "  --help,           -h  Show this help message\n")
	}

	flag.Parse()

	if len(os.Args) == 1 {
		flag.Usage()
		os.Exit(0)
	}

	cfg := config.Load()
	c := &cmd{cfg: cfg, log: logger.NewLog(cfg.LogLevel), ctx: context.Background()}

	switch {
	case *balance:
		c.balance()
	case *googleSignIn:
		c.key = promptKey()
		c.googleSignIn()
	case *claim:
		c.claim()
	default:
		flag.Usage()
		os.Exit(0)
	}
}

func promptKey() [32]byte {
	if val := os.Getenv("SEPOLIAR_ENCRYPTION_KEY"); val != "" {
		return crypto.DeriveKey(val)
	}
	fi, err := os.Stdin.Stat()
	if err != nil || (fi.Mode()&os.ModeCharDevice) == 0 {
		_, _ = fmt.Fprintln(os.Stderr, "FATAL: SEPOLIAR_ENCRYPTION_KEY is required in non-interactive mode")
		os.Exit(1)
	}
	_, _ = fmt.Fprint(os.Stderr, "Encryption key: ")
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	passphrase := strings.TrimSpace(line)
	if passphrase == "" {
		_, _ = fmt.Fprintln(os.Stderr, "Error: encryption key cannot be empty")
		os.Exit(1)
	}
	return crypto.DeriveKey(passphrase)
}

func (c *cmd) validateSessions(files []string) {
	var encFiles []string
	for _, f := range files {
		if strings.HasSuffix(f, ".enc") {
			encFiles = append(encFiles, f)
		}
	}
	if len(encFiles) == 0 {
		c.log.Fatal(c.ctx, "No encrypted session files found. Run --google-sign-in first.")
	}
	for _, f := range encFiles {
		raw, err := os.ReadFile(f)
		if err != nil {
			c.log.Fatal(c.ctx, "Could not read session file", logger.String("file", f), logger.Err(err))
		}
		if _, err = crypto.Decrypt(raw, c.key); err != nil {
			c.log.Fatal(c.ctx, "Could not decrypt session file — wrong key?", logger.String("file", f), logger.Err(err))
		}
	}
}
func (c *cmd) balance() {
	fetcher := rpc.New(c.cfg.RPC.SepoliaRPCURL, c.cfg.RPC.EtherscanAPIKey)

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
			go func(j int, cfg model.ClaimConfig) {
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
func (c *cmd) googleSignIn() {
	pw, err := playwright.Run()
	if err != nil {
		c.log.Fatal(c.ctx, "Could not start playwright", logger.Err(err))
	}
	defer func() { _ = pw.Stop() }()

	capturer := browser.NewSessionCapturer(pw, c.log, c.key)
	if err := capturer.CaptureSession(c.ctx); err != nil {
		c.log.Fatal(c.ctx, "Could not capture session", logger.Err(err))
	}
}
func (c *cmd) claim() {
	accountFiles, err := readAccountFiles()
	if err != nil {
		c.log.Fatal(c.ctx, "Could not read account files", logger.Err(err))
	}
	if len(accountFiles) == 0 {
		c.log.Fatal(c.ctx, "No account files found in data/account/. Run --google-sign-in first.")
	}
	c.key = promptKey()
	c.validateSessions(accountFiles)

	pw, err := playwright.Run()
	if err != nil {
		c.log.Fatal(c.ctx, "Could not start playwright", logger.Err(err))
	}
	defer func() { _ = pw.Stop() }()

	wallets := config.LoadWallets(accountFiles)

	if (c.cfg.Telegram.BotToken == "") != (c.cfg.Telegram.ChatID == "") {
		c.log.Warn(c.ctx, "Only one of TELEGRAM_BOT_TOKEN / TELEGRAM_CHAT_ID is set — Telegram disabled.")
	}

	entries := make([]service.AccountEntry, len(accountFiles))
	for i, authFile := range accountFiles {
		entries[i] = service.AccountEntry{
			Name:    wallets[i].Name,
			Claimer: browser.New(pw, c.log, authFile, c.key),
			Wallet:  wallets[i].Address,
			Configs: c.buildConfigs(wallets[i].Address),
		}
	}

	notifier := telegram.New(c.cfg.Telegram.BotToken, c.cfg.Telegram.ChatID, c.log)
	balanceFetcher := rpc.New(c.cfg.RPC.SepoliaRPCURL, c.cfg.RPC.EtherscanAPIKey)
	svc := service.New(entries, notifier, balanceFetcher, c.log)
	svc.Run(c.ctx)
}
func (c *cmd) buildConfigs(walletAddress string) []model.ClaimConfig {
	all := []model.ClaimConfig{
		{
			FaucetURL:     c.cfg.FaucetURLETH,
			WalletAddress: walletAddress,
			TokenName:     "ETH",
			TokenDecimals: 18,
		},
		{
			FaucetURL:     c.cfg.FaucetURLPYUSD,
			WalletAddress: walletAddress,
			TokenName:     "PYUSD",
			TokenAddress:  c.cfg.PyUSDContractAddress,
			TokenDecimals: 6,
		},
	}

	return all
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
		if !e.IsDir() && (strings.HasSuffix(e.Name(), ".json") || strings.HasSuffix(e.Name(), ".enc")) {
			files = append(files, filepath.Join("data/account", e.Name()))
		}
	}
	return files, nil
}
