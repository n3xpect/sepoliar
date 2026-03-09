package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

type WalletEntry struct {
	Name    string
	Address string
}

func LoadWallets(accountFiles []string) []WalletEntry {
	raw := getEnvRequired("WALLET_ADDRESSES")
	parts := strings.Split(raw, ",")
	if len(parts) != len(accountFiles) {
		panic(fmt.Sprintf(
			"account count (%d) does not match wallet count (%d)",
			len(accountFiles), len(parts),
		))
	}
	entries := make([]WalletEntry, len(parts))
	for i, p := range parts {
		base := strings.TrimSuffix(filepath.Base(accountFiles[i]), ".json")
		if idx := strings.Index(base, "_"); idx >= 0 {
			base = base[idx+1:]
		}
		entries[i] = WalletEntry{
			Name:    base,
			Address: strings.TrimSpace(p),
		}
	}
	return entries
}

func loadWallets() []WalletEntry {
	raw := getEnvRequired("WALLET_ADDRESSES")
	parts := strings.Split(raw, ",")
	entries := make([]WalletEntry, len(parts))
	for i, p := range parts {
		entries[i] = WalletEntry{Address: strings.TrimSpace(p)}
	}
	return entries
}
