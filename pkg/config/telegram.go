package config

type TelegramConfig struct {
	BotToken string
	ChatID   string
}

func loadTelegramConfig() TelegramConfig {
	return TelegramConfig{
		BotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		ChatID:   getEnv("TELEGRAM_CHAT_ID", ""),
	}
}
