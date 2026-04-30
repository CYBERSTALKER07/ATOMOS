package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

// TelegramClient sends messages via the Telegram Bot API using raw HTTP.
// No external Go SDK dependency — pure stdlib, production-grade.
type TelegramClient struct {
	botToken string
	apiBase  string
}

type telegramMessage struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

type telegramResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
}

// NewTelegramClient constructs a client from the TELEGRAM_BOT_TOKEN env var.
// If the env var is not set, returns a no-op client that logs instead of crashing.
func NewTelegramClient() *TelegramClient {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Println("[TELEGRAM] TELEGRAM_BOT_TOKEN not set — Telegram fallback will log-only.")
	}
	return &TelegramClient{
		botToken: token,
		apiBase:  "https://api.telegram.org",
	}
}

// SendMessage fires a Telegram message to the retailer's chat ID.
// This is the last-resort fallback when FCM push fails.
func (t *TelegramClient) SendMessage(chatID string, text string) error {
	if t.botToken == "" {
		// Graceful no-op: token not configured; log the payload instead.
		log.Printf("[TELEGRAM FALLBACK — DRY RUN] chat_id=%s msg=%q\n", chatID, text)
		return nil
	}

	payload := telegramMessage{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "Markdown",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("telegram marshal error: %w", err)
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", t.apiBase, t.botToken)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body)) //nolint:noctx
	if err != nil {
		return fmt.Errorf("telegram HTTP error: %w", err)
	}
	defer resp.Body.Close()

	var result telegramResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("telegram response decode error: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("telegram API rejected: %s", result.Description)
	}

	log.Printf("[TELEGRAM] Message delivered to chat_id=%s\n", chatID)
	return nil
}

// FormatPredictionAlert formats a standard AI restock alert message.
func FormatPredictionAlert(shopName string, amount int64) string {
	return fmt.Sprintf(
		"🔔 *AI Restock Alert*\n\n*Shop:* %s\n*Amount:* %s\n\n_Tap the Retailer App to authorize dispatch._",
		shopName,
		formatAmount(amount),
	)
}

// formatAmount formats an integer as a human-readable string (e.g. 1_500_000 → "1,500,000").
func formatAmount(amount int64) string {
	s := fmt.Sprintf("%d", amount)
	// Insert commas every 3 digits from the right
	result := []byte{}
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, c)
	}
	return string(result)
}
