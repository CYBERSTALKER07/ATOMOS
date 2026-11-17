package notifications

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	defaultPlayMobileOriginator = "3700"
	maxPlayMobileResponseBytes  = 4096
)

// CircuitBreaker guards outbound provider calls against cascading failures.
// *circuit.Breaker satisfies this interface directly.
type CircuitBreaker interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

// Transport defines a notification delivery channel.
type Transport interface {
	Deliver(ctx context.Context, recipientID, recipientRole string, notif FormattedNotification) error
}

// RecipientContactResolver resolves a recipient ID/role into phone and email.
type RecipientContactResolver interface {
	ResolveContact(ctx context.Context, recipientID, recipientRole string) (phone string, email string, err error)
}

// LogTransport is a fallback that logs notifications without delivering them.
// Used when WS/FCM/APNs transports are not yet wired.
type LogTransport struct {
	Log *slog.Logger
}

// Deliver logs the notification delivery attempt.
func (t *LogTransport) Deliver(ctx context.Context, recipientID, recipientRole string, notif FormattedNotification) error {
	if t.Log == nil {
		t.Log = slog.Default()
	}
	t.Log.InfoContext(ctx, "notification delivery (log transport)",
		"recipient_id", recipientID,
		"recipient_role", recipientRole,
		"title", notif.Title,
		"priority", notif.Priority,
	)
	return nil
}

// SMSTransport delivers notifications via SMS gateways (PlayMobile / Twilio).
type SMSTransport struct {
	Provider   string // "playmobile", "twilio"
	BaseURL    string
	Login      string
	Password   string
	Originator string // PlayMobile originator; falls back to PLAYMOBILE_ORIGINATOR env, then "3700"
	AccountSID string
	AuthToken  string
	FromNumber string
	Resolver   RecipientContactResolver
	Client     *http.Client
	Breaker    CircuitBreaker
	Log        *slog.Logger
}

// SetCircuitBreaker configures the outbound circuit breaker for SMS delivery.
func (t *SMSTransport) SetCircuitBreaker(b CircuitBreaker) {
	if t == nil {
		return
	}
	t.Breaker = b
}

func (t *SMSTransport) runWithBreaker(ctx context.Context, fn func(context.Context) error) error {
	if t != nil && t.Breaker != nil {
		return t.Breaker.Do(ctx, fn)
	}
	return fn(ctx)
}

// Deliver sends an SMS notification.
func (t *SMSTransport) Deliver(ctx context.Context, recipientID, recipientRole string, notif FormattedNotification) error {
	log := t.Log
	if log == nil {
		log = slog.Default()
	}
	phone := recipientID
	if t.Resolver != nil {
		p, _, err := t.Resolver.ResolveContact(ctx, recipientID, recipientRole)
		if err == nil && p != "" {
			phone = p
		}
	}
	phone = strings.TrimSpace(phone)
	if phone == "" {
		log.WarnContext(ctx, "sms delivery skipped: missing phone number", "recipient_id", recipientID)
		return nil
	}

	hc := t.Client
	if hc == nil {
		hc = &http.Client{Timeout: 10 * time.Second}
	}

	body := notif.Title + ": " + notif.Body
	if notif.DeepLink != "" {
		body += " " + notif.DeepLink
	}

	switch strings.ToLower(t.Provider) {
	case "playmobile":
		return t.runWithBreaker(ctx, func(callCtx context.Context) error {
			return t.sendPlayMobile(callCtx, hc, phone, body)
		})
	case "twilio":
		return t.runWithBreaker(ctx, func(callCtx context.Context) error {
			return t.sendTwilio(callCtx, hc, phone, body)
		})
	default:
		log.InfoContext(ctx, "sms mock delivery", "provider", t.Provider, "phone", phone, "body", body)
		return nil
	}
}

func (t *SMSTransport) resolvePlayMobileOriginator() string {
	if v := strings.TrimSpace(t.Originator); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv("PLAYMOBILE_ORIGINATOR")); v != "" {
		return v
	}
	return defaultPlayMobileOriginator
}

func (t *SMSTransport) sendPlayMobile(ctx context.Context, hc *http.Client, phone, body string) error {
	endpoint := t.BaseURL
	if endpoint == "" {
		endpoint = "https://send.smsxabar.uz/broker-api/send"
	}
	payload := map[string]any{
		"messages": []map[string]any{{
			"recipient":  strings.TrimPrefix(phone, "+"),
			"message-id": uuid.NewString(),
			"sms": map[string]string{
				"originator": t.resolvePlayMobileOriginator(),
				"content":    body,
			},
		}},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	if t.Login != "" && t.Password != "" {
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(t.Login+":"+t.Password)))
	}
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, maxPlayMobileResponseBytes))
	if readErr != nil {
		return fmt.Errorf("playmobile read response: %w", readErr)
	}
	if resp.StatusCode/100 != 2 {
		if bodyErr := parsePlayMobileBodyError(respBody); bodyErr != nil {
			return fmt.Errorf("playmobile status %d: %w", resp.StatusCode, bodyErr)
		}
		return fmt.Errorf("playmobile status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return parsePlayMobileBodyError(respBody)
}

func parsePlayMobileBodyError(raw []byte) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || (trimmed[0] != '{' && trimmed[0] != '[') {
		return nil
	}
	var obj map[string]any
	if err := json.Unmarshal(trimmed, &obj); err != nil || obj == nil {
		return nil
	}
	if err := inspectPlayMobileErrorFields(obj); err != nil {
		return err
	}
	if msgs, ok := obj["messages"].([]any); ok {
		for _, item := range msgs {
			if m, ok := item.(map[string]any); ok {
				if err := inspectPlayMobileErrorFields(m); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func inspectPlayMobileErrorFields(m map[string]any) error {
	code := playMobileValueString(m["error-code"])
	if code == "" {
		code = playMobileValueString(m["error_code"])
	}
	if code == "" {
		code = playMobileValueString(m["code"])
	}
	desc := strings.TrimSpace(playMobileValueString(m["error-description"]))
	if desc == "" {
		desc = strings.TrimSpace(playMobileValueString(m["error_description"]))
	}
	if desc == "" {
		desc = strings.TrimSpace(playMobileValueString(m["message"]))
	}

	if code != "" && code != "0" {
		if desc != "" {
			return fmt.Errorf("playmobile error %s: %s", code, desc)
		}
		return fmt.Errorf("playmobile error %s", code)
	}
	if errField, hasErr := m["error"]; hasErr && errField != nil {
		errStr := strings.TrimSpace(playMobileValueString(errField))
		if errStr != "" && errStr != "0" && errStr != "false" {
			return fmt.Errorf("playmobile error: %s", errStr)
		}
	}
	if status, ok := m["status"].(string); ok {
		switch strings.ToUpper(strings.TrimSpace(status)) {
		case "ERROR", "FAILED", "REJECTED":
			if desc != "" {
				return fmt.Errorf("playmobile status %s: %s", status, desc)
			}
			return fmt.Errorf("playmobile status %s", status)
		}
	}
	return nil
}

func playMobileValueString(v any) string {
	switch val := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(val)
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	default:
		b, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprint(val)
		}
		return string(b)
	}
}

func (t *SMSTransport) sendTwilio(ctx context.Context, hc *http.Client, phone, body string) error {
	endpoint := t.BaseURL
	if endpoint == "" {
		endpoint = fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", t.AccountSID)
	}
	form := url.Values{}
	form.Set("To", phone)
	form.Set("From", t.FromNumber)
	form.Set("Body", body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if t.AccountSID != "" && t.AuthToken != "" {
		req.SetBasicAuth(t.AccountSID, t.AuthToken)
	}
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("twilio sms status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

// EmailTransport delivers notifications via SendGrid or SMTP REST API.
type EmailTransport struct {
	Provider string // "sendgrid", "smtp"
	APIKey   string
	From     string
	FromName string
	Resolver RecipientContactResolver
	Client   *http.Client
	Log      *slog.Logger
}

// Deliver sends an Email notification.
func (t *EmailTransport) Deliver(ctx context.Context, recipientID, recipientRole string, notif FormattedNotification) error {
	if t.Log == nil {
		t.Log = slog.Default()
	}
	email := recipientID
	if t.Resolver != nil {
		_, e, err := t.Resolver.ResolveContact(ctx, recipientID, recipientRole)
		if err == nil && e != "" {
			email = e
		}
	}
	email = strings.TrimSpace(email)
	if email == "" || !strings.Contains(email, "@") {
		t.Log.WarnContext(ctx, "email delivery skipped: invalid email", "recipient_id", recipientID)
		return nil
	}

	hc := t.Client
	if hc == nil {
		hc = &http.Client{Timeout: 10 * time.Second}
	}

	fromName := t.FromName
	if fromName == "" {
		fromName = "PegasusX Notifications"
	}

	payload := map[string]any{
		"personalizations": []map[string]any{
			{"to": []map[string]string{{"email": email}}},
		},
		"from":    map[string]string{"email": t.From, "name": fromName},
		"subject": notif.Title,
		"content": []map[string]string{{"type": "text/plain", "value": notif.Body + "\n\n" + notif.DeepLink}},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.sendgrid.com/v3/mail/send", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.APIKey)
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("sendgrid status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

// CompositeTransport delivers notifications to multiple underlying transports concurrently.
type CompositeTransport struct {
	Transports []Transport
}

// Deliver sends notification through all child transports.
func (c *CompositeTransport) Deliver(ctx context.Context, recipientID, recipientRole string, notif FormattedNotification) error {
	var firstErr error
	for _, t := range c.Transports {
		if t == nil {
			continue
		}
		if err := t.Deliver(ctx, recipientID, recipientRole, notif); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
