package domain

import (
	"context"
	"time"
)

type FaucetClaimer interface {
	SessionExists() bool
	CaptureSession(ctx context.Context) error
	LoadSession() error
	Claim(ctx context.Context, cfg ClaimConfig) ClaimResult
	Close() error
}

type Notifier interface {
	Send(ctx context.Context, msg string) error
	StartPolling(ctx context.Context, activeTokens func() []string, nextRun func() time.Time, getBalances func() string)
}

type BalanceFetcher interface {
	GetBalance(ctx context.Context, cfg ClaimConfig) (string, error)
}
