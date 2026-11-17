package notifications

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFormatFiscalReceiptQR_WholeSomLocalization(t *testing.T) {
	// 12,500,000 tiyin = 125,000 so'm (must divide minor by 100).
	uz := FormatFiscalReceiptQR("ord-100", "rec-7", "https://ofd.soliq.uz/check?t=1", 12500000, "UZS", "uz")
	if !strings.Contains(uz.Body, "125 000 so'm") {
		t.Fatalf("expected grouped whole so'm in uz body, got %q", uz.Body)
	}
	if strings.Contains(uz.Body, "12500000") {
		t.Fatalf("tiyin minor leak in uz body: %q", uz.Body)
	}
	if uz.DeepLink != "https://ofd.soliq.uz/check?t=1" {
		t.Fatalf("deep link = %q", uz.DeepLink)
	}

	ru := FormatFiscalReceiptQR("ord-100", "rec-7", "https://ofd.soliq.uz/check?t=1", 12500000, "UZS", "ru")
	if !strings.Contains(ru.Body, "125 000 сум") || !strings.Contains(ru.Title, "Фискальный чек") {
		t.Fatalf("unexpected ru notification: %+v", ru)
	}

	en := FormatFiscalReceiptQR("ord-100", "rec-7", "https://ofd.soliq.uz/check?t=1", 12500000, "UZS", "en")
	if !strings.Contains(en.Body, "125 000 UZS") {
		t.Fatalf("unexpected en notification: %+v", en)
	}
}

func TestTelegramBotSender_SendFiscalReceiptQR(t *testing.T) {
	var capturedPath string
	var capturedBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Fatalf("decode telegram body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":42}}`))
	}))
	defer srv.Close()

	sender := &TelegramBotSender{
		BotToken: "test-token",
		BaseURL:  srv.URL,
		Client:   srv.Client(),
	}
	notif := FormatFiscalReceiptQR("ord-22", "rec-22", "https://ofd.soliq.uz/check?id=rec-22", 500000, "UZS", "uz")
	if err := sender.SendFiscalReceiptQR(context.Background(), "998877", "uz", notif, notif.DeepLink); err != nil {
		t.Fatalf("SendFiscalReceiptQR: %v", err)
	}
	if capturedPath != "/bottest-token/sendMessage" {
		t.Fatalf("path = %q", capturedPath)
	}
	if capturedBody["chat_id"] != "998877" {
		t.Fatalf("chat_id = %v", capturedBody["chat_id"])
	}
	text, _ := capturedBody["text"].(string)
	if !strings.Contains(text, "5 000 so'm") || !strings.Contains(text, "https://ofd.soliq.uz/check?id=rec-22") {
		t.Fatalf("unexpected telegram text: %q", text)
	}
	if _, ok := capturedBody["reply_markup"]; !ok {
		t.Fatalf("expected inline_keyboard reply_markup for https QR url")
	}
}
