package notifications

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// SMSProvider is the interface for sending SMS messages.
// Implementations: EskizProvider (UZ), TwilioProvider (global).
type SMSProvider interface {
	Send(phone, message string) error
}

// ─── Eskiz.uz Provider (Uzbekistan) ─────────────────────────────────────────

// EskizProvider sends SMS via Eskiz.uz API.
// Credentials loaded from env: ESKIZ_EMAIL, ESKIZ_PASSWORD.
// Auth token is auto-refreshed on first send and when expired.
type EskizProvider struct {
	Email    string
	Password string

	mu       sync.Mutex
	token    string
	expireAt time.Time
}

const eskizBaseURL = "https://notify.eskiz.uz/api"

func (e *EskizProvider) authenticate() error {
	if e.Email == "" || e.Password == "" {
		return fmt.Errorf("eskiz: no credentials configured (ESKIZ_EMAIL / ESKIZ_PASSWORD)")
	}

	form := url.Values{}
	form.Set("email", e.Email)
	form.Set("password", e.Password)

	resp, err := http.PostForm(eskizBaseURL+"/auth/login", form)
	if err != nil {
		return fmt.Errorf("eskiz auth: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("eskiz auth failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("eskiz auth decode: %w", err)
	}

	e.token = result.Data.Token
	e.expireAt = time.Now().Add(29 * 24 * time.Hour) // Eskiz tokens valid ~30 days
	log.Printf("[SMS/ESKIZ] Authenticated, token valid until %s", e.expireAt.Format(time.RFC3339))
	return nil
}

func (e *EskizProvider) ensureToken() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.token == "" || time.Now().After(e.expireAt) {
		return e.authenticate()
	}
	return nil
}

func (e *EskizProvider) Send(phone, message string) error {
	if e.Email == "" {
		log.Printf("[SMS/ESKIZ] No credentials, skipping → %s", phone)
		return nil
	}

	if err := e.ensureToken(); err != nil {
		return err
	}

	form := url.Values{}
	form.Set("mobile_phone", phone)
	form.Set("message", message)
	form.Set("from", "4546") // Eskiz default sender ID

	req, err := http.NewRequest("POST", eskizBaseURL+"/message/sms/send", strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("eskiz send: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+e.token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("eskiz send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		// Token expired mid-session — re-auth and retry once
		e.mu.Lock()
		e.token = ""
		e.mu.Unlock()
		if err := e.ensureToken(); err != nil {
			return err
		}
		return e.Send(phone, message) // Single retry
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("eskiz send failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	log.Printf("[SMS/ESKIZ] Sent → %s", phone)
	return nil
}

// ─── Twilio Provider (Global) ───────────────────────────────────────────────

// TwilioProvider sends SMS via Twilio REST API.
// Credentials loaded from env: TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN, TWILIO_FROM_NUMBER.
type TwilioProvider struct {
	AccountSID string
	AuthToken  string
	FromNumber string
}

func (t *TwilioProvider) Send(phone, message string) error {
	if t.AccountSID == "" {
		log.Printf("[SMS/TWILIO] No credentials, skipping → %s", phone)
		return nil
	}

	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", t.AccountSID)

	form := url.Values{}
	form.Set("To", phone)
	form.Set("From", t.FromNumber)
	form.Set("Body", message)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("twilio send: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(t.AccountSID, t.AuthToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("twilio send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("twilio send failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	log.Printf("[SMS/TWILIO] Sent → %s", phone)
	return nil
}

// ─── Factory ────────────────────────────────────────────────────────────────

// NewSMSProvider returns the appropriate SMS provider based on the provider name.
// Follows FCM graceful degradation: no-op (nil return) when credentials are missing.
func NewSMSProvider(providerName string) SMSProvider {
	switch providerName {
	case "ESKIZ":
		return &EskizProvider{
			Email:    getEnvDefault("ESKIZ_EMAIL", ""),
			Password: getEnvDefault("ESKIZ_PASSWORD", ""),
		}
	case "TWILIO":
		return &TwilioProvider{
			AccountSID: getEnvDefault("TWILIO_ACCOUNT_SID", ""),
			AuthToken:  getEnvDefault("TWILIO_AUTH_TOKEN", ""),
			FromNumber: getEnvDefault("TWILIO_FROM_NUMBER", ""),
		}
	default:
		return nil
	}
}

func getEnvDefault(key, fallback string) string {
	if v := strings.TrimSpace(lookupEnv(key)); v != "" {
		return v
	}
	return fallback
}

// lookupEnv wraps os.LookupEnv for testability
var lookupEnv = func(key string) string {
	v := os.Getenv(key)
	return v
}
