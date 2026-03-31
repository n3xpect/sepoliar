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

func LoadWallets(accountFiles []string, walletAddresses string) []WalletEntry {
	parts := strings.Split(walletAddresses, ",")
	if len(parts) != len(accountFiles) {
		panic(fmt.Sprintf(
			"account count (%d) does not match wallet count (%d)",
			len(accountFiles), len(parts),
		))
	}
	entries := make([]WalletEntry, len(parts))
	for i, p := range parts {
		base := strings.TrimSuffix(strings.TrimSuffix(filepath.Base(accountFiles[i]), ".enc"), ".json")
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

func loadWallets(walletAddresses string) []WalletEntry {
	parts := strings.Split(walletAddresses, ",")
	entries := make([]WalletEntry, len(parts))
	for i, p := range parts {
		entries[i] = WalletEntry{Address: strings.TrimSpace(p)}
	}
	return entries
}
