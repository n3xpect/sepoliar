package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"sepoliar/internal/domain"
	"sepoliar/pkg/logger"
)

const interval = 24*time.Hour + time.Minute

type ClaimUseCase struct {
	lg      logger.Logger
	fetcher domain.BalanceFetcher
}

func New(lg logger.Logger, fetcher domain.BalanceFetcher) *ClaimUseCase {
	return &ClaimUseCase{lg: lg, fetcher: fetcher}
}

func (uc *ClaimUseCase) Execute(ctx context.Context, claimer domain.FaucetClaimer, configs []domain.ClaimConfig) []domain.ClaimResult {
	results := make([]domain.ClaimResult, 0, len(configs))
	for _, cfg := range configs {
		var balBefore string
		if uc.fetcher != nil {
			if bal, err := uc.fetcher.GetBalance(ctx, cfg); err == nil {
				balBefore = bal
				uc.lg.Info(ctx, "Balance before claim", logger.String("token", cfg.TokenName), logger.String("balance", bal))
			} else {
				uc.lg.Warn(ctx, "Balance fetch failed (before)", logger.String("token", cfg.TokenName), logger.Err(err))
			}
		}
		result := claimer.Claim(ctx, cfg)
		if uc.fetcher != nil {
			if bal, err := uc.fetcher.GetBalance(ctx, cfg); err == nil {
				result.BalanceAfter = bal
				uc.lg.Info(ctx, "Balance after claim", logger.String("token", cfg.TokenName), logger.String("balance", bal))
			} else {
				uc.lg.Warn(ctx, "Balance fetch failed (after)", logger.String("token", cfg.TokenName), logger.Err(err))
			}
			result.BalanceBefore = balBefore
		}
		results = append(results, result)
	}
	return results
}

func (uc *ClaimUseCase) FetchBalances(ctx context.Context, configs []domain.ClaimConfig) string {
	if uc.fetcher == nil {
		return "Balance fetcher not configured."
	}
	var sb strings.Builder
	for _, cfg := range configs {
		bal, err := uc.fetcher.GetBalance(ctx, cfg)
		if err != nil {
			sb.WriteString(fmt.Sprintf("%s: error: %v\n", cfg.TokenName, err))
		} else {
			sb.WriteString(fmt.Sprintf("%-6s %s\n", cfg.TokenName+":", bal))
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}

func (uc *ClaimUseCase) ComputeNext(results []domain.ClaimResult) time.Time {
	next := time.Now().Add(interval)
	for _, r := range results {
		if r.RetryAt != nil {
			if candidate := r.RetryAt.Add(interval); candidate.After(next) {
				next = candidate
			}
		}
	}
	return next
}

func (uc *ClaimUseCase) FormatMessage(results []domain.ClaimResult, next time.Time) string {
	var sb strings.Builder
	for i, r := range results {
		if i > 0 {
			sb.WriteString("\n")
		}
		if r.Err != nil {
			sb.WriteString(fmt.Sprintf("%s: ❌ %v\n", r.TokenName, r.Err))
		} else {
			sb.WriteString(fmt.Sprintf("%s: %s\n", r.TokenName, r.Message))
		}
		if r.BalanceBefore != "" || r.BalanceAfter != "" {
			sb.WriteString(fmt.Sprintf("  💰 Before: %s → After: %s\n", r.BalanceBefore, r.BalanceAfter))
		}
	}
	sb.WriteString(fmt.Sprintf("\n⏰ Next run: %s", next.Format("Mon, 02 Jan 2006 15:04:05")))
	return sb.String()
}
