package app

import (
	"context"
	"sync"
	"time"

	"sepoliar/internal/domain"
	"sepoliar/internal/usecase"
	"sepoliar/pkg/logger"
)

type App struct {
	configs      []domain.ClaimConfig
	claimer      domain.FaucetClaimer
	notifier     domain.Notifier
	uc           *usecase.ClaimUseCase
	lg           logger.Logger
	mu           sync.RWMutex
	nextRunAt    time.Time
	activeTokens []string
}

func New(
	configs []domain.ClaimConfig,
	claimer domain.FaucetClaimer,
	notifier domain.Notifier,
	uc *usecase.ClaimUseCase,
	lg logger.Logger,
) *App {
	tokens := make([]string, 0, len(configs))
	for _, c := range configs {
		tokens = append(tokens, c.TokenName)
	}
	return &App{
		configs:      configs,
		claimer:      claimer,
		notifier:     notifier,
		uc:           uc,
		lg:           lg,
		activeTokens: tokens,
	}
}

func (a *App) Run(ctx context.Context) {
	if err := a.claimer.LoadSession(); err != nil {
		a.lg.Fatal(ctx, "Could not load session. Run with --capture first.", logger.Err(err))
	}
	defer func() { _ = a.claimer.Close() }()

	if a.notifier != nil {
		go a.notifier.StartPolling(ctx, a.getActiveTokens, a.getNextRun, func() string {
			return a.uc.FetchBalances(ctx, a.configs)
		})
		a.lg.Info(ctx, "Telegram notifications enabled.")
	} else {
		a.lg.Info(ctx, "Telegram not configured, running in console mode.")
	}

	for {
		results := a.uc.Execute(ctx, a.claimer, a.configs)
		next := a.uc.ComputeNext(results)
		a.setNextRun(next)
		msg := a.uc.FormatMessage(results, next)
		a.lg.Info(ctx, msg)
		if a.notifier != nil {
			_ = a.notifier.Send(ctx, msg)
		}
		a.lg.Info(ctx, "Next attempt", logger.String("at", next.Format("Mon, 02 Jan 2006 15:04:05")))
		time.Sleep(time.Until(next))
	}
}

func (a *App) setNextRun(t time.Time) {
	a.mu.Lock()
	a.nextRunAt = t
	a.mu.Unlock()
}

func (a *App) getNextRun() time.Time {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.nextRunAt
}

func (a *App) getActiveTokens() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.activeTokens
}
