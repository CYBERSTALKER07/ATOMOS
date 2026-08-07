package ar

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Off-app dunning transports: SMS + email behind a per-channel provider
// interface — the pattern every collections platform uses. FCM/inbox remain
// the in-app path (bootstrap notify); these channels reach actors who never
// open the app.
//
// Selection (fail-closed):
//
//	DUNNING_SMS_PROVIDER=twilio|playmobile  (unset/empty/off = SMS disabled)
//	DUNNING_EMAIL_PROVIDER=sendgrid         (unset/empty/off = email disabled)
//
// A selected provider with missing credentials is a construction error at
// bootstrap — a misconfigured collections path must surface at deploy, not at
// dunning time.

// Contact is the resolved off-app address set for one actor.
type Contact struct {
	Phone string
	Email string
	Name  string
}

// ChannelTransport sends one dunning notice over one channel.
type ChannelTransport interface {
	Channel() string
	Send(ctx context.Context, to Contact, body string) error
}

// ContactResolver resolves an invoice's parties to off-app contacts.
type ContactResolver interface {
	ResolveRetailer(ctx context.Context, retailerID string) (Contact, error)
	ResolveSupplierStaff(ctx context.Context, supplierID string) ([]Contact, error)
}

func envOr(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

// TransportsFromEnv builds the enabled off-app transports. Unknown providers
// and missing credentials are hard errors (fail-closed).
func TransportsFromEnv() ([]ChannelTransport, error) {
	var out []ChannelTransport
	switch p := strings.ToLower(strings.TrimSpace(os.Getenv("DUNNING_SMS_PROVIDER"))); p {
	case "", "off":
	case "twilio":
		t, err := twilioSMSFromEnv()
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	case "playmobile":
		t, err := playMobileSMSFromEnv()
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	default:
		return nil, fmt.Errorf("unknown DUNNING_SMS_PROVIDER %q", p)
	}
	switch p := strings.ToLower(strings.TrimSpace(os.Getenv("DUNNING_EMAIL_PROVIDER"))); p {
	case "", "off":
	case "sendgrid":
		t, err := sendGridFromEnv()
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	default:
		return nil, fmt.Errorf("unknown DUNNING_EMAIL_PROVIDER %q", p)
	}
	return out, nil
}

// MultiChannelNotify composes the dunning notify: in-app notify stays owned by
// bootstrap; this fans out to off-app transports. Per-channel failures are
// logged and aggregated; one dead channel never blocks the others.
func MultiChannelNotify(log *slog.Logger, resolver ContactResolver, transports []ChannelTransport) DunningNotifyFunc {
	return func(ctx context.Context, inv Invoice, prevStep, nextStep int64) error {
		if len(transports) == 0 || resolver == nil {
			return nil
		}
		_, title, body := NotifyMessage(inv, nextStep)
		text := strings.TrimSpace(title + "\n" + body)

		var targets []Contact
		if c, err := resolver.ResolveRetailer(ctx, inv.RetailerID); err != nil {
			log.WarnContext(ctx, "dunning contact resolve failed", "retailer_id", inv.RetailerID, "err", err)
		} else {
			targets = append(targets, c)
		}
		if staff, err := resolver.ResolveSupplierStaff(ctx, inv.SupplierID); err != nil {
			log.WarnContext(ctx, "dunning supplier staff resolve failed", "supplier_id", inv.SupplierID, "err", err)
		} else {
			targets = append(targets, staff...)
		}

		var errs []error
		for _, tr := range transports {
			for _, to := range targets {
				if tr.Channel() == "sms" && to.Phone == "" {
					continue
				}
				if tr.Channel() == "email" && to.Email == "" {
					continue
				}
				if err := tr.Send(ctx, to, text); err != nil {
					errs = append(errs, fmt.Errorf("%s -> %s: %w", tr.Channel(), maskContact(to, tr.Channel()), err))
					log.WarnContext(ctx, "dunning off-app send failed",
						"channel", tr.Channel(), "invoice_id", inv.InvoiceID, "err", err)
				}
			}
		}
		return errors.Join(errs...)
	}
}

func maskContact(c Contact, channel string) string {
	switch channel {
	case "sms":
		if len(c.Phone) > 4 {
			return "***" + c.Phone[len(c.Phone)-4:]
		}
		return "***"
	case "email":
		if i := strings.Index(c.Email, "@"); i > 1 {
			return c.Email[:1] + "***" + c.Email[i:]
		}
		return "***"
	}
	return "***"
}

// --- SMS: Twilio ---

type twilioSMS struct {
	sid, token, from string
	baseURL          string
	hc               *http.Client
}

func twilioSMSFromEnv() (ChannelTransport, error) {
	sid, token, from := os.Getenv("TWILIO_ACCOUNT_SID"), os.Getenv("TWILIO_AUTH_TOKEN"), os.Getenv("TWILIO_FROM_NUMBER")
	if sid == "" || token == "" || from == "" {
		return nil, fmt.Errorf("DUNNING_SMS_PROVIDER=twilio requires TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN, TWILIO_FROM_NUMBER")
	}
	return &twilioSMS{sid: sid, token: token, from: from,
		baseURL: envOr("TWILIO_BASE_URL", "https://api.twilio.com"),
		hc:      &http.Client{Timeout: 10 * time.Second}}, nil
}

func (t *twilioSMS) Channel() string { return "sms" }

func (t *twilioSMS) Send(ctx context.Context, to Contact, body string) error {
	form := url.Values{"From": {t.from}, "To": {to.Phone}, "Body": {body}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		t.baseURL+"/2010-04-01/Accounts/"+t.sid+"/Messages.json",
		strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(t.sid+":"+t.token)))
	resp, err := t.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("twilio status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

// --- SMS: PlayMobile (send.smsxabar.uz, UZ carrier-grade) ---

type playMobileSMS struct {
	login, password string
	baseURL         string
	hc              *http.Client
}

func playMobileSMSFromEnv() (ChannelTransport, error) {
	login, password := os.Getenv("PLAYMOBILE_LOGIN"), os.Getenv("PLAYMOBILE_PASSWORD")
	if login == "" || password == "" {
		return nil, fmt.Errorf("DUNNING_SMS_PROVIDER=playmobile requires PLAYMOBILE_LOGIN, PLAYMOBILE_PASSWORD")
	}
	return &playMobileSMS{
		login: login, password: password,
		baseURL: envOr("PLAYMOBILE_BASE_URL", "https://send.smsxabar.uz/broker-api/send"),
		hc:      &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func (p *playMobileSMS) Channel() string { return "sms" }

func (p *playMobileSMS) Send(ctx context.Context, to Contact, body string) error {
	payload := map[string]any{
		"messages": []map[string]any{{
			"recipient":  strings.TrimPrefix(to.Phone, "+"),
			"message-id": fmt.Sprintf("dun-%d", time.Now().UnixNano()),
			"sms":        map[string]string{"originator": "3700", "content": body},
		}},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(p.login+":"+p.password)))
	resp, err := p.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("playmobile status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

// --- Email: SendGrid ---

type sendGridEmail struct {
	apiKey, from, fromName string
	hc                     *http.Client
}

func sendGridFromEnv() (ChannelTransport, error) {
	key, from := os.Getenv("SENDGRID_API_KEY"), os.Getenv("SENDGRID_FROM_EMAIL")
	if key == "" || from == "" {
		return nil, fmt.Errorf("DUNNING_EMAIL_PROVIDER=sendgrid requires SENDGRID_API_KEY, SENDGRID_FROM_EMAIL")
	}
	return &sendGridEmail{
		apiKey: key, from: from,
		fromName: envOr("SENDGRID_FROM_NAME", "PegasusX Collections"),
		hc:       &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func (s *sendGridEmail) Channel() string { return "email" }

func (s *sendGridEmail) Send(ctx context.Context, to Contact, body string) error {
	payload := map[string]any{
		"personalizations": []map[string]any{
			{"to": []map[string]string{{"email": to.Email, "name": to.Name}}},
		},
		"from":    map[string]string{"email": s.from, "name": s.fromName},
		"subject": "PegasusX: invoice payment notice",
		"content": []map[string]string{{"type": "text/plain", "value": body}},
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
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	resp, err := s.hc.Do(req)
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
