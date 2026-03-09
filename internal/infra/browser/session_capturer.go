package browser

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/playwright-community/playwright-go"

	"sepoliar/pkg/logger"
)

type PlaywrightSessionCapturer struct {
	pw *playwright.Playwright
	lg logger.Logger
}

func NewSessionCapturer(pw *playwright.Playwright, lg logger.Logger) *PlaywrightSessionCapturer {
	return &PlaywrightSessionCapturer{pw: pw, lg: lg}
}

func (s *PlaywrightSessionCapturer) CaptureSession(ctx context.Context) error {
	s.lg.Info(ctx, "Launching browser in interactive mode for session capture")

	b, err := s.pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false),
		Args:     []string{"--no-sandbox", "--disable-blink-features=AutomationControlled"},
	})
	if err != nil {
		return fmt.Errorf("could not launch browser: %w", err)
	}
	defer b.Close()

	bCtx, err := b.NewContext(playwright.BrowserNewContextOptions{
		Viewport:  &playwright.Size{Width: 1280, Height: 900},
		UserAgent: playwright.String("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	})
	if err != nil {
		return fmt.Errorf("could not create context: %w", err)
	}
	defer bCtx.Close()

	page, err := bCtx.NewPage()
	if err != nil {
		return fmt.Errorf("could not create page: %w", err)
	}

	if _, err = page.Goto("https://accounts.google.com", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad,
	}); err != nil {
		return fmt.Errorf("could not navigate to Google sign-in: %w", err)
	}

	s.lg.Info(ctx, "Google sign-in page opened. Sign in, then press Enter in the terminal...")
	fmt.Scanln() //nolint:errcheck

	if _, err = page.Goto("https://cloud.google.com/application/web3/faucet/ethereum/sepolia", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad,
	}); err != nil {
		return fmt.Errorf("could not navigate to faucet: %w", err)
	}

	email := s.detectEmail(ctx, page, bCtx)

	state, err := bCtx.StorageState()
	if err != nil {
		return fmt.Errorf("could not get storage state: %w", err)
	}

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("could not marshal storage state: %w", err)
	}

	if err = os.MkdirAll("data/account", 0700); err != nil {
		return fmt.Errorf("could not create data/account directory: %w", err)
	}

	index := countExistingAccountFiles()
	filePath := fmt.Sprintf("data/account/%d_%s.json", index, email)

	if err = os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("could not write session file: %w", err)
	}

	s.lg.Info(ctx, fmt.Sprintf("Session captured successfully → %s. Run with --no-capture to start claiming.", filePath))
	return nil
}

func (s *PlaywrightSessionCapturer) detectEmail(ctx context.Context, page playwright.Page, bCtx playwright.BrowserContext) string {
	email := evalEmail(page)
	if email != "" {
		s.lg.Info(ctx, fmt.Sprintf("Detected account email: %s", email))
		return email
	}

	if newPage, err := bCtx.NewPage(); err == nil {
		if _, err = newPage.Goto("https://myaccount.google.com", playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateLoad,
		}); err == nil {
			email = evalEmail(newPage)
		}
		_ = newPage.Close()
	}
	if email != "" {
		s.lg.Info(ctx, fmt.Sprintf("Detected account email: %s", email))
		return email
	}

	s.lg.Warn(ctx, "Could not auto-detect email. Please type your Google account email:")
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	email = strings.TrimSpace(line)
	return email
}

func evalEmail(page playwright.Page) string {
	result, err := page.Evaluate(`(async () => {
		try {
			const r = await fetch(
				'https://www.googleapis.com/oauth2/v1/userinfo?alt=json',
				{ credentials: 'include' }
			);
			if (r.ok) {
				const d = await r.json();
				if (d.email) return d.email;
			}
		} catch (_) {}
		const m = document.body.innerText.match(/[\w.+\-]+@[\w.]+\.[a-z]{2,}/i);
		return m ? m[0] : null;
	})()`)
	if err != nil || result == nil {
		return ""
	}
	v := strings.TrimSpace(fmt.Sprintf("%v", result))
	if strings.Contains(v, "@") {
		return v
	}
	return ""
}

func countExistingAccountFiles() int {
	entries, err := os.ReadDir("data/account")
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			count++
		}
	}
	return count
}
