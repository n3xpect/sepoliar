package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"sepoliar/internal/browser"
	"sepoliar/internal/model"
	"sepoliar/internal/rpc"
	"sepoliar/internal/telegram"
	"sepoliar/pkg/logger"
)

const interval = 24*time.Hour + time.Minute

type AccountEntry struct {
	Name    string
	Claimer *browser.PlaywrightFaucetClaimer
	Wallet  string
	Configs []model.ClaimConfig
}

type accountResult struct {
	Name    string
	Wallet  string
	Results []model.ClaimResult
}

type Service struct {
	accounts     []AccountEntry
	notifier     *telegram.Notifier
	fetcher      *rpc.BalanceFetcher
	lg           logger.Logger
	mu           sync.RWMutex
	nextRunAt    time.Time
	activeTokens []string
}

func New(
	accounts []AccountEntry,
	notifier *telegram.Notifier,
	fetcher *rpc.BalanceFetcher,
	lg logger.Logger,
) *Service {
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
	return &Service{
		accounts:     accounts,
		notifier:     notifier,
		fetcher:      fetcher,
		lg:           lg,
		activeTokens: activeTokens,
	}
}

func (s *Service) Run(ctx context.Context) {
	for _, acc := range s.accounts {
		if err := acc.Claimer.LoadSession(); err != nil {
			s.lg.Fatal(ctx, "Could not load session — run --capture first.",
				logger.String("account", acc.Name), logger.Err(err))
		}
	}
	for _, acc := range s.accounts {
		acc := acc
		defer func() { _ = acc.Claimer.Close() }()
	}

	if s.notifier != nil {
		go s.notifier.StartPolling(ctx, s.getActiveTokens, s.getNextRun, func() string {
			var sb strings.Builder
			for i, acc := range s.accounts {
				if i > 0 {
					sb.WriteString("\n")
				}
				sb.WriteString(s.fetchBalancesForConfigs(ctx, acc.Name, acc.Wallet, acc.Configs))
			}
			return sb.String()
		})
		s.lg.Info(ctx, "Telegram notifications enabled.")
	} else {
		s.lg.Info(ctx, "Telegram not configured, running in console mode.")
	}

	for {
		var allResults []accountResult
		for _, acc := range s.accounts {
			results := s.execute(ctx, acc.Claimer, acc.Configs)
			allResults = append(allResults, accountResult{
				Name:    acc.Name,
				Wallet:  acc.Wallet,
				Results: results,
			})
		}

		next := s.computeNext(allResults)
		s.setNextRun(next)
		msg := s.formatCombinedMessage(allResults, next)
		s.lg.Info(ctx, msg)
		if s.notifier != nil {
			_ = s.notifier.Send(ctx, msg)
		}
		s.lg.Info(ctx, "Next attempt", logger.String("at", next.Format("02.01.2006 - 15:04:05")))
		time.Sleep(time.Until(next))
	}
}
func (s *Service) execute(ctx context.Context, claimer *browser.PlaywrightFaucetClaimer, configs []model.ClaimConfig) []model.ClaimResult {
	results := make([]model.ClaimResult, 0, len(configs))
	for _, cfg := range configs {
		var balBefore string
		if s.fetcher != nil {
			if bal, err := s.fetcher.GetBalance(ctx, cfg); err == nil {
				balBefore = bal
				s.lg.Info(ctx, "Balance before claim", logger.String("token", cfg.TokenName), logger.String("balance", bal))
			} else {
				s.lg.Warn(ctx, "Balance fetch failed (before)", logger.String("token", cfg.TokenName), logger.Err(err))
			}
		}
		result := claimer.Claim(ctx, cfg)
		if s.fetcher != nil {
			if bal, err := s.fetcher.GetBalance(ctx, cfg); err == nil {
				result.BalanceAfter = bal
				s.lg.Info(ctx, "Balance after claim", logger.String("token", cfg.TokenName), logger.String("balance", bal))
			} else {
				s.lg.Warn(ctx, "Balance fetch failed (after)", logger.String("token", cfg.TokenName), logger.Err(err))
			}
			result.BalanceBefore = balBefore
		}
		results = append(results, result)
	}
	return results
}
func (s *Service) fetchBalancesForConfigs(ctx context.Context, name, wallet string, configs []model.ClaimConfig) string {
	if s.fetcher == nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s\n%s\n", name, wallet))
	for _, cfg := range configs {
		bal, err := s.fetcher.GetBalance(ctx, cfg)
		if err != nil {
			sb.WriteString(fmt.Sprintf("  %-6s error: %v\n", cfg.TokenName+":", err))
		} else {
			sb.WriteString(fmt.Sprintf("  %-6s %s\n", cfg.TokenName+":", bal))
		}
	}
	return sb.String()
}
func (s *Service) computeNext(allResults []accountResult) time.Time {
	next := time.Now().Add(interval)
	for _, acc := range allResults {
		for _, r := range acc.Results {
			if r.RetryAt != nil {
				if candidate := r.RetryAt.Add(interval); candidate.After(next) {
					next = candidate
				}
			}
		}
	}
	return next
}
func (s *Service) formatCombinedMessage(accounts []accountResult, next time.Time) string {
	var sb strings.Builder
	for i, acc := range accounts {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("%s\n%s\n", acc.Name, acc.Wallet))
		for _, r := range acc.Results {
			if r.Err != nil {
				sb.WriteString(fmt.Sprintf("  %s: ❌ %v\n", r.TokenName, r.Err))
			} else {
				sb.WriteString(fmt.Sprintf("  %s: %s\n", r.TokenName, r.Message))
			}
			if r.BalanceBefore != "" || r.BalanceAfter != "" {
				sb.WriteString(fmt.Sprintf("    💰 Before: %s → After: %s\n", r.BalanceBefore, r.BalanceAfter))
			}
		}
	}
	sb.WriteString(fmt.Sprintf("\n⏰ Next run: %s", next.Format("02.01.2006 - 15:04:05")))
	return sb.String()
}
func (s *Service) setNextRun(t time.Time) {
	s.mu.Lock()
	s.nextRunAt = t
	s.mu.Unlock()
}
func (s *Service) getNextRun() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.nextRunAt
}
func (s *Service) getActiveTokens() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.activeTokens
}
