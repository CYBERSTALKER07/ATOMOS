// Package simulator provides a local-only Global Pay gateway mock.
//
// Routes mounted by main.go under /sim/globalpay when GLOBAL_PAY_ENV is
// "local" or "dev". The simulator exposes the same URL surface that the real
// checkout-api.globalpay.uz uses so that globalpayProviderExecutor can talk
// to it unchanged.
//
// IMPORTANT: never mount in production. The register guard checks GLOBAL_PAY_ENV.
package simulator

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// SimCard represents a simulated payment card.
type SimCard struct {
	ID       string
	Label    string
	Last4    string
	Balance  int64 // in minor units (UZS tiyin)
	Declined bool
}

var simCards = []SimCard{
	{ID: "sim-card-01", Label: "Visa · Balance 1,000,000 UZS", Last4: "4242", Balance: 100_000_000, Declined: false},
	{ID: "sim-card-02", Label: "Mastercard · Balance 500,000 UZS", Last4: "5555", Balance: 50_000_000, Declined: false},
	{ID: "sim-card-03", Label: "Declined Card (always fails)", Last4: "0002", Balance: 0, Declined: true},
	{ID: "sim-card-04", Label: "Zero Balance Card", Last4: "0001", Balance: 0, Declined: false},
}

// Handler holds the simulator state.
type Handler struct {
	webhookSecret  string
	backendBaseURL string // e.g. http://localhost:8080
	log            *slog.Logger

	// in-memory token store: userServiceToken -> tokenMeta
	tokens map[string]tokenMeta
}

type tokenMeta struct {
	orderID     string
	amountMinor int64
	currency    string
	issuedAt    time.Time
}

// NewHandler creates a simulator handler.
func NewHandler(webhookSecret, backendBaseURL string, log *slog.Logger) *Handler {
	if log == nil {
		log = slog.Default()
	}
	return &Handler{
		webhookSecret:  webhookSecret,
		backendBaseURL: strings.TrimRight(backendBaseURL, "/"),
		log:            log,
		tokens:         make(map[string]tokenMeta),
	}
}

// RegisterRoutes mounts the simulator under the given chi router prefix.
// Expected to be called with r.Route("/sim/globalpay", ...).
func RegisterRoutes(r chi.Router, h *Handler) {
	// Mirrors real Global Pay checkout-api URLs (relative to the sim prefix)
	r.Post("/v1/merchant/auth", h.handleAuth)
	r.Post("/v1/user-service-tokens", h.handleCreateToken)

	// Browser-facing checkout UI
	r.Get("/checkout", h.handleCheckoutUI)
	r.Post("/checkout/process", h.handleCheckoutProcess)

	// Health / list cards (dev helper)
	r.Get("/cards", h.handleListCards)

	// Backoffice capture (CP) — mirrors /payments/v2/payment/{id}/perform
	r.Post("/payments/v2/payment/{paymentID}/perform", h.handlePaymentPerform)
}

// ── Mock Auth ──────────────────────────────────────────────────────────────

type authRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) handleAuth(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	// Accept any non-empty credentials in sim mode.
	if strings.TrimSpace(req.Username) == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing_credentials"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"access_token": "sim-access-token-" + fmt.Sprintf("%d", time.Now().UnixMilli()),
	})
}

// ── Mock Token Creation ────────────────────────────────────────────────────

type createTokenRequest struct {
	ServiceID   string `json:"service_id"`
	OrderID     string `json:"order_id"`
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
}

func (h *Handler) handleCreateToken(w http.ResponseWriter, r *http.Request) {
	// Validate bearer token (sim: any non-empty value passes)
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") || len(authHeader) < 8 {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req createTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	if strings.TrimSpace(req.OrderID) == "" || req.AmountMinor <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id and amount_minor required"})
		return
	}

	token := "simtok_" + req.OrderID + "_" + fmt.Sprintf("%d", time.Now().UnixNano())
	h.tokens[token] = tokenMeta{
		orderID:     req.OrderID,
		amountMinor: req.AmountMinor,
		currency:    req.Currency,
		issuedAt:    time.Now().UTC(),
	}

	redirectURL := fmt.Sprintf("%s/sim/globalpay/checkout?token=%s", h.backendBaseURL, token)
	h.log.Info("[simulator] token created", "order_id", req.OrderID, "amount_minor", req.AmountMinor, "token", token)

	writeJSON(w, http.StatusOK, map[string]string{
		"token":           token,
		"userRedirectUrl": redirectURL,
	})
}

// ── Checkout UI ────────────────────────────────────────────────────────────

func (h *Handler) handleCheckoutUI(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	meta, ok := h.tokens[token]
	if !ok {
		http.Error(w, "Invalid or expired checkout token", http.StatusBadRequest)
		return
	}

	amountDisplay := fmt.Sprintf("%s %s", formatMinor(meta.amountMinor, meta.currency), meta.currency)
	html := buildCheckoutHTML(token, meta.orderID, amountDisplay)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(html))
}

func formatMinor(minor int64, currency string) string {
	switch strings.ToUpper(currency) {
	case "UZS":
		// UZS has no sub-unit, amount_minor == tiyin (1/100 sum), so divide by 100
		return fmt.Sprintf("%d", minor/100)
	default:
		return fmt.Sprintf("%.2f", float64(minor)/100)
	}
}

// ── Checkout Process ───────────────────────────────────────────────────────

type processRequest struct {
	Token  string `json:"token"`
	CardID string `json:"card_id"`
}

func (h *Handler) handleCheckoutProcess(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_form"})
		return
	}
	token := strings.TrimSpace(r.FormValue("token"))
	cardID := strings.TrimSpace(r.FormValue("card_id"))

	meta, ok := h.tokens[token]
	if !ok {
		http.Error(w, "Invalid or expired checkout token", http.StatusBadRequest)
		return
	}

	// Find card
	var selectedCard *SimCard
	for i, c := range simCards {
		if c.ID == cardID {
			selectedCard = &simCards[i]
			break
		}
	}
	if selectedCard == nil {
		http.Error(w, "Invalid card selection", http.StatusBadRequest)
		return
	}

	// Determine outcome
	status := "SUCCESS"
	if selectedCard.Declined || selectedCard.Balance < meta.amountMinor {
		status = "FAILED"
	}

	h.log.Info("[simulator] checkout process", "order_id", meta.orderID, "card", selectedCard.Last4, "status", status)

	// Fire webhook to backend
	go h.fireWebhook(meta, status, selectedCard)

	// Remove token (one-use)
	delete(h.tokens, token)

	// Render result page
	resultHTML := buildResultHTML(meta.orderID, status, selectedCard)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(resultHTML))
}

// fireWebhook sends a Global Pay-style webhook to the backend webhook handler.
func (h *Handler) fireWebhook(meta tokenMeta, status string, card *SimCard) {
	txID := fmt.Sprintf("sim-tx-%s-%d", meta.orderID, time.Now().UnixNano())

	payload := map[string]any{
		"transaction_id": txID,
		"order_id":       meta.orderID,
		"status":         status,
		"amount_minor":   meta.amountMinor,
		"currency":       meta.currency,
	}
	body, _ := json.Marshal(payload)

	// Compute HMAC-SHA256 signature (mirrors the real GlobalPay signature pattern)
	mac := hmac.New(sha256.New, []byte(h.webhookSecret))
	mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))

	url := h.backendBaseURL + "/v1/webhooks/global-pay"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		h.log.Error("[simulator] webhook build failed", "err", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GlobalPay-Signature", sig)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		h.log.Error("[simulator] webhook delivery failed", "err", err, "order_id", meta.orderID)
		return
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	h.log.Info("[simulator] webhook delivered", "status", resp.StatusCode, "order_id", meta.orderID, "body", string(respBody))
}

// ── Dev helpers ────────────────────────────────────────────────────────────

func (h *Handler) handleListCards(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"cards": simCards})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	b, _ := json.Marshal(v)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write(b)
}

// ── HTML Builders ──────────────────────────────────────────────────────────

func buildCheckoutHTML(token, orderID, amountDisplay string) string {
	var cardOptions strings.Builder
	for _, c := range simCards {
		balanceBadge := ""
		if c.Declined {
			balanceBadge = `<span class="badge declined">Will Decline</span>`
		} else if c.Balance == 0 {
			balanceBadge = `<span class="badge zero">Zero Balance</span>`
		}
		cardOptions.WriteString(fmt.Sprintf(`
		<label class="card-option">
			<input type="radio" name="card_id" value="%s" required>
			<div class="card-label">
				<span class="card-name">%s</span>
				<span class="card-last4">···· %s</span>
				%s
			</div>
		</label>`, c.ID, c.Label, c.Last4, balanceBadge))
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Global Pay · Simulated Checkout</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; background: #0f0f13; color: #e2e8f0; min-height: 100vh; display: flex; align-items: center; justify-content: center; }
  .container { background: #1a1a24; border: 1px solid #2a2a3a; border-radius: 16px; padding: 36px; width: 420px; max-width: 95vw; box-shadow: 0 25px 60px rgba(0,0,0,0.5); }
  .header { text-align: center; margin-bottom: 28px; }
  .logo { display: inline-flex; align-items: center; gap: 8px; margin-bottom: 12px; }
  .logo-mark { width: 36px; height: 36px; background: linear-gradient(135deg, #6366f1, #8b5cf6); border-radius: 8px; display: flex; align-items: center; justify-content: center; font-weight: 800; font-size: 16px; color: white; }
  .logo-text { font-size: 18px; font-weight: 700; color: #c7d2fe; }
  .sim-badge { background: #f59e0b22; color: #f59e0b; border: 1px solid #f59e0b44; border-radius: 999px; font-size: 11px; font-weight: 600; padding: 2px 10px; letter-spacing: 0.05em; display: inline-block; margin-bottom: 16px; }
  h1 { font-size: 22px; font-weight: 700; color: #f1f5f9; }
  .order-info { background: #12121a; border: 1px solid #2a2a3a; border-radius: 10px; padding: 14px 18px; margin-bottom: 24px; display: flex; justify-content: space-between; align-items: center; }
  .order-label { font-size: 12px; color: #64748b; font-weight: 500; text-transform: uppercase; letter-spacing: 0.06em; }
  .order-value { font-size: 13px; color: #94a3b8; font-weight: 600; font-family: monospace; }
  .amount-value { font-size: 24px; font-weight: 800; color: #6366f1; }
  .section-label { font-size: 12px; font-weight: 600; color: #64748b; text-transform: uppercase; letter-spacing: 0.06em; margin-bottom: 12px; }
  .card-option { display: flex; align-items: center; gap: 12px; background: #12121a; border: 1.5px solid #2a2a3a; border-radius: 10px; padding: 14px; margin-bottom: 10px; cursor: pointer; transition: border-color 0.15s; }
  .card-option:hover { border-color: #4f4f7a; }
  .card-option input[type="radio"] { accent-color: #6366f1; width: 16px; height: 16px; flex-shrink: 0; }
  .card-option input[type="radio"]:checked + .card-label { color: #c7d2fe; }
  .card-label { display: flex; flex-direction: column; gap: 4px; flex: 1; }
  .card-name { font-size: 14px; font-weight: 600; color: #e2e8f0; }
  .card-last4 { font-size: 12px; color: #64748b; font-family: monospace; }
  .badge { font-size: 10px; font-weight: 700; border-radius: 999px; padding: 2px 8px; letter-spacing: 0.04em; display: inline-block; width: fit-content; }
  .badge.declined { background: #ef444422; color: #ef4444; border: 1px solid #ef444444; }
  .badge.zero { background: #f59e0b22; color: #f59e0b; border: 1px solid #f59e0b44; }
  .pay-btn { width: 100%%; padding: 14px; background: linear-gradient(135deg, #6366f1, #8b5cf6); color: white; font-size: 16px; font-weight: 700; border: none; border-radius: 10px; cursor: pointer; margin-top: 20px; transition: opacity 0.15s, transform 0.1s; letter-spacing: 0.01em; }
  .pay-btn:hover { opacity: 0.9; transform: translateY(-1px); }
  .pay-btn:active { transform: translateY(0); }
  .footer { text-align: center; font-size: 11px; color: #334155; margin-top: 16px; }
</style>
</head>
<body>
<div class="container">
  <div class="header">
    <div class="logo">
      <div class="logo-mark">GP</div>
      <span class="logo-text">Global Pay</span>
    </div>
    <div class="sim-badge">&#x26A1; LOCAL SIMULATION MODE</div>
    <h1>Complete Payment</h1>
  </div>
  <div class="order-info">
    <div>
      <div class="order-label">Order</div>
      <div class="order-value">%s</div>
    </div>
    <div style="text-align:right">
      <div class="order-label">Amount</div>
      <div class="amount-value">%s</div>
    </div>
  </div>
  <div class="section-label">Select Payment Card</div>
  <form method="POST" action="/sim/globalpay/checkout/process">
    <input type="hidden" name="token" value="%s">
    %s
    <button type="submit" class="pay-btn">Pay Now &#x2192;</button>
  </form>
  <p class="footer">This is a local simulator. No real money is moved.</p>
</div>
</body>
</html>`, orderID, amountDisplay, token, cardOptions.String())
	return html
}

func buildResultHTML(orderID, status string, card *SimCard) string {
	icon := "&#x2705;"
	title := "Payment Successful"
	subtitle := "Your payment was processed successfully. The order will now advance."
	color := "#22c55e"
	if status != "SUCCESS" {
		icon = "&#x274C;"
		title = "Payment Failed"
		subtitle = "The card was declined or had insufficient balance. Please try a different card."
		color = "#ef4444"
	}
	// Use string concatenation to avoid fmt.Sprintf %s22/%s44 ambiguity.
	css := `
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; background: #0f0f13; color: #e2e8f0; min-height: 100vh; display: flex; align-items: center; justify-content: center; }
  .container { background: #1a1a24; border: 1px solid #2a2a3a; border-radius: 16px; padding: 48px 36px; width: 420px; max-width: 95vw; text-align: center; box-shadow: 0 25px 60px rgba(0,0,0,0.5); }
  .icon { font-size: 56px; margin-bottom: 20px; }
  h1 { font-size: 24px; font-weight: 800; color: ` + color + `; margin-bottom: 12px; }
  p { font-size: 14px; color: #64748b; line-height: 1.6; margin-bottom: 8px; }
  .meta { background: #12121a; border: 1px solid #2a2a3a; border-radius: 10px; padding: 14px; margin: 20px 0; }
  .meta-row { display: flex; justify-content: space-between; font-size: 13px; padding: 4px 0; }
  .meta-key { color: #64748b; }
  .meta-val { color: #94a3b8; font-weight: 600; font-family: monospace; }
  .status-badge { display: inline-block; padding: 4px 14px; border-radius: 999px; font-size: 12px; font-weight: 700; background: ` + color + `22; color: ` + color + `; border: 1px solid ` + color + `44; letter-spacing: 0.05em; }
  .close-note { font-size: 12px; color: #334155; margin-top: 24px; }`
	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>Global Pay · ` + title + `</title>
<style>` + css + `
</style>
</head>
<body>
<div class="container">
  <div class="icon">` + icon + `</div>
  <h1>` + title + `</h1>
  <p>` + subtitle + `</p>
  <div class="meta">
    <div class="meta-row"><span class="meta-key">Order ID</span><span class="meta-val">` + orderID + `</span></div>
    <div class="meta-row"><span class="meta-key">Card</span><span class="meta-val">` + "····" + ` ` + card.Last4 + `</span></div>
    <div class="meta-row"><span class="meta-key">Status</span><span class="status-badge">` + status + `</span></div>
  </div>
  <p class="close-note">You may close this window. The webhook has been delivered to the backend.</p>
</div>
</body>
</html>`
}

func (h *Handler) handlePaymentPerform(w http.ResponseWriter, r *http.Request) {
	paymentID := strings.TrimSpace(chi.URLParam(r, "paymentID"))
	if paymentID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "payment_id_required"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"paymentId":    paymentID,
		"status":       "SUCCESS",
		"paid":         true,
		"isSuccessful": true,
	})
}
