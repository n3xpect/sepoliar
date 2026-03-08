package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

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

func (s *PlaywrightSessionCapturer) SessionExists() bool {
	_, err := os.Stat(authStateFile)
	return !os.IsNotExist(err)
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

	// Google sign-in sayfasına git
	if _, err = page.Goto("https://accounts.google.com", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad,
	}); err != nil {
		return fmt.Errorf("could not navigate to Google sign-in: %w", err)
	}

	s.lg.Info(ctx, "Google sign-in page opened. Sign in, then press Enter in the terminal...")
	fmt.Scanln() //nolint:errcheck

	// Oturumu faucet sayfasında da geçerli kılmak için faucet'e git
	if _, err = page.Goto("https://cloud.google.com/application/web3/faucet/ethereum/sepolia", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad,
	}); err != nil {
		return fmt.Errorf("could not navigate to faucet: %w", err)
	}

	state, err := bCtx.StorageState()
	if err != nil {
		return fmt.Errorf("could not get storage state: %w", err)
	}

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("could not marshal storage state: %w", err)
	}

	if err = os.MkdirAll("data", 0700); err != nil {
		return fmt.Errorf("could not create data directory: %w", err)
	}

	if err = os.WriteFile(authStateFile, data, 0600); err != nil {
		return fmt.Errorf("could not write auth.json: %w", err)
	}

	s.lg.Info(ctx, "Session captured successfully. Run with --no-capture to start claiming.")
	return nil
}
