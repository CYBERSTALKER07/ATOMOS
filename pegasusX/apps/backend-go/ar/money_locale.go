package ar

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// FormatWholeSom converts minor units (tiyin) into whole currency units using
// integer half-up rounding and space thousands grouping (e.g. "1 250 so'm").
func FormatWholeSom(minor int64, currency string) string {
	whole := minor / 100
	rem := minor % 100
	if rem >= 50 {
		whole++
	} else if rem <= -50 {
		whole--
	}
	grouped := formatGroupedInt64(whole)
	cur := strings.ToUpper(strings.TrimSpace(currency))
	if cur == "" || cur == "UZS" {
		return grouped + " so'm"
	}
	return grouped + " " + cur
}

func formatGroupedInt64(v int64) string {
	neg := v < 0
	var digits string
	if neg {
		// Handle MinInt64 safely without unary negation overflow.
		if v == -1<<63 {
			digits = strconv.FormatUint(uint64(1)<<63, 10)
		} else {
			digits = strconv.FormatInt(-v, 10)
		}
	} else {
		digits = strconv.FormatInt(v, 10)
	}
	n := len(digits)
	if n <= 3 {
		if neg {
			return "-" + digits
		}
		return digits
	}
	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	first := n % 3
	if first == 0 {
		first = 3
	}
	b.WriteString(digits[:first])
	for i := first; i < n; i += 3 {
		b.WriteByte(' ')
		b.WriteString(digits[i : i+3])
	}
	return b.String()
}

// NormalizeLocale normalizes a BCP-47/short locale into "uz", "ru", or "en".
func NormalizeLocale(raw string) (string, bool) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return "", false
	}
	if idx := strings.IndexAny(s, "-_"); idx > 0 {
		s = s[:idx]
	}
	switch s {
	case "uz", "uzb":
		return "uz", true
	case "ru", "rus":
		return "ru", true
	case "en", "eng":
		return "en", true
	default:
		return "", false
	}
}

// ResolveLocale derives retailer notification locale from optional LOCALE_PREF capability config
// or CountryCode (RU/KZ -> "ru", default "uz").
func ResolveLocale(countryCode, packConfigJSON string) string {
	trimmed := strings.TrimSpace(packConfigJSON)
	if trimmed != "" {
		var cfg map[string]any
		if err := json.Unmarshal([]byte(trimmed), &cfg); err == nil {
			for _, key := range []string{"locale", "preferred_locale", "language", "lang"} {
				if raw, ok := cfg[key].(string); ok {
					if loc, valid := NormalizeLocale(raw); valid {
						return loc
					}
				}
			}
		} else if loc, valid := NormalizeLocale(trimmed); valid {
			return loc
		}
	}
	switch strings.ToUpper(strings.TrimSpace(countryCode)) {
	case "RU", "KZ":
		return "ru"
	default:
		return "uz"
	}
}

// LocalizedNotifyMessage formats a dunning notification in uz (default), ru, or en
// using integer so'm amounts (never raw tiyin).
func LocalizedNotifyMessage(inv Invoice, nextStep int64, locale string) (eventType, title, body string) {
	loc, ok := NormalizeLocale(locale)
	if !ok {
		loc = "uz"
	}
	name := StepName(nextStep)
	eventType = "AR_DUNNING_" + name
	amt := FormatWholeSom(inv.BalanceMinor, inv.Currency)
	orderRef := strings.TrimSpace(inv.OrderID)
	if orderRef == "" {
		orderRef = inv.InvoiceID
	}
	dueStr := inv.DueAt.Format("2006-01-02")

	switch loc {
	case "ru":
		switch nextStep {
		case StepDueSoon:
			title = fmt.Sprintf("Скоро срок оплаты %s", inv.InvoiceID)
			body = fmt.Sprintf("Заказ %s: задолженность %s. Срок оплаты: %s.", orderRef, amt, dueStr)
		case StepOverdue:
			title = fmt.Sprintf("Просрочка оплаты %s", inv.InvoiceID)
			body = fmt.Sprintf("Заказ %s: просроченная задолженность %s (срок %s). Пожалуйста, оплатите счет.", orderRef, amt, dueStr)
		case StepEscalated1, StepEscalated2:
			title = fmt.Sprintf("Напоминание о просрочке %s", inv.InvoiceID)
			body = fmt.Sprintf("Заказ %s: остаток %s не оплачен (срок %s). Оплатите во избежание ограничения кредита.", orderRef, amt, dueStr)
		case StepCreditHold:
			title = fmt.Sprintf("Кредит приостановлен %s", inv.InvoiceID)
			body = fmt.Sprintf("Заказ %s: кредит приостановлен из-за задолженности %s (срок %s).", orderRef, amt, dueStr)
		case StepCollections:
			title = fmt.Sprintf("Взыскание задолженности %s", inv.InvoiceID)
			body = fmt.Sprintf("Заказ %s: задолженность %s передана в отдел взыскания (срок %s).", orderRef, amt, dueStr)
		default:
			title = fmt.Sprintf("Счет %s: %s", inv.InvoiceID, name)
			body = fmt.Sprintf("Заказ %s: сумма %s — этап %s (срок %s).", orderRef, amt, name, dueStr)
		}
	case "en":
		title = fmt.Sprintf("Invoice %s: %s", inv.InvoiceID, name)
		body = fmt.Sprintf("Order %s balance %s — dunning step %s (due %s)", orderRef, amt, name, dueStr)
	default:
		switch nextStep {
		case StepDueSoon:
			title = fmt.Sprintf("To'lov muddati yaqinlashmoqda %s", inv.InvoiceID)
			body = fmt.Sprintf("Buyurtma %s bo'yicha qarzdorlik %s. To'lov muddati: %s.", orderRef, amt, dueStr)
		case StepOverdue:
			title = fmt.Sprintf("Muddati o'tgan hisob %s", inv.InvoiceID)
			body = fmt.Sprintf("Buyurtma %s bo'yicha muddati o'tgan qarzdorlik %s (muddat %s). Iltimos, to'lovni amalga oshiring.", orderRef, amt, dueStr)
		case StepEscalated1, StepEscalated2:
			title = fmt.Sprintf("Kechiktirilgan to'lov eslatmasi %s", inv.InvoiceID)
			body = fmt.Sprintf("Buyurtma %s bo'yicha %s qoldiq hali to'lanmagan (muddat %s). Kredit limitini saqlash uchun to'lang.", orderRef, amt, dueStr)
		case StepCreditHold:
			title = fmt.Sprintf("Kredit to'xtatildi %s", inv.InvoiceID)
			body = fmt.Sprintf("Buyurtma %s bo'yicha %s qarzdorlik sababli kredit vaqtincha to'xtatildi (muddat %s).", orderRef, amt, dueStr)
		case StepCollections:
			title = fmt.Sprintf("Qarzdorlik undiruvi %s", inv.InvoiceID)
			body = fmt.Sprintf("Buyurtma %s bo'yicha %s qarzdorlik undiruv bo'limiga o'tkazildi (muddat %s).", orderRef, amt, dueStr)
		default:
			title = fmt.Sprintf("Hisob-faktura %s: %s", inv.InvoiceID, name)
			body = fmt.Sprintf("Buyurtma %s bo'yicha qoldiq %s — bosqich %s (muddat %s).", orderRef, amt, name, dueStr)
		}
	}
	return eventType, title, body
}
