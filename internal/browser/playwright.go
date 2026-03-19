package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"

	"sepoliar/internal/model"
	"sepoliar/pkg/crypto"
	"sepoliar/pkg/logger"
)

type PlaywrightFaucetClaimer struct {
	pw            *playwright.Playwright
	browser       playwright.Browser
	storageState  playwright.OptionalStorageState
	lg            logger.Logger
	authStateFile string
	key           [32]byte
}

func New(pw *playwright.Playwright, lg logger.Logger, authStateFile string, key [32]byte) *PlaywrightFaucetClaimer {
	return &PlaywrightFaucetClaimer{pw: pw, lg: lg, authStateFile: authStateFile, key: key}
}

func (p *PlaywrightFaucetClaimer) LoadSession() error {
	raw, err := os.ReadFile(p.authStateFile)
	if err != nil {
		return fmt.Errorf("could not read auth file: %w", err)
	}
	stateData := raw
	if strings.HasSuffix(p.authStateFile, ".enc") {
		stateData, err = crypto.Decrypt(raw, p.key)
		if err != nil {
			return fmt.Errorf("could not decrypt auth file: %w", err)
		}
	}
	if err = json.Unmarshal(stateData, &p.storageState); err != nil {
		return fmt.Errorf("could not parse auth file: %w", err)
	}
	p.browser, err = p.pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
		Args:     []string{"--no-sandbox", "--disable-blink-features=AutomationControlled"},
	})
	if err != nil {
		return fmt.Errorf("could not launch browser: %w", err)
	}
	return nil
}
func (p *PlaywrightFaucetClaimer) Close() error {
	if p.browser != nil {
		return p.browser.Close()
	}
	return nil
}
func (p *PlaywrightFaucetClaimer) Claim(ctx context.Context, cfg model.ClaimConfig) model.ClaimResult {
	msg, retryAt, err := p.doClaim(ctx, cfg)
	return model.ClaimResult{
		TokenName: cfg.TokenName,
		Message:   msg,
		RetryAt:   retryAt,
		Err:       err,
	}
}
func (p *PlaywrightFaucetClaimer) doClaim(ctx context.Context, cfg model.ClaimConfig) (string, *time.Time, error) {
	bCtx, err := p.browser.NewContext(playwright.BrowserNewContextOptions{
		Viewport:     &playwright.Size{Width: 1280, Height: 900},
		StorageState: &p.storageState,
	})
	if err != nil {
		return "", nil, fmt.Errorf("could not create context: %w", err)
	}
	defer bCtx.Close()

	page, err := bCtx.NewPage()
	if err != nil {
		return "", nil, fmt.Errorf("could not create page: %w", err)
	}

	p.lg.Info(ctx, "Navigating to faucet", logger.String("token", cfg.TokenName))
	if _, err = page.Goto(cfg.FaucetURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad,
		Timeout:   playwright.Float(60000),
	}); err != nil {
		return "", nil, fmt.Errorf("could not navigate to faucet")
	}

	if strings.Contains(page.URL(), "accounts.google.com") {
		if removeErr := os.Remove(p.authStateFile); removeErr != nil {
			p.lg.Warn(ctx, "Could not delete auth file", logger.Err(removeErr))
		}
		p.lg.Fatal(ctx, "Session expired. Deleted auth file — run --capture again.")
	}

	time.Sleep(1 * time.Second)
	selectedOpt, _ := page.QuerySelector("[role='option'][aria-selected='true'], mat-option.mat-selected, li[aria-selected='true']")
	if selectedOpt != nil {
		_ = selectedOpt.Click()
		time.Sleep(500 * time.Millisecond)
	}

	p.lg.Info(ctx, "Waiting for wallet input", logger.String("token", cfg.TokenName))
	walletInput, err := page.WaitForSelector(
		"input:not([type='hidden']):not([type='submit']):not([type='button']):not([type='checkbox']):not([type='radio']):not([type='file'])",
		playwright.PageWaitForSelectorOptions{Timeout: playwright.Float(30000)},
	)
	if err != nil {
		return "", nil, fmt.Errorf("page load failed: wallet input not found")
	}

	if err = walletInput.Fill(cfg.WalletAddress); err != nil {
		return "", nil, fmt.Errorf("could not fill wallet address")
	}
	p.lg.Info(ctx, "Wallet address filled", logger.String("token", cfg.TokenName), logger.String("address", cfg.WalletAddress))

	time.Sleep(1 * time.Second)

	earlyWarning, _ := page.Evaluate(`() => document.body.innerText.match(/Try again after/i)?.[0] ? (() => {
		const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
		let node;
		while (node = walker.nextNode()) {
			if (/Try again after/i.test(node.textContent)) {
				return node.parentElement?.closest('div, p, span, section')?.innerText?.trim() || node.textContent.trim();
			}
		}
		return null;
	})() : null`)
	if earlyWarning != nil {
		rateLimitTime, parseErr := parseRateLimitTime(fmt.Sprintf("%v", earlyWarning))
		if parseErr != nil {
			p.lg.Error(ctx, "Rate limit time parse error", logger.String("token", cfg.TokenName), logger.Err(parseErr))
			return "⏳ Rate limit active, skipping this round", nil, nil
		}
		retryAt := rateLimitTime.Add(time.Minute)
		p.lg.Warn(ctx, "Rate limit active", logger.String("token", cfg.TokenName), logger.String("retryAt", retryAt.Format("Mon, 02 Jan 2006 15:04:05")))
		return fmt.Sprintf("⏳ Rate limit. Retry at: %s", retryAt.Format("Mon, 02 Jan 2006 15:04:05")), &rateLimitTime, nil
	}

	p.lg.Info(ctx, "Clicking claim button", logger.String("token", cfg.TokenName))
	claimButton, err := page.WaitForSelector(
		fmt.Sprintf("button:has-text('%s')", cfg.ButtonText),
		playwright.PageWaitForSelectorOptions{Timeout: playwright.Float(15000)},
	)
	if err != nil {
		return "", nil, fmt.Errorf("claim button not found")
	}

	if err = claimButton.Click(); err != nil {
		return "", nil, fmt.Errorf("could not click claim button")
	}
	p.lg.Info(ctx, "Claim button clicked, waiting 20s for result", logger.String("token", cfg.TokenName))

	time.Sleep(20 * time.Second)

	bodyText, evalErr := page.Evaluate("() => document.body.innerText")
	if evalErr != nil {
		return "⚠️ Claim submitted, result unknown", nil, nil
	}
	text := strings.ToLower(fmt.Sprintf("%v", bodyText))

	if strings.Contains(text, "transaction complete") {
		return "✅ Transaction complete! Check your wallet address", nil, nil
	}
	if strings.Contains(text, "try again after") {
		rateLimitTime, parseErr := parseRateLimitTime(fmt.Sprintf("%v", bodyText))
		if parseErr != nil {
			p.lg.Error(ctx, "Rate limit time parse error (post-click)", logger.String("token", cfg.TokenName), logger.Err(parseErr))
			return "⚠️ Check your wallet address (rate limited, time unknown)", nil, nil
		}
		return "⚠️ Check your wallet address", &rateLimitTime, nil
	}
	return "⚠️ Claim submitted, result unknown", nil, nil
}

func parseRateLimitTime(text string) (time.Time, error) {
	re := regexp.MustCompile(`Try again after (.+?)\.`)
	m := re.FindStringSubmatch(text)
	if len(m) < 2 {
		return time.Time{}, fmt.Errorf("rate limit time not found in: %s", text)
	}
	dateStr := strings.TrimSpace(m[1])
	dateStr = strings.ReplaceAll(dateStr, "\u202f", " ") // narrow no-break space
	dateStr = strings.ReplaceAll(dateStr, "\u00a0", " ") // non-breaking space
	return time.ParseInLocation("Jan 2, 2006, 3:04:05 PM", dateStr, time.Local)
}
