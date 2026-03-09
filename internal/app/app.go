package app

import (
	"context"
	"strings"
	"sync"
	"time"

	"sepoliar/internal/domain"
	"sepoliar/internal/usecase"
	"sepoliar/pkg/logger"
)

type AccountEntry struct {
	Name    string
	Claimer domain.FaucetClaimer
	Wallet  string
	Configs []domain.ClaimConfig
}

type App struct {
	accounts     []AccountEntry
	notifier     domain.Notifier
	uc           *usecase.ClaimUseCase
	lg           logger.Logger
	mu           sync.RWMutex
	nextRunAt    time.Time
	activeTokens []string
}

func New(
	accounts []AccountEntry,
	notifier domain.Notifier,
	uc *usecase.ClaimUseCase,
	lg logger.Logger,
) *App {
	tokens := make(map[string]bool)
	for _, acc := range accounts {
		for _, c := range acc.Configs {
			tokens[c.TokenName] = true
		}
	}
	activeTokens := make([]string, 0, len(tokens))
	for t := range tokens {
		activeTokens = append(activeTokens, t)
	}
	return &App{
		accounts:     accounts,
		notifier:     notifier,
		uc:           uc,
		lg:           lg,
		activeTokens: activeTokens,
	}
}

func (a *App) Run(ctx context.Context) {
	for _, acc := range a.accounts {
		if err := acc.Claimer.LoadSession(); err != nil {
			a.lg.Fatal(ctx, "Could not load session — run --capture first.",
				logger.String("account", acc.Name), logger.Err(err))
		}
	}
	for _, acc := range a.accounts {
		acc := acc
		defer func() { _ = acc.Claimer.Close() }()
	}

	if a.notifier != nil {
		go a.notifier.StartPolling(ctx, a.getActiveTokens, a.getNextRun, func() string {
			var sb strings.Builder
			for i, acc := range a.accounts {
				if i > 0 {
					sb.WriteString("\n")
				}
				sb.WriteString(a.uc.FetchBalancesForConfigs(ctx, acc.Name, acc.Wallet, acc.Configs))
			}
			return sb.String()
		})
		a.lg.Info(ctx, "Telegram notifications enabled.")
	} else {
		a.lg.Info(ctx, "Telegram not configured, running in console mode.")
	}

	for {
		var allResults []usecase.AccountResult
		for _, acc := range a.accounts {
			results := a.uc.Execute(ctx, acc.Claimer, acc.Configs)
			allResults = append(allResults, usecase.AccountResult{
				Name:    acc.Name,
				Wallet:  acc.Wallet,
				Results: results,
			})
		}

		next := a.uc.ComputeNext(allResults)
		a.setNextRun(next)
		msg := a.uc.FormatCombinedMessage(allResults, next)
		a.lg.Info(ctx, msg)
		if a.notifier != nil {
			_ = a.notifier.Send(ctx, msg)
		}
		a.lg.Info(ctx, "Next attempt", logger.String("at", next.Format("02.01.2006 - 15:04:05")))
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
