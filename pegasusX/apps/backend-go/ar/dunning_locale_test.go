package ar

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestFormatWholeSom_RoundingAndGrouping(t *testing.T) {
	cases := []struct {
		minor int64
		cur   string
		want  string
	}{
		{0, "UZS", "0 so'm"},
		{49, "UZS", "0 so'm"},
		{50, "UZS", "1 so'm"},
		{125_000, "UZS", "1 250 so'm"},
		{125_049, "", "1 250 so'm"},
		{125_050, "uzs", "1 251 so'm"},
		{150_000_000, "UZS", "1 500 000 so'm"},
		{-125_050, "UZS", "-1 251 so'm"},
		{100_000, "USD", "1 000 USD"},
	}
	for _, tc := range cases {
		if got := FormatWholeSom(tc.minor, tc.cur); got != tc.want {
			t.Fatalf("FormatWholeSom(%d, %q) = %q, want %q", tc.minor, tc.cur, got, tc.want)
		}
	}
}

func TestResolveLocale_PrecedenceAndDefaults(t *testing.T) {
	if got := ResolveLocale("UZ", ""); got != "uz" {
		t.Fatalf("got %q want uz", got)
	}
	if got := ResolveLocale("RU", ""); got != "ru" {
		t.Fatalf("got %q want ru", got)
	}
	if got := ResolveLocale("KZ", ""); got != "ru" {
		t.Fatalf("got %q want ru", got)
	}
	if got := ResolveLocale("US", `{"locale":"ru"}`); got != "ru" {
		t.Fatalf("pack override failed: %q", got)
	}
	if got := ResolveLocale("RU", `{"preferred_locale":"uz-Latn-UZ"}`); got != "uz" {
		t.Fatalf("pack preferred_locale failed: %q", got)
	}
	if got := ResolveLocale("", `{"language":"en"}`); got != "en" {
		t.Fatalf("pack language failed: %q", got)
	}
}

func TestLocalizedNotifyMessage_And_MultiChannelNotify(t *testing.T) {
	inv := Invoice{
		InvoiceID:    "inv-42",
		OrderID:      "ord-42",
		RetailerID:   "ret-1",
		SupplierID:   "sup-1",
		BalanceMinor: 125_000, // 1 250 so'm (NEVER 125000 in SMS!)
		Currency:     "UZS",
		DueAt:        time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC),
	}

	ev, titleUz, bodyUz := NotifyMessage(inv, StepOverdue)
	if ev != "AR_DUNNING_OVERDUE" {
		t.Fatalf("eventType = %q", ev)
	}
	if !strings.Contains(bodyUz, "1 250 so'm") || strings.Contains(bodyUz, "125000") || !strings.Contains(titleUz, "inv-42") {
		t.Fatalf("unexpected uz default notify: title=%q body=%q", titleUz, bodyUz)
	}

	_, titleRu, bodyRu := LocalizedNotifyMessage(inv, StepCreditHold, "ru")
	if !strings.Contains(bodyRu, "1 250 so'm") || strings.Contains(bodyRu, "125000") || !strings.Contains(titleRu, "inv-42") {
		t.Fatalf("unexpected ru notify: title=%q body=%q", titleRu, bodyRu)
	}

	_, _, bodyEn := LocalizedNotifyMessage(inv, StepDueSoon, "en")
	if !strings.Contains(bodyEn, "1 250 so'm") || strings.Contains(bodyEn, "125000") {
		t.Fatalf("unexpected en notify: body=%q", bodyEn)
	}

	sms := &fakeTransport{channel: "sms"}
	resolver := fakeResolver{
		retailer: Contact{Phone: "+998901112233", Locale: "ru"},
	}
	notify := MultiChannelNotify(testLogger(), resolver, []ChannelTransport{sms})
	if err := notify(context.Background(), inv, StepDueSoon, StepOverdue); err != nil {
		t.Fatal(err)
	}
	if len(sms.sent) != 1 || !strings.Contains(sms.sent[0], "1 250 so'm") || !strings.Contains(sms.sent[0], "Заказ ord-42") {
		t.Fatalf("expected Russian integer so'm SMS, got %+v", sms.sent)
	}
}
