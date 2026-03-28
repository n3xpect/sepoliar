package model

import "time"

type ClaimConfig struct {
	FaucetURL     string
	WalletAddress string
	TokenName     string
	TokenAddress  string
	TokenDecimals int
}

type ClaimResult struct {
	TokenName     string
	Message       string
	RetryAt       *time.Time
	Err           error
	BalanceBefore string
	BalanceAfter  string
}
