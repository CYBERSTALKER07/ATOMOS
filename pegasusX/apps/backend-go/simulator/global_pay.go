// Package simulator provides a local-only Global Pay gateway mock and replica.
//
// Routes mounted by main.go under /sim/globalpay when GLOBAL_PAY_ENV is
// "local" or "dev". The simulator exposes the exact URL surface that the real
// checkout-api.globalpay.uz and backoffice-api.globalpay.uz use so that
// globalpayProviderExecutor can talk to it unchanged.
//
// In addition, this provides an authentic replica of GlobalPay.uz:
// - Official Uzbek branding (#00C389 theme, Central Bank license badge №20)
// - Dynamic PAN format, auto BIN detection (UzCard 8600/5614, Humo 9860, Visa 4, Mastercard 5)
// - SMS OTP verification modal (auto-fill 123456)
// - P2P Money Transfers (card-to-card) with dynamic balance adjustments and fee calculation (0.5%)
// - Card tokenization & direct payment endpoints
// - Thread-safe balance ledger (minor units / tiyin)
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
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
)

// SimCard represents a simulated payment card in the GlobalPay system.
type SimCard struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	PAN       string `json:"pan"`
	Last4     string `json:"last4"`
	Expiry    string `json:"expiry"`
	Holder    string `json:"holder"`
	Scheme    string `json:"scheme"` // UZCARD, HUMO, VISA, MASTERCARD
	Balance   int64  `json:"balance"` // in minor units (UZS tiyin, 1 UZS = 100 tiyin)
	Declined  bool   `json:"declined"`
	PhoneMask string `json:"phone_mask"`
	BankName  string `json:"bank_name"`
}

// Initial default cards with realistic balances in Uzbekistan.
var defaultCards = []SimCard{
	{
		ID:        "sim-card-01",
		Label:     "UzCard · Alisher Navoiy (10,000,000 UZS)",
		PAN:       "8600123456780001",
		Last4:     "0001",
		Expiry:    "12/28",
		Holder:    "ALISHER NAVOIY",
		Scheme:    "UZCARD",
		Balance:   1_000_000_000, // 10,000,000 UZS
		Declined:  false,
		PhoneMask: "+998 90 *** ** 01",
		BankName:  "O'zsanoatqurilishbank",
	},
	{
		ID:        "sim-card-02",
		Label:     "Humo · Zulfiya Isroilova (50,000,000 UZS)",
		PAN:       "9860123456780002",
		Last4:     "0002",
		Expiry:    "08/27",
		Holder:    "ZULFIYA ISROILOVA",
		Scheme:    "HUMO",
		Balance:   5_000_000_000, // 50,000,000 UZS
		Declined:  false,
		PhoneMask: "+998 93 *** ** 02",
		BankName:  "Ipak Yo'li Bank",
	},
	{
		ID:        "sim-card-03",
		Label:     "UzCard · Declined Card (always fails)",
		PAN:       "8600999999999999",
		Last4:     "9999",
		Expiry:    "12/25",
		Holder:    "BLOCKED CARDHOLDER",
		Scheme:    "UZCARD",
		Balance:   100_000_000,
		Declined:  true,
		PhoneMask: "+998 90 *** ** 99",
		BankName:  "Agrobank",
	},
	{
		ID:        "sim-card-04",
		Label:     "UzCard · Zero Balance Card",
		PAN:       "8600000000000000",
		Last4:     "0000",
		Expiry:    "01/26",
		Holder:    "TEMUR MALIK",
		Scheme:    "UZCARD",
		Balance:   0,
		Declined:  false,
		PhoneMask: "+998 90 *** ** 00",
		BankName:  "Xalq Banki",
	},
	{
		ID:        "sim-card-05",
		Label:     "Visa Classic · Bobur Mirzo (25,000,000 UZS)",
		PAN:       "4242424242424242",
		Last4:     "4242",
		Expiry:    "05/29",
		Holder:    "BOBUR MIRZO",
		Scheme:    "VISA",
		Balance:   2_500_000_000, // 25,000,000 UZS
		Declined:  false,
		PhoneMask: "+998 97 *** ** 42",
		BankName:  "Kapitalbank",
	},
	{
		ID:        "sim-card-06",
		Label:     "Mastercard World · Nodira Begim (60,000,000 UZS)",
		PAN:       "5555555555555555",
		Last4:     "5555",
		Expiry:    "11/26",
		Holder:    "NODIRA BEGIM",
		Scheme:    "MASTERCARD",
		Balance:   6_000_000_000, // 60,000,000 UZS
		Declined:  false,
		PhoneMask: "+998 99 *** ** 55",
		BankName:  "NBU (Milliy Bank)",
	},
	{
		ID:        "sim-card-07",
		Label:     "UzCard · Jasur Qodirov Driver (3,500,000 UZS)",
		PAN:       "8600111122223333",
		Last4:     "3333",
		Expiry:    "03/27",
		Holder:    "JASUR QODIROV",
		Scheme:    "UZCARD",
		Balance:   350_000_000, // 3,500,000 UZS
		Declined:  false,
		PhoneMask: "+998 91 *** ** 33",
		BankName:  "Hamkorbank",
	},
}

// P2PTransfer represents a card-to-card money transfer transaction.
type P2PTransfer struct {
	TransferID       string    `json:"transfer_id"`
	SenderCardID     string    `json:"sender_card_id"`
	SenderPAN        string    `json:"sender_pan_masked"`
	RecipientCardID  string    `json:"recipient_card_id"`
	RecipientPAN     string    `json:"recipient_pan_masked"`
	RecipientHolder  string    `json:"recipient_holder"`
	AmountMinor      int64     `json:"amount_minor"`
	FeeMinor         int64     `json:"fee_minor"`
	TotalMinor       int64     `json:"total_minor"`
	Currency         string    `json:"currency"`
	Status           string    `json:"status"` // PENDING_OTP, COMPLETED, FAILED
	OTPCode          string    `json:"otp_code"`
	RRN              string    `json:"rrn"`
	AuthCode         string    `json:"auth_code"`
	CreatedAt        time.Time `json:"created_at"`
	CompletedAt      *time.Time`json:"completed_at,omitempty"`
}

// CardBinding represents an in-flight card tokenization session.
type CardBinding struct {
	BindingID string    `json:"binding_id"`
	Card      *SimCard  `json:"card"`
	OTPCode   string    `json:"otp_code"`
	CreatedAt time.Time `json:"created_at"`
}

// Handler holds the simulator state and dynamic balance ledger.
type Handler struct {
	webhookSecret  string
	backendBaseURL string
	log            *slog.Logger

	mu         sync.RWMutex
	cards      map[string]*SimCard    // key: ID and clean PAN
	cardList   []*SimCard             // ordered list of unique cards
	tokens     map[string]tokenMeta   // checkout tokens
	transfers  map[string]*P2PTransfer
	bindings   map[string]*CardBinding
	cardTokens map[string]*SimCard    // gp_tok_... -> card
}

type tokenMeta struct {
	orderID     string
	amountMinor int64
	currency    string
	issuedAt    time.Time
}

// NewHandler creates an authentic GlobalPay.uz replica simulator handler.
func NewHandler(webhookSecret, backendBaseURL string, log *slog.Logger) *Handler {
	if log == nil {
		log = slog.Default()
	}
	h := &Handler{
		webhookSecret:  webhookSecret,
		backendBaseURL: strings.TrimRight(backendBaseURL, "/"),
		log:            log,
		cards:          make(map[string]*SimCard),
		tokens:         make(map[string]tokenMeta),
		transfers:      make(map[string]*P2PTransfer),
		bindings:       make(map[string]*CardBinding),
		cardTokens:     make(map[string]*SimCard),
	}

	// Initialize default cards
	for _, c := range defaultCards {
		cardCopy := c
		h.cardList = append(h.cardList, &cardCopy)
		h.cards[cardCopy.ID] = &cardCopy
		h.cards[cardCopy.PAN] = &cardCopy
	}

	return h
}

// RegisterRoutes mounts the GlobalPay replica under the chi router prefix.
func RegisterRoutes(r chi.Router, h *Handler) {
	// ── Standard GlobalPay Gateway Endpoints ──────────────────────────────
	r.Post("/v1/merchant/auth", h.handleAuth)
	r.Post("/v1/user-service-tokens", h.handleCreateToken)

	// ── Interactive Web Checkout UI ───────────────────────────────────────
	r.Get("/checkout", h.handleCheckoutUI)
	r.Post("/checkout/process", h.handleCheckoutProcess)

	// ── P2P Money Transfers (Card-to-Card) ────────────────────────────────
	r.Get("/transfer", h.handleTransferUI)
	r.Post("/v1/transfers/p2p", h.handleP2PTransferInit)
	r.Post("/v1/transfers/p2p/confirm", h.handleP2PTransferConfirm)
	r.Get("/v1/transfers/{transferID}", h.handleP2PTransferGet)

	// ── Card Tokenization & Direct Pay ────────────────────────────────────
	r.Post("/v1/cards/new", h.handleCardTokenNew)
	r.Post("/v1/cards/bind", h.handleCardTokenBind)
	r.Post("/v1/payments/token", h.handlePaymentTokenDirect)

	// ── Dev & Discovery Helpers ───────────────────────────────────────────
	r.Get("/cards", h.handleListCards)
	r.Post("/v1/cards/lookup", h.handleCardLookup)

	// ── Backoffice Perform & Query ────────────────────────────────────────
	r.Post("/payments/v2/payment/{paymentID}/perform", h.handlePaymentPerform)
	r.Get("/payments/v2/payment/{paymentID}", h.handlePaymentStatusCheck)
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

	h.mu.Lock()
	token := "simtok_" + req.OrderID + "_" + fmt.Sprintf("%d", time.Now().UnixNano())
	h.tokens[token] = tokenMeta{
		orderID:     req.OrderID,
		amountMinor: req.AmountMinor,
		currency:    req.Currency,
		issuedAt:    time.Now().UTC(),
	}
	h.mu.Unlock()

	redirectURL := fmt.Sprintf("%s/sim/globalpay/checkout?token=%s", h.backendBaseURL, token)
	h.log.Info("[simulator] token created", "order_id", req.OrderID, "amount_minor", req.AmountMinor, "token", token)

	writeJSON(w, http.StatusOK, map[string]string{
		"token":           token,
		"userRedirectUrl": redirectURL,
	})
}

// ── Card BIN & Scheme Detection ───────────────────────────────────────────

func detectScheme(pan string) string {
	clean := strings.ReplaceAll(pan, " ", "")
	clean = strings.ReplaceAll(clean, "-", "")
	if strings.HasPrefix(clean, "8600") || strings.HasPrefix(clean, "5614") {
		return "UZCARD"
	}
	if strings.HasPrefix(clean, "9860") {
		return "HUMO"
	}
	if strings.HasPrefix(clean, "4") {
		return "VISA"
	}
	if strings.HasPrefix(clean, "51") || strings.HasPrefix(clean, "52") ||
		strings.HasPrefix(clean, "53") || strings.HasPrefix(clean, "54") ||
		strings.HasPrefix(clean, "55") || strings.HasPrefix(clean, "22") {
		return "MASTERCARD"
	}
	if strings.HasPrefix(clean, "62") {
		return "UNIONPAY"
	}
	return "UNKNOWN"
}

// ── Card Lookup ───────────────────────────────────────────────────────────

func (h *Handler) handleCardLookup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PAN string `json:"pan"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	clean := strings.ReplaceAll(strings.ReplaceAll(req.PAN, " ", ""), "-", "")

	h.mu.RLock()
	card, ok := h.cards[clean]
	h.mu.RUnlock()

	if !ok {
		// Provide synthetic masked holder for unknown cards in Uzbekistan
		scheme := detectScheme(clean)
		writeJSON(w, http.StatusOK, map[string]any{
			"pan":               maskPAN(clean),
			"scheme":            scheme,
			"cardholder_masked": "MIJOZ K.",
			"bank_name":         "O'zbekiston Tijorat Banki",
			"is_valid":          len(clean) == 16,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"pan":               maskPAN(card.PAN),
		"scheme":            card.Scheme,
		"cardholder_masked": maskHolder(card.Holder),
		"bank_name":         card.BankName,
		"is_valid":          !card.Declined,
	})
}

func maskPAN(pan string) string {
	if len(pan) < 8 {
		return pan
	}
	return pan[:4] + " •••• •••• " + pan[len(pan)-4:]
}

func maskHolder(holder string) string {
	parts := strings.Fields(holder)
	if len(parts) >= 2 {
		return parts[0] + " " + string(parts[1][0]) + "."
	}
	return holder
}

// ── P2P Money Transfers ───────────────────────────────────────────────────

func (h *Handler) handleP2PTransferInit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SenderCardID string `json:"sender_card_id"`
		SenderPAN    string `json:"sender_pan"`
		RecipientPAN string `json:"recipient_pan"`
		AmountMinor  int64  `json:"amount_minor"`
		Currency     string `json:"currency"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	cleanSender := strings.ReplaceAll(strings.ReplaceAll(req.SenderPAN, " ", ""), "-", "")
	cleanRecipient := strings.ReplaceAll(strings.ReplaceAll(req.RecipientPAN, " ", ""), "-", "")
	if req.AmountMinor <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "amount_minor must be positive"})
		return
	}
	if req.Currency == "" {
		req.Currency = "UZS"
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	var sender *SimCard
	if req.SenderCardID != "" {
		sender = h.cards[req.SenderCardID]
	} else if cleanSender != "" {
		sender = h.cards[cleanSender]
	}
	if sender == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "sender_card_not_found"})
		return
	}
	if sender.Declined {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "sender_card_blocked"})
		return
	}

	// Calculate 0.5% commission fee (min 1,000 UZS = 100,000 tiyin)
	feeMinor := (req.AmountMinor * 5) / 1000
	if feeMinor < 100_000 {
		feeMinor = 100_000
	}
	totalMinor := req.AmountMinor + feeMinor

	if sender.Balance < totalMinor {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":         "insufficient_funds",
			"balance_minor": fmt.Sprintf("%d", sender.Balance),
			"needed_minor":  fmt.Sprintf("%d", totalMinor),
		})
		return
	}

	recipientHolder := "MIJOZ T."
	var recipientCardID string
	if recCard, ok := h.cards[cleanRecipient]; ok {
		recipientHolder = maskHolder(recCard.Holder)
		recipientCardID = recCard.ID
	}

	transferID := fmt.Sprintf("p2p_%d", time.Now().UnixNano())
	transfer := &P2PTransfer{
		TransferID:      transferID,
		SenderCardID:    sender.ID,
		SenderPAN:       maskPAN(sender.PAN),
		RecipientCardID: recipientCardID,
		RecipientPAN:    maskPAN(cleanRecipient),
		RecipientHolder: recipientHolder,
		AmountMinor:     req.AmountMinor,
		FeeMinor:        feeMinor,
		TotalMinor:      totalMinor,
		Currency:        req.Currency,
		Status:          "PENDING_OTP",
		OTPCode:         "123456",
		CreatedAt:       time.Now().UTC(),
	}

	h.transfers[transferID] = transfer

	writeJSON(w, http.StatusOK, map[string]any{
		"transfer_id":      transferID,
		"sender_card":      sender.Label,
		"recipient_pan":    maskPAN(cleanRecipient),
		"recipient_holder": recipientHolder,
		"amount_minor":     req.AmountMinor,
		"fee_minor":        feeMinor,
		"total_minor":      totalMinor,
		"currency":         req.Currency,
		"sms_sent":         true,
		"phone_mask":       sender.PhoneMask,
		"otp_hint":         "123456",
		"message":          "SMS OTP verification code sent to cardholder",
	})
}

func (h *Handler) handleP2PTransferConfirm(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TransferID string `json:"transfer_id"`
		OTPCode    string `json:"otp_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	transfer, ok := h.transfers[req.TransferID]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "transfer_not_found"})
		return
	}
	if transfer.Status != "PENDING_OTP" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "transfer_already_processed", "status": transfer.Status})
		return
	}
	if req.OTPCode != transfer.OTPCode {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_otp_code"})
		return
	}

	sender := h.cards[transfer.SenderCardID]
	if sender == nil || sender.Balance < transfer.TotalMinor {
		transfer.Status = "FAILED"
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "insufficient_funds_at_confirmation"})
		return
	}

	// Atomic balance adjustment
	sender.Balance -= transfer.TotalMinor
	if transfer.RecipientCardID != "" {
		if recCard := h.cards[transfer.RecipientCardID]; recCard != nil {
			recCard.Balance += transfer.AmountMinor
		}
	}

	now := time.Now().UTC()
	transfer.Status = "COMPLETED"
	transfer.CompletedAt = &now
	transfer.RRN = fmt.Sprintf("GP%d", time.Now().UnixMilli())
	transfer.AuthCode = fmt.Sprintf("%06d", (time.Now().UnixNano()/1000)%1000000)

	h.log.Info("[simulator] P2P transfer completed",
		"transfer_id", transfer.TransferID,
		"amount_minor", transfer.AmountMinor,
		"fee_minor", transfer.FeeMinor,
		"sender", sender.PAN,
		"sender_new_balance", sender.Balance)

	writeJSON(w, http.StatusOK, map[string]any{
		"transfer_id":      transfer.TransferID,
		"status":           transfer.Status,
		"amount_minor":     transfer.AmountMinor,
		"fee_minor":        transfer.FeeMinor,
		"total_minor":      transfer.TotalMinor,
		"currency":         transfer.Currency,
		"rrn":              transfer.RRN,
		"auth_code":        transfer.AuthCode,
		"sender_balance":   sender.Balance,
		"completed_at":     transfer.CompletedAt,
		"recipient_holder": transfer.RecipientHolder,
	})
}

func (h *Handler) handleP2PTransferGet(w http.ResponseWriter, r *http.Request) {
	transferID := chi.URLParam(r, "transferID")
	h.mu.RLock()
	transfer, ok := h.transfers[transferID]
	h.mu.RUnlock()

	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "transfer_not_found"})
		return
	}
	writeJSON(w, http.StatusOK, transfer)
}

// ── Card Tokenization ─────────────────────────────────────────────────────

func (h *Handler) handleCardTokenNew(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PAN    string `json:"pan"`
		Expiry string `json:"expiry"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	clean := strings.ReplaceAll(strings.ReplaceAll(req.PAN, " ", ""), "-", "")

	h.mu.Lock()
	defer h.mu.Unlock()

	card, ok := h.cards[clean]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "card_not_found"})
		return
	}

	bindingID := fmt.Sprintf("bind_%d", time.Now().UnixNano())
	h.bindings[bindingID] = &CardBinding{
		BindingID: bindingID,
		Card:      card,
		OTPCode:   "123456",
		CreatedAt: time.Now().UTC(),
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"binding_id": bindingID,
		"sms_sent":   true,
		"phone_mask": card.PhoneMask,
		"otp_hint":   "123456",
	})
}

func (h *Handler) handleCardTokenBind(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BindingID string `json:"binding_id"`
		OTPCode   string `json:"otp_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	binding, ok := h.bindings[req.BindingID]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "binding_not_found"})
		return
	}
	if req.OTPCode != binding.OTPCode {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_otp_code"})
		return
	}

	token := fmt.Sprintf("gp_tok_%s_%d", binding.Card.Last4, time.Now().UnixNano())
	h.cardTokens[token] = binding.Card
	delete(h.bindings, req.BindingID)

	writeJSON(w, http.StatusOK, map[string]any{
		"card_token": token,
		"pan_masked": maskPAN(binding.Card.PAN),
		"scheme":     binding.Card.Scheme,
		"holder":     binding.Card.Holder,
	})
}

func (h *Handler) handlePaymentTokenDirect(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CardToken   string `json:"card_token"`
		OrderID     string `json:"order_id"`
		AmountMinor int64  `json:"amount_minor"`
		Currency    string `json:"currency"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	h.mu.Lock()
	card, ok := h.cardTokens[req.CardToken]
	if !ok {
		h.mu.Unlock()
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "token_not_found"})
		return
	}

	status := "SUCCESS"
	if card.Declined || card.Balance < req.AmountMinor {
		status = "FAILED"
	} else {
		card.Balance -= req.AmountMinor
	}
	h.mu.Unlock()

	meta := tokenMeta{
		orderID:     req.OrderID,
		amountMinor: req.AmountMinor,
		currency:    req.Currency,
		issuedAt:    time.Now().UTC(),
	}
	go h.fireWebhook(meta, status, card)

	writeJSON(w, http.StatusOK, map[string]any{
		"order_id":     req.OrderID,
		"status":       status,
		"amount_minor": req.AmountMinor,
		"card_masked":  maskPAN(card.PAN),
		"scheme":       card.Scheme,
		"isSuccessful": status == "SUCCESS",
	})
}

// ── Checkout UI ────────────────────────────────────────────────────────────

func (h *Handler) handleCheckoutUI(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	h.mu.RLock()
	meta, ok := h.tokens[token]
	h.mu.RUnlock()

	if !ok {
		http.Error(w, "Invalid or expired checkout token", http.StatusBadRequest)
		return
	}

	amountDisplay := fmt.Sprintf("%s %s", formatMinor(meta.amountMinor, meta.currency), meta.currency)
	h.mu.RLock()
	cardsCopy := make([]SimCard, len(h.cardList))
	for i, c := range h.cardList {
		cardsCopy[i] = *c
	}
	h.mu.RUnlock()

	html := buildCheckoutHTML(token, meta.orderID, amountDisplay, cardsCopy)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(html))
}

func formatMinor(minor int64, currency string) string {
	switch strings.ToUpper(currency) {
	case "UZS":
		// Format in Uzbek Sums: 100 tiyin = 1 sum, with space thousands separator
		sums := minor / 100
		return formatNumberSpace(sums)
	default:
		return fmt.Sprintf("%.2f", float64(minor)/100)
	}
}

func formatNumberSpace(n int64) string {
	str := fmt.Sprintf("%d", n)
	if len(str) <= 3 {
		return str
	}
	var res []string
	rem := len(str) % 3
	if rem > 0 {
		res = append(res, str[:rem])
	}
	for i := rem; i < len(str); i += 3 {
		res = append(res, str[i:i+3])
	}
	return strings.Join(res, " ")
}

// ── Checkout Process ───────────────────────────────────────────────────────

func (h *Handler) handleCheckoutProcess(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_form"})
		return
	}
	token := strings.TrimSpace(r.FormValue("token"))
	cardID := strings.TrimSpace(r.FormValue("card_id"))
	otpCode := strings.TrimSpace(r.FormValue("otp_code"))

	h.mu.Lock()
	meta, ok := h.tokens[token]
	if !ok {
		h.mu.Unlock()
		http.Error(w, "Invalid or expired checkout token", http.StatusBadRequest)
		return
	}

	// Find card by ID or PAN
	cleanCardID := strings.ReplaceAll(strings.ReplaceAll(cardID, " ", ""), "-", "")
	selectedCard := h.cards[cardID]
	if selectedCard == nil {
		selectedCard = h.cards[cleanCardID]
	}
	if selectedCard == nil {
		h.mu.Unlock()
		http.Error(w, "Invalid card selection", http.StatusBadRequest)
		return
	}

	// Determine outcome & verify OTP if submitted
	status := "SUCCESS"
	if selectedCard.Declined || selectedCard.Balance < meta.amountMinor {
		status = "FAILED"
	} else if otpCode != "" && otpCode != "123456" {
		status = "FAILED"
	} else {
		// Debit card balance
		selectedCard.Balance -= meta.amountMinor
	}

	h.log.Info("[simulator] checkout process",
		"order_id", meta.orderID,
		"card", selectedCard.Last4,
		"scheme", selectedCard.Scheme,
		"status", status,
		"remaining_balance", selectedCard.Balance)

	// Remove token (one-use)
	delete(h.tokens, token)
	cardCopy := *selectedCard
	h.mu.Unlock()

	// Fire webhook to backend
	go h.fireWebhook(meta, status, &cardCopy)

	// Render result page
	resultHTML := buildResultHTML(meta.orderID, status, &cardCopy, meta.amountMinor, meta.currency)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(resultHTML))
}

// ── P2P Transfer UI ───────────────────────────────────────────────────────

func (h *Handler) handleTransferUI(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	cardsCopy := make([]SimCard, len(h.cardList))
	for i, c := range h.cardList {
		cardsCopy[i] = *c
	}
	h.mu.RUnlock()

	html := buildP2PTransferHTML(cardsCopy)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(html))
}

// ── Webhook Dispatch ──────────────────────────────────────────────────────

func (h *Handler) fireWebhook(meta tokenMeta, status string, card *SimCard) {
	txID := fmt.Sprintf("sim-tx-%s-%d", meta.orderID, time.Now().UnixNano())

	payload := map[string]any{
		"transaction_id": txID,
		"order_id":       meta.orderID,
		"status":         status,
		"amount_minor":   meta.amountMinor,
		"currency":       meta.currency,
		"pan_masked":     maskPAN(card.PAN),
		"scheme":         card.Scheme,
		"rrn":            fmt.Sprintf("RRN%d", time.Now().UnixMilli()),
		"auth_code":      "123456",
	}
	body, _ := json.Marshal(payload)

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
	h.mu.RLock()
	defer h.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"cards":  h.cardList,
		"count":  len(h.cardList),
		"status": "ready",
	})
}

func (h *Handler) handlePaymentPerform(w http.ResponseWriter, r *http.Request) {
	paymentID := strings.TrimSpace(chi.URLParam(r, "paymentID"))
	if paymentID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "payment_id_required"})
		return
	}
	var body struct {
		Action      string `json:"action"`
		AmountMinor int64  `json:"amount_minor"`
		Currency    string `json:"currency"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	action := strings.ToUpper(strings.TrimSpace(body.Action))
	if action == "" {
		action = "CP"
	}
	switch action {
	case "CP", "RF":
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported_action", "action": action})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"paymentId":    paymentID,
		"status":       "SUCCESS",
		"paid":         action == "CP",
		"refunded":     action == "RF",
		"action":       action,
		"isSuccessful": true,
	})
}

func (h *Handler) handlePaymentStatusCheck(w http.ResponseWriter, r *http.Request) {
	paymentID := strings.TrimSpace(chi.URLParam(r, "paymentID"))
	writeJSON(w, http.StatusOK, map[string]any{
		"paymentId":    paymentID,
		"status":       "SUCCESS",
		"paid":         true,
		"isSuccessful": true,
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	b, _ := json.Marshal(v)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write(b)
}

// ── HTML Builders (Authentic GlobalPay.uz Replica) ─────────────────────────

func buildCheckoutHTML(token, orderID, amountDisplay string, cards []SimCard) string {
	var cardRows strings.Builder
	for _, c := range cards {
		badgeClass := "badge-scheme"
		badgeText := c.Scheme
		if c.Declined {
			badgeClass = "badge-declined"
			badgeText = "Bloklangan / Declined"
		} else if c.Balance == 0 {
			badgeClass = "badge-zero"
			badgeText = "0 UZS Balans"
		}

		balanceFormatted := formatNumberSpace(c.Balance / 100)
		cardRows.WriteString(fmt.Sprintf(`
		<div class="card-item" onclick="selectCard('%s', '%s', '%s', '%s', '%s')">
			<div class="card-item-left">
				<div class="scheme-icon scheme-%s">%s</div>
				<div class="card-info">
					<div class="card-holder">%s</div>
					<div class="card-pan">%s</div>
				</div>
			</div>
			<div class="card-item-right">
				<div class="card-bal">%s so'm</div>
				<span class="badge %s">%s</span>
			</div>
		</div>`, c.ID, c.PAN, c.Expiry, c.Holder, c.Scheme,
			strings.ToLower(c.Scheme), c.Scheme[:2],
			c.Holder, maskPAN(c.PAN),
			balanceFormatted, badgeClass, badgeText))
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="uz">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>GlobalPay · Xavfsiz To'lov Gateway</title>
<style>
  :root {
    --gp-primary: #00C389;
    --gp-primary-dark: #009E6D;
    --gp-bg: #0D1117;
    --gp-card: #161B22;
    --gp-border: #30363D;
    --gp-text: #F0F6FC;
    --gp-muted: #8B949E;
  }
  * { box-sizing: border-box; margin: 0; padding: 0; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; }
  body { background: var(--gp-bg); color: var(--gp-text); min-height: 100vh; display: flex; align-items: center; justify-content: center; padding: 20px; }
  .checkout-container { width: 100%%; max-width: 460px; background: var(--gp-card); border: 1px solid var(--gp-border); border-radius: 20px; box-shadow: 0 30px 80px rgba(0,0,0,0.6); overflow: hidden; }
  .top-banner { background: #1C2128; border-bottom: 1px solid var(--gp-border); padding: 16px 24px; display: flex; justify-content: space-between; align-items: center; }
  .brand-logo { display: flex; align-items: center; gap: 10px; }
  .brand-mark { width: 34px; height: 34px; background: linear-gradient(135deg, #00C389, #007A55); border-radius: 9px; display: flex; align-items: center; justify-content: center; font-weight: 900; font-size: 16px; color: #fff; box-shadow: 0 4px 12px rgba(0,195,137,0.3); }
  .brand-name { font-size: 19px; font-weight: 800; color: #fff; letter-spacing: -0.5px; }
  .brand-dot { color: var(--gp-primary); }
  .license-badge { font-size: 10px; color: var(--gp-muted); background: #21262D; border: 1px solid var(--gp-border); border-radius: 6px; padding: 4px 8px; text-align: right; line-height: 1.2; }
  .body-content { padding: 24px; }
  .amount-box { background: #21262D; border: 1px solid var(--gp-border); border-radius: 14px; padding: 16px 20px; margin-bottom: 20px; display: flex; justify-content: space-between; align-items: center; }
  .order-meta-label { font-size: 11px; text-transform: uppercase; color: var(--gp-muted); font-weight: 600; letter-spacing: 0.5px; }
  .order-id-val { font-size: 13px; color: var(--gp-text); font-weight: 600; font-family: ui-monospace, monospace; margin-top: 2px; }
  .amount-val { font-size: 22px; font-weight: 800; color: var(--gp-primary); text-align: right; }
  .credit-card-preview { background: linear-gradient(135deg, #1E2630, #141A22); border: 1px solid #3B434D; border-radius: 14px; padding: 20px; margin-bottom: 24px; position: relative; box-shadow: inset 0 1px 0 rgba(255,255,255,0.1); }
  .card-chip { width: 38px; height: 26px; background: linear-gradient(135deg, #FFD700, #B8860B); border-radius: 6px; margin-bottom: 16px; opacity: 0.85; }
  .preview-pan { font-size: 19px; font-family: ui-monospace, monospace; letter-spacing: 2px; margin-bottom: 14px; font-weight: 600; }
  .preview-bottom { display: flex; justify-content: space-between; font-size: 11px; text-transform: uppercase; color: var(--gp-muted); }
  .preview-holder { font-weight: 600; color: #fff; font-size: 13px; margin-top: 2px; }
  .preview-exp { font-weight: 600; color: #fff; font-size: 13px; margin-top: 2px; text-align: right; }
  .scheme-tag { position: absolute; top: 18px; right: 20px; font-weight: 800; font-size: 15px; color: #fff; letter-spacing: 1px; }
  .input-group { margin-bottom: 16px; }
  .input-label { display: block; font-size: 12px; font-weight: 600; color: var(--gp-muted); margin-bottom: 6px; }
  .input-field { width: 100%%; background: #0D1117; border: 1.5px solid var(--gp-border); border-radius: 10px; padding: 12px 14px; color: #fff; font-size: 15px; font-family: ui-monospace, monospace; transition: all 0.2s; outline: none; }
  .input-field:focus { border-color: var(--gp-primary); box-shadow: 0 0 0 3px rgba(0,195,137,0.15); }
  .row-2 { display: flex; gap: 12px; }
  .btn-pay { width: 100%%; background: linear-gradient(135deg, var(--gp-primary), var(--gp-primary-dark)); border: none; border-radius: 12px; padding: 16px; color: #fff; font-size: 16px; font-weight: 700; cursor: pointer; transition: all 0.15s; margin-top: 8px; box-shadow: 0 4px 14px rgba(0,195,137,0.4); }
  .btn-pay:hover { opacity: 0.95; transform: translateY(-1px); }
  .btn-pay:active { transform: translateY(0); }
  .quick-cards-title { font-size: 12px; font-weight: 700; color: var(--gp-muted); text-transform: uppercase; letter-spacing: 0.5px; margin: 24px 0 10px; }
  .card-list { max-height: 200px; overflow-y: auto; border: 1px solid var(--gp-border); border-radius: 12px; }
  .card-item { display: flex; justify-content: space-between; align-items: center; padding: 10px 14px; border-bottom: 1px solid var(--gp-border); cursor: pointer; transition: background 0.15s; font-size: 13px; }
  .card-item:last-child { border-bottom: none; }
  .card-item:hover { background: #21262D; }
  .card-item-left { display: flex; align-items: center; gap: 10px; }
  .scheme-icon { width: 28px; height: 20px; border-radius: 4px; display: flex; align-items: center; justify-content: center; font-size: 10px; font-weight: 800; color: #fff; }
  .scheme-uzcard { background: #006699; }
  .scheme-humo { background: #FF9900; }
  .scheme-visa { background: #1A1F71; }
  .scheme-mastercard { background: #EB001B; }
  .card-holder { font-weight: 600; font-size: 12px; color: #fff; }
  .card-pan { font-size: 11px; color: var(--gp-muted); font-family: monospace; }
  .card-item-right { text-align: right; }
  .card-bal { font-size: 11px; font-weight: 600; color: var(--gp-muted); }
  .badge { font-size: 9px; font-weight: 700; padding: 2px 6px; border-radius: 4px; display: inline-block; margin-top: 2px; }
  .badge-scheme { background: rgba(0,195,137,0.15); color: var(--gp-primary); }
  .badge-declined { background: rgba(239,68,68,0.2); color: #EF4444; }
  .badge-zero { background: rgba(245,158,11,0.2); color: #F59E0B; }
  .footer-note { text-align: center; font-size: 11px; color: var(--gp-muted); margin-top: 20px; line-height: 1.4; }

  /* OTP Modal */
  .modal-overlay { display: none; position: fixed; inset: 0; background: rgba(0,0,0,0.75); backdrop-filter: blur(4px); z-index: 100; align-items: center; justify-content: center; padding: 20px; }
  .modal-overlay.active { display: flex; }
  .otp-modal { background: var(--gp-card); border: 1px solid var(--gp-border); border-radius: 18px; width: 100%%; max-width: 380px; padding: 28px; text-align: center; box-shadow: 0 20px 60px rgba(0,0,0,0.8); }
  .otp-icon { font-size: 40px; margin-bottom: 12px; }
  .otp-title { font-size: 18px; font-weight: 700; color: #fff; margin-bottom: 6px; }
  .otp-subtitle { font-size: 13px; color: var(--gp-muted); line-height: 1.5; margin-bottom: 20px; }
  .otp-input { width: 100%%; background: #0D1117; border: 2px solid var(--gp-border); border-radius: 12px; padding: 14px; font-size: 26px; font-weight: 800; letter-spacing: 12px; text-align: center; color: var(--gp-primary); font-family: ui-monospace, monospace; margin-bottom: 16px; outline: none; }
  .otp-input:focus { border-color: var(--gp-primary); }
  .btn-autofill { background: #21262D; border: 1px solid var(--gp-border); border-radius: 8px; padding: 8px 14px; color: #F0F6FC; font-size: 12px; font-weight: 600; cursor: pointer; margin-bottom: 18px; display: inline-flex; align-items: center; gap: 6px; }
  .btn-autofill:hover { background: #30363D; border-color: var(--gp-primary); }
  .btn-confirm { width: 100%%; background: var(--gp-primary); border: none; border-radius: 10px; padding: 14px; color: #fff; font-size: 15px; font-weight: 700; cursor: pointer; }
</style>
</head>
<body>

<div class="checkout-container">
  <div class="top-banner">
    <div class="brand-logo">
      <div class="brand-mark">GP</div>
      <span class="brand-name">GlobalPay<span class="brand-dot">.uz</span></span>
    </div>
    <div class="license-badge">
      Markaziy Bank<br><strong>Litsenziyasi №20</strong>
    </div>
  </div>

  <div class="body-content">
    <div class="amount-box">
      <div>
        <div class="order-meta-label">Buyurtma ID</div>
        <div class="order-id-val">%s</div>
      </div>
      <div>
        <div class="order-meta-label">To'lov miqdori</div>
        <div class="amount-val">%s</div>
      </div>
    </div>

    <!-- Interactive Card Mockup -->
    <div class="credit-card-preview" id="cardPreview">
      <div class="scheme-tag" id="previewScheme">UZCARD</div>
      <div class="card-chip"></div>
      <div class="preview-pan" id="previewPan">8600 •••• •••• 0001</div>
      <div class="preview-bottom">
        <div>
          <span>Karta egasi</span>
          <div class="preview-holder" id="previewHolder">ALISHER NAVOIY</div>
        </div>
        <div>
          <span>Muddat</span>
          <div class="preview-exp" id="previewExp">12/28</div>
        </div>
      </div>
    </div>

    <form id="payForm" method="POST" action="/sim/globalpay/checkout/process">
      <input type="hidden" name="token" value="%s">
      <input type="hidden" name="card_id" id="selectedCardId" value="sim-card-01">
      <input type="hidden" name="otp_code" id="otpCodeInput" value="123456">

      <div class="input-group">
        <label class="input-label">Karta raqami (16 ta raqam)</label>
        <input type="text" class="input-field" id="panInput" value="8600 1234 5678 0001" maxlength="19" oninput="onPanChange(this.value)" required>
      </div>

      <div class="row-2">
        <div class="input-group" style="flex:1">
          <label class="input-label">Amal qilish muddati</label>
          <input type="text" class="input-field" id="expInput" value="12/28" maxlength="5" oninput="onExpChange(this.value)" required>
        </div>
        <div class="input-group" style="flex:1">
          <label class="input-label">CVC / CVV</label>
          <input type="password" class="input-field" value="•••" maxlength="3">
        </div>
      </div>

      <button type="button" class="btn-pay" onclick="openOtpModal()">To'lashni tasdiqlash &#x2192;</button>
    </form>

    <div class="quick-cards-title">&#x26A1; Test kartalarini tanlash (1 click)</div>
    <div class="card-list">
      %s
    </div>

    <div class="footer-note">
      Bu mahalliy sinov simulyatori (GlobalPay Replica).<br>
      Hech qanday haqiqiy mablag' yechilmaydi.
    </div>
  </div>
</div>

<!-- OTP Modal -->
<div class="modal-overlay" id="otpModal">
  <div class="otp-modal">
    <div class="otp-icon">&#x1F4F1;</div>
    <div class="otp-title">SMS Tasdiqlash Kodingiz</div>
    <div class="otp-subtitle" id="otpPhoneText">
      Kod bir martalik SMS orqali yuborildi:<br>
      <strong style="color: #fff;">+998 90 *** ** 01</strong>
    </div>

    <input type="text" class="otp-input" id="otpInput" maxlength="6" value="123456">

    <div>
      <button type="button" class="btn-autofill" onclick="autofillOtp()">
        &#x26A1; Avtoto'ldirish (123456)
      </button>
    </div>

    <button type="button" class="btn-confirm" onclick="submitFinalPayment()">Tasdiqlash va To'lash</button>
  </div>
</div>

<script>
function onPanChange(val) {
  let clean = val.replace(/\D/g, '');
  let formatted = '';
  for (let i = 0; i < clean.length; i++) {
    if (i > 0 && i %% 4 === 0) formatted += ' ';
    formatted += clean[i];
  }
  document.getElementById('panInput').value = formatted;
  document.getElementById('selectedCardId').value = clean;

  let scheme = 'UZCARD';
  if (clean.startsWith('9860')) scheme = 'HUMO';
  else if (clean.startsWith('4')) scheme = 'VISA';
  else if (clean.startsWith('5')) scheme = 'MASTERCARD';

  document.getElementById('previewScheme').innerText = scheme;
  document.getElementById('previewPan').innerText = formatted || '•••• •••• •••• ••••';
}

function onExpChange(val) {
  let clean = val.replace(/\D/g, '');
  if (clean.length >= 2) {
    clean = clean.substring(0, 2) + '/' + clean.substring(2, 4);
  }
  document.getElementById('expInput').value = clean;
  document.getElementById('previewExp').innerText = clean || 'MM/YY';
}

function selectCard(id, pan, exp, holder, scheme) {
  document.getElementById('selectedCardId').value = id;
  document.getElementById('panInput').value = pan.match(/.{1,4}/g).join(' ');
  document.getElementById('expInput').value = exp;
  document.getElementById('previewPan').innerText = pan.match(/.{1,4}/g).join(' ');
  document.getElementById('previewHolder').innerText = holder;
  document.getElementById('previewExp').innerText = exp;
  document.getElementById('previewScheme').innerText = scheme;
}

function openOtpModal() {
  document.getElementById('otpModal').classList.add('active');
  document.getElementById('otpInput').focus();
}

function autofillOtp() {
  document.getElementById('otpInput').value = '123456';
}

function submitFinalPayment() {
  document.getElementById('otpCodeInput').value = document.getElementById('otpInput').value;
  document.getElementById('payForm').submit();
}
</script>

</body>
</html>`, orderID, amountDisplay, token, cardRows.String())
}

func buildResultHTML(orderID, status string, card *SimCard, amountMinor int64, currency string) string {
	icon := "&#x2705;"
	title := "To'lov Muvaffaqiyatli Amalga Oshirildi"
	subtitle := "GlobalPay tizimi orqali to'lov qabul qilindi va buyurtma keyingi bosqichga o'tkazildi."
	color := "#00C389"
	if status != "SUCCESS" {
		icon = "&#x274C;"
		title = "To'lov Rad Etildi"
		subtitle = "Karta bloklangan yoki hisobda yetarli mablag' mavjud emas. Iltimos boshqa karta bilan qayta urinib ko'ring."
		color = "#EF4444"
	}

	amountFormatted := formatMinor(amountMinor, currency) + " " + currency
	rrn := fmt.Sprintf("GP%d", time.Now().UnixMilli())
	authCode := "123456"

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="uz">
<head>
<meta charset="UTF-8">
<title>GlobalPay · Kvitansiya</title>
<style>
  :root { --gp-bg: #0D1117; --gp-card: #161B22; --gp-border: #30363D; --gp-text: #F0F6FC; --gp-muted: #8B949E; }
  * { box-sizing: border-box; margin: 0; padding: 0; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; }
  body { background: var(--gp-bg); color: var(--gp-text); min-height: 100vh; display: flex; align-items: center; justify-content: center; padding: 20px; }
  .receipt-box { width: 100%%; max-width: 440px; background: var(--gp-card); border: 1px solid var(--gp-border); border-radius: 20px; padding: 36px 28px; text-align: center; box-shadow: 0 30px 80px rgba(0,0,0,0.6); }
  .icon { font-size: 56px; margin-bottom: 16px; }
  h1 { font-size: 21px; font-weight: 800; color: %s; margin-bottom: 8px; line-height: 1.3; }
  p { font-size: 13px; color: var(--gp-muted); line-height: 1.5; margin-bottom: 24px; }
  .meta-card { background: #21262D; border: 1px solid var(--gp-border); border-radius: 12px; padding: 16px; margin-bottom: 24px; text-align: left; }
  .row { display: flex; justify-content: space-between; padding: 6px 0; font-size: 13px; border-bottom: 1px solid rgba(255,255,255,0.05); }
  .row:last-child { border-bottom: none; }
  .label { color: var(--gp-muted); }
  .val { color: #fff; font-weight: 600; font-family: ui-monospace, monospace; }
  .badge { display: inline-block; padding: 3px 10px; border-radius: 6px; font-size: 11px; font-weight: 700; background: %s22; color: %s; border: 1px solid %s55; }
  .footer-note { font-size: 11px; color: var(--gp-muted); }
</style>
</head>
<body>
<div class="receipt-box">
  <div class="icon">%s</div>
  <h1>%s</h1>
  <p>%s</p>
  <div class="meta-card">
    <div class="row"><span class="label">Buyurtma ID</span><span class="val">%s</span></div>
    <div class="row"><span class="label">Summa</span><span class="val" style="color:#00C389;">%s</span></div>
    <div class="row"><span class="label">Karta</span><span class="val">%s (%s)</span></div>
    <div class="row"><span class="label">RRN</span><span class="val">%s</span></div>
    <div class="row"><span class="label">Avtorizatsiya</span><span class="val">%s</span></div>
    <div class="row"><span class="label">Holat</span><span class="badge">%s</span></div>
  </div>
  <p class="footer-note">Webhooks avtomatik tarzda jo'natildi. Oynani yopishingiz mumkin.</p>
</div>
</body>
</html>`, color, color, color, color, icon, title, subtitle, orderID, amountFormatted, maskPAN(card.PAN), card.Scheme, rrn, authCode, status)
}

func buildP2PTransferHTML(cards []SimCard) string {
	var senderOptions strings.Builder
	for _, c := range cards {
		if c.Declined || c.Balance == 0 {
			continue
		}
		senderOptions.WriteString(fmt.Sprintf(`<option value="%s">%s (Balans: %s so'm)</option>`,
			c.ID, c.Label, formatNumberSpace(c.Balance/100)))
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="uz">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>GlobalPay · Kartadan Kartaga Pul O'tkazish (P2P)</title>
<style>
  :root { --gp-primary: #00C389; --gp-bg: #0D1117; --gp-card: #161B22; --gp-border: #30363D; --gp-text: #F0F6FC; --gp-muted: #8B949E; }
  * { box-sizing: border-box; margin: 0; padding: 0; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; }
  body { background: var(--gp-bg); color: var(--gp-text); min-height: 100vh; display: flex; align-items: center; justify-content: center; padding: 20px; }
  .transfer-card { width: 100%%; max-width: 460px; background: var(--gp-card); border: 1px solid var(--gp-border); border-radius: 20px; padding: 28px; box-shadow: 0 30px 80px rgba(0,0,0,0.6); }
  .header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 24px; }
  .title { font-size: 19px; font-weight: 800; }
  .badge { background: rgba(0,195,137,0.15); color: var(--gp-primary); border: 1px solid rgba(0,195,137,0.3); border-radius: 6px; padding: 4px 10px; font-size: 11px; font-weight: 700; }
  .input-group { margin-bottom: 16px; }
  .label { display: block; font-size: 12px; font-weight: 600; color: var(--gp-muted); margin-bottom: 6px; }
  .select, .input { width: 100%%; background: #0D1117; border: 1.5px solid var(--gp-border); border-radius: 10px; padding: 12px; color: #fff; font-size: 14px; outline: none; }
  .select:focus, .input:focus { border-color: var(--gp-primary); }
  .fee-preview { background: #21262D; border: 1px solid var(--gp-border); border-radius: 12px; padding: 14px; margin: 20px 0; font-size: 13px; }
  .fee-row { display: flex; justify-content: space-between; margin-bottom: 6px; }
  .fee-row:last-child { margin-bottom: 0; font-weight: 700; color: var(--gp-primary); }
  .btn { width: 100%%; background: var(--gp-primary); border: none; border-radius: 12px; padding: 15px; color: #fff; font-size: 15px; font-weight: 700; cursor: pointer; }
</style>
</head>
<body>
<div class="transfer-card">
  <div class="header">
    <div class="title">Kartadan Kartaga O'tkazma</div>
    <div class="badge">&#x26A1; P2P Simulyator</div>
  </div>

  <div class="input-group">
    <label class="label">Yuboruvchi Karta</label>
    <select class="select" id="senderSelect">
      %s
    </select>
  </div>

  <div class="input-group">
    <label class="label">Qabul Qiluvchi Karta Raqami</label>
    <input type="text" class="input" id="recipientInput" placeholder="8600 •••• •••• 0002" value="9860 1234 5678 0002">
  </div>

  <div class="input-group">
    <label class="label">O'tkazma Summasi (so'm)</label>
    <input type="number" class="input" id="amountInput" value="500000" oninput="updateFee()">
  </div>

  <div class="fee-preview">
    <div class="fee-row"><span>Komissiya (0.5%%)</span><span id="feeDisplay">2 500 so'm</span></div>
    <div class="fee-row"><span>Jami yechiladi</span><span id="totalDisplay">502 500 so'm</span></div>
  </div>

  <button type="button" class="btn" onclick="executeP2P()">O'tkazishni Boshlash &#x2192;</button>
  <div id="resultStatus" style="margin-top: 16px; font-size: 13px; text-align: center;"></div>
</div>

<script>
function updateFee() {
  let val = parseInt(document.getElementById('amountInput').value) || 0;
  let fee = Math.max(1000, Math.floor(val * 0.005));
  let total = val + fee;
  document.getElementById('feeDisplay').innerText = fee.toLocaleString() + " so'm";
  document.getElementById('totalDisplay').innerText = total.toLocaleString() + " so'm";
}

async function executeP2P() {
  let senderId = document.getElementById('senderSelect').value;
  let recipient = document.getElementById('recipientInput').value.replace(/\s/g, '');
  let val = parseInt(document.getElementById('amountInput').value) || 0;
  let amountMinor = val * 100;

  let resEl = document.getElementById('resultStatus');
  resEl.innerText = 'SMS OTP jo'natilmoqda...';

  try {
    let resp = await fetch('/sim/globalpay/v1/transfers/p2p', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({
        sender_card_id: senderId,
        recipient_pan: recipient,
        amount_minor: amountMinor,
        currency: 'UZS'
      })
    });
    let data = await resp.json();
    if (!resp.ok) {
      resEl.innerHTML = '<span style="color:#EF4444">&#x274C; Xatolik: ' + (data.error || 'Muvaffaqiyatsiz') + '</span>';
      return;
    }

    // Auto-confirm with OTP 123456
    resEl.innerText = 'OTP tasdiqlanmoqda (123456)...';
    let confirmResp = await fetch('/sim/globalpay/v1/transfers/p2p/confirm', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({
        transfer_id: data.transfer_id,
        otp_code: '123456'
      })
    });
    let confirmData = await confirmResp.json();
    if (confirmResp.ok) {
      resEl.innerHTML = '<span style="color:#00C389">&#x2705; Muvaffaqiyatli o'tkazildi! RRN: ' + confirmData.rrn + '</span>';
    } else {
      resEl.innerHTML = '<span style="color:#EF4444">&#x274C; Tasdiqlash xatoligi: ' + confirmData.error + '</span>';
    }
  } catch(e) {
    resEl.innerHTML = '<span style="color:#EF4444">&#x274C; Tarmoq xatoligi: ' + e + '</span>';
  }
}
</script>
</body>
</html>`, senderOptions.String())
}
