package domain

import "time"

type ClaimConfig struct {
	FaucetURL     string
	WalletAddress string
	ButtonText    string
	TokenName     string
	TokenAddress  string // boşsa native ETH, doluysa ERC-20 contract adresi
	TokenDecimals int    // ETH için 18, PYUSD için 6
}

type ClaimResult struct {
	TokenName     string
	Message       string
	RetryAt       *time.Time
	Err           error
	BalanceBefore string
	BalanceAfter  string
}
