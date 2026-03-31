package config

type TelegramConfig struct {
	BotToken string
	ChatID   string
}

func loadTelegramConfig(botToken, chatID string) TelegramConfig {
	return TelegramConfig{
		BotToken: botToken,
		ChatID:   chatID,
	}
}
