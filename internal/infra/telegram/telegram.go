package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"sepoliar/internal/domain"
	"sepoliar/pkg/logger"
)

type Notifier struct {
	token  string
	chatID string
	log    logger.Logger
}

func New(token, chatID string, lg logger.Logger) domain.Notifier {
	if token == "" || chatID == "" {
		return nil
	}
	return &Notifier{token: token, chatID: chatID, log: lg}
}

func (n *Notifier) Send(ctx context.Context, msg string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", n.token)
	body, _ := json.Marshal(map[string]string{"chat_id": n.chatID, "text": msg})
	resp, err := http.Post(url, "application/json", bytes.NewReader(body)) //nolint:noctx
	if err != nil {
		n.log.Error(ctx, "Telegram send error", logger.Err(err))
		return err
	}
	_ = resp.Body.Close()
	return nil
}
func (n *Notifier) StartPolling(ctx context.Context, activeTokens func() []string, nextRun func() time.Time, getBalances func() string) {
	type tgChat struct {
		ID int64 `json:"id"`
	}
	type tgMessage struct {
		Chat tgChat `json:"chat"`
		Text string `json:"text"`
	}
	type tgUpdate struct {
		UpdateID int64     `json:"update_id"`
		Message  tgMessage `json:"message"`
	}
	type tgResponse struct {
		OK     bool       `json:"ok"`
		Result []tgUpdate `json:"result"`
	}

	var offset int64
	for {
		url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=30", n.token, offset)
		resp, err := http.Get(url) //nolint:noctx
		if err != nil {
			n.log.Error(ctx, "Telegram polling error", logger.Err(err))
			time.Sleep(5 * time.Second)
			continue
		}

		var result tgResponse
		if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
			_ = resp.Body.Close()
			n.log.Error(ctx, "Telegram polling decode error", logger.Err(err))
			time.Sleep(5 * time.Second)
			continue
		}
		_ = resp.Body.Close()

		for _, update := range result.Result {
			offset = update.UpdateID + 1
			chatIDStr := strconv.FormatInt(update.Message.Chat.ID, 10)
			if chatIDStr != n.chatID {
				n.sendMsg(ctx, chatIDStr, "Unauthorized user")
				continue
			}
			switch update.Message.Text {
			case "/start":
				var sb strings.Builder
				tokens := activeTokens()
				if len(tokens) == 0 {
					sb.WriteString("No wallet addresses configured.")
				} else {
					sb.WriteString("Active: " + strings.Join(tokens, ", ") + "\n")
					next := nextRun()
					if next.IsZero() {
						sb.WriteString("Next run time not yet determined.")
					} else {
						sb.WriteString(fmt.Sprintf("Next run: %s", next.Format("Mon, 02 Jan 2006 15:04:05")))
					}
				}
				n.sendMsg(ctx, chatIDStr, sb.String())
				case "/balance":
				n.sendMsg(ctx, chatIDStr, getBalances())
			default:
				n.sendMsg(ctx, chatIDStr, "Unknown command. Use /start or /balance.")
			}
		}
	}
}
func (n *Notifier) sendMsg(ctx context.Context, chatID, text string) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", n.token)
	body, _ := json.Marshal(map[string]string{"chat_id": chatID, "text": text})
	resp, err := http.Post(url, "application/json", bytes.NewReader(body)) //nolint:noctx
	if err != nil {
		n.log.Error(ctx, "Telegram send error", logger.Err(err))
		return
	}
	_ = resp.Body.Close()
}
