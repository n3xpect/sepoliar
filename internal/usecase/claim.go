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

type AccountResult struct {
	Name    string
	Wallet  string
	Results []domain.ClaimResult
}

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

func (uc *ClaimUseCase) FetchBalancesForConfigs(ctx context.Context, name, wallet string, configs []domain.ClaimConfig) string {
	if uc.fetcher == nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s\n%s\n", name, wallet))
	for _, cfg := range configs {
		bal, err := uc.fetcher.GetBalance(ctx, cfg)
		if err != nil {
			sb.WriteString(fmt.Sprintf("  %-6s error: %v\n", cfg.TokenName+":", err))
		} else {
			sb.WriteString(fmt.Sprintf("  %-6s %s\n", cfg.TokenName+":", bal))
		}
	}
	return sb.String()
}

func (uc *ClaimUseCase) ComputeNext(allResults []AccountResult) time.Time {
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

func (uc *ClaimUseCase) FormatCombinedMessage(accounts []AccountResult, next time.Time) string {
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
