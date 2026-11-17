package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultTelegramAPIBaseURL  = "https://api.telegram.org"
	maxTelegramResponseBytes   = 8192
	defaultTelegramHTTPTimeout = 10 * time.Second
)

// NormalizeLocale normalizes notification locale to uz, ru, or en (defaulting to uz).
func NormalizeLocale(locale string) string {
	l := strings.ToLower(strings.TrimSpace(locale))
	switch {
	case strings.HasPrefix(l, "ru"):
		return "ru"
	case strings.HasPrefix(l, "en"):
		return "en"
	default:
		return "uz"
	}
}

// WholeUZS converts minor units (tiyin) to whole Uzbek so'm using integer arithmetic only.
func WholeUZS(minor int64) int64 {
	return minor / 100
}

// FormatGroupedInt formats an int64 with ASCII space thousands separators.
func FormatGroupedInt(v int64) string {
	neg := v < 0
	if neg {
		// Guard MinInt64 safely without overflowing.
		if v == -1<<63 {
			return "-9 223 372 036 854 775 808"
		}
		v = -v
	}
	raw := strconv.FormatInt(v, 10)
	if len(raw) <= 3 {
		if neg {
			return "-" + raw
		}
		return raw
	}
	var out strings.Builder
	if neg {
		out.WriteByte('-')
	}
	rem := len(raw) % 3
	if rem > 0 {
		out.WriteString(raw[:rem])
		if len(raw) > rem {
			out.WriteByte(' ')
		}
	}
	for i := rem; i < len(raw); i += 3 {
		out.WriteString(raw[i : i+3])
		if i+3 < len(raw) {
			out.WriteByte(' ')
		}
	}
	return out.String()
}

// FormatWholeUZS formats tiyin minor units into grouped whole so'm (e.g. "125 000 so'm").
func FormatWholeUZS(minor int64) string {
	return FormatGroupedInt(WholeUZS(minor)) + " so'm"
}

// FormatFiscalReceiptQR produces a localized fiscal QR receipt notification
// using whole so'm (minor / 100) rather than raw tiyin units.
func FormatFiscalReceiptQR(orderID, receiptID, fiscalQR string, amountMinor int64, currency, locale string) FormattedNotification {
	loc := NormalizeLocale(locale)
	wholeStr := FormatGroupedInt(WholeUZS(amountMinor))
	cur := strings.ToUpper(strings.TrimSpace(currency))
	orderLabel := strings.TrimSpace(orderID)
	if orderLabel == "" {
		orderLabel = "order"
	}
	recLabel := strings.TrimSpace(receiptID)
	if recLabel == "" {
		recLabel = "-"
	}
	deepLink := strings.TrimSpace(fiscalQR)
	if deepLink == "" {
		deepLink = "/orders/" + orderLabel
	}

	switch loc {
	case "ru":
		unit := "сум"
		if cur != "" && cur != "UZS" {
			unit = cur
		}
		return FormattedNotification{
			Title:    "Фискальный чек готов",
			Body:     fmt.Sprintf("Заказ %s: чек №%s на %s %s.", orderLabel, recLabel, wholeStr, unit),
			DeepLink: deepLink,
			Priority: "normal",
		}
	case "en":
		unit := "UZS"
		if cur != "" {
			unit = cur
		}
		return FormattedNotification{
			Title:    "Fiscal receipt ready",
			Body:     fmt.Sprintf("Order %s: receipt #%s for %s %s.", orderLabel, recLabel, wholeStr, unit),
			DeepLink: deepLink,
			Priority: "normal",
		}
	default:
		unit := "so'm"
		if cur != "" && cur != "UZS" {
			unit = cur
		}
		return FormattedNotification{
			Title:    "Fiskal chek tayyor",
			Body:     fmt.Sprintf("Buyurtma %s: %s %s (chek №%s).", orderLabel, wholeStr, unit, recLabel),
			DeepLink: deepLink,
			Priority: "normal",
		}
	}
}

// TelegramBotSender sends localized notifications and QR buttons via Telegram Bot API.
type TelegramBotSender struct {
	BotToken string
	BaseURL  string
	Client   *http.Client
	Breaker  CircuitBreaker
	Log      *slog.Logger
}

// NewTelegramBotSenderFromEnv creates a TelegramBotSender configured from environment variables.
func NewTelegramBotSenderFromEnv(log *slog.Logger) *TelegramBotSender {
	if log == nil {
		log = slog.Default()
	}
	return &TelegramBotSender{
		BotToken: strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN")),
		BaseURL:  strings.TrimSpace(os.Getenv("TELEGRAM_API_BASE_URL")),
		Log:      log,
	}
}

// SendFiscalReceiptQR delivers a fiscal QR receipt message to a bound Telegram chat.
func (s *TelegramBotSender) SendFiscalReceiptQR(ctx context.Context, chatID, locale string, notif FormattedNotification, qrURL string) error {
	if s == nil {
		return nil
	}
	token := strings.TrimSpace(s.BotToken)
	chatID = strings.TrimSpace(chatID)
	if token == "" || chatID == "" {
		return nil
	}
	base := strings.TrimRight(strings.TrimSpace(s.BaseURL), "/")
	if base == "" {
		base = defaultTelegramAPIBaseURL
	}
	hc := s.Client
	if hc == nil {
		hc = &http.Client{Timeout: defaultTelegramHTTPTimeout}
	}

	text := strings.TrimSpace(notif.Title) + "\n" + strings.TrimSpace(notif.Body)
	qr := strings.TrimSpace(qrURL)
	if qr != "" {
		text += "\n" + qr
	}

	payload := map[string]any{
		"chat_id":                  chatID,
		"text":                     text,
		"disable_web_page_preview": false,
	}
	if strings.HasPrefix(strings.ToLower(qr), "https://") || strings.HasPrefix(strings.ToLower(qr), "http://") {
		btnLabel := "Fiskal chek QR"
		switch NormalizeLocale(locale) {
		case "ru":
			btnLabel = "Фискальный чек QR"
		case "en":
			btnLabel = "Fiscal receipt QR"
		}
		payload["reply_markup"] = map[string]any{
			"inline_keyboard": [][]map[string]string{{
				{"text": btnLabel, "url": qr},
			}},
		}
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	call := func(callCtx context.Context) error {
		endpoint := base + "/bot" + token + "/sendMessage"
		req, err := http.NewRequestWithContext(callCtx, http.MethodPost, endpoint, bytes.NewReader(raw))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := hc.Do(req)
		if err != nil {
			return err
		}
		defer func() {
			if cErr := resp.Body.Close(); cErr != nil && s.Log != nil {
				s.Log.DebugContext(callCtx, "close telegram response body", "err", cErr)
			}
		}()
		respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, maxTelegramResponseBytes))
		if readErr != nil {
			return fmt.Errorf("read telegram response: %w", readErr)
		}
		if resp.StatusCode/100 != 2 {
			return fmt.Errorf("telegram sendMessage status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
		}
		var parsed struct {
			OK          bool   `json:"ok"`
			Description string `json:"description"`
		}
		if err := json.Unmarshal(respBody, &parsed); err == nil && !parsed.OK {
			return fmt.Errorf("telegram sendMessage rejected: %s", parsed.Description)
		}
		return nil
	}

	if s.Breaker != nil {
		return s.Breaker.Do(ctx, call)
	}
	return call(ctx)
}
