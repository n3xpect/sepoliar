package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	LogLevel      string
	EnabledTokens string
	Telegram      TelegramConfig
	Wallets       []WalletEntry
	RPC           RPCConfig
}

func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		LogLevel:      getEnv("LOG_LEVEL", "info"),
		EnabledTokens: getEnv("ENABLED_TOKENS", "ETH"),
		Telegram:      loadTelegramConfig(),
		Wallets:       loadWallets(),
		RPC:           loadRPCConfig(),
	}
}

func getEnvRequired(key string) string {
	value := os.Getenv(key)
	if value == "" {
		_, _ = fmt.Fprintf(os.Stderr, "FATAL: Required environment variable %s is not set\n", key)
		os.Exit(1)
	}
	return value
}

func getEnv(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvIntRequired(key string) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return 0, fmt.Errorf("required environment variable %s is not set", key)
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("environment variable %s must be a valid integer, got: %s", key, value)
	}
	return intValue, nil
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}
