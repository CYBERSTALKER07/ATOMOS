package retailer

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"google.golang.org/api/iterator"
)

const (
	telegramOTPTTL            = 10 * time.Minute
	defaultTelegramBotName    = "pegasusx_retailer_bot"
	telegramWebhookSecretHdr  = "X-Bot-Webhook-Secret"
	telegramBotSecretEnvMain  = "TELEGRAM_BOT_WEBHOOK_SECRET"
	telegramBotSecretEnvAlias = "BOT_WEBHOOK_SECRET"
)

// TelegramLinkRecord persists RetailerTelegramLinks row state in Spanner and tests.
type TelegramLinkRecord struct {
	RetailerID     string     `json:"retailer_id"`
	UserID         string     `json:"user_id"`
	PhoneNumber    string     `json:"phone_number,omitempty"`
	TelegramChatID string     `json:"telegram_chat_id,omitempty"`
	OtpHash        string     `json:"-"`
	Locale         string     `json:"locale"`
	VerifiedAt     *time.Time `json:"verified_at,omitempty"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// TelegramChatOrderSummary matches #163 Telegram bot chat context order items.
type TelegramChatOrderSummary struct {
	OrderID               string `json:"order_id"`
	Status                string `json:"status"`
	ConfirmationStatus    string `json:"confirmation_status,omitempty"`
	TotalAmountMinor      int64  `json:"total_amount_minor"`
	Currency              string `json:"currency"`
	RequestedDeliveryDate string `json:"requested_delivery_date,omitempty"`
}

// SetBotWebhookSecret overrides TELEGRAM_BOT_WEBHOOK_SECRET in tests.
func (s *Service) SetBotWebhookSecret(secret string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.botWebhookSecret = strings.TrimSpace(secret)
	s.mu.Unlock()
}

func (s *Service) effectiveBotWebhookSecret() string {
	s.mu.RLock()
	configured := s.botWebhookSecret
	s.mu.RUnlock()
	if configured != "" {
		return configured
	}
	if v := strings.TrimSpace(os.Getenv(telegramBotSecretEnvMain)); v != "" {
		return v
	}
	return strings.TrimSpace(os.Getenv(telegramBotSecretEnvAlias))
}

func (s *Service) validateBotWebhookSecret(r *http.Request) (bool, int, string) {
	expected := s.effectiveBotWebhookSecret()
	if expected == "" {
		return false, http.StatusServiceUnavailable, "bot_webhook_secret_not_configured"
	}
	got := strings.TrimSpace(r.Header.Get(telegramWebhookSecretHdr))
	if got == "" {
		return false, http.StatusUnauthorized, "unauthorized"
	}
	if subtle.ConstantTimeCompare([]byte(got), []byte(expected)) != 1 {
		return false, http.StatusUnauthorized, "invalid_bot_webhook_secret"
	}
	return true, 0, ""
}

func retailerActorFromRequest(r *http.Request) (retailerID, userID, phone string, err error) {
	retailerID, err = retailerIDFromRequest(r)
	if err != nil {
		return "", "", "", err
	}
	userID = retailerID
	if claims, ok := auth.FromContext(r.Context()); ok {
		if u := strings.TrimSpace(claims.RetailerUserID); u != "" {
			userID = u
		} else if sub := strings.TrimSpace(claims.Subject); sub != "" {
			userID = sub
		}
		phone = NormalizeUZPhone(strings.TrimSpace(claims.PhoneNumber))
	}
	return retailerID, userID, phone, nil
}

// HandleTelegramLinkRequest serves POST /v1/retailer/telegram/link-request (retailer JWT).
func (s *Service) HandleTelegramLinkRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	retailerID, userID, claimsPhone, err := retailerActorFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	var req struct {
		PhoneNumber string `json:"phone_number"`
		Locale      string `json:"locale"`
	}
	if r.Body != nil {
		if body, ok := readLimitedBody(w, r, 16*1024); !ok {
			return
		} else if len(strings.TrimSpace(string(body))) > 0 {
			if unmarshalErr := json.Unmarshal(body, &req); unmarshalErr != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
				return
			}
		}
	}

	phone := NormalizeUZPhone(strings.TrimSpace(req.PhoneNumber))
	if phone == "" {
		phone = claimsPhone
	}
	if phone == "" {
		phone = s.lookupRetailerUserPhone(r.Context(), retailerID, userID)
	}

	existing, hasExisting, err := s.getTelegramLinkByKey(r.Context(), retailerID, userID)
	if err != nil {
		s.log.Warn("telegram link lookup failed", "retailer_id", retailerID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "telegram_link_lookup_failed"})
		return
	}
	locale := normalizeRetailerLocale(req.Locale)
	if strings.TrimSpace(req.Locale) == "" && hasExisting && existing.Locale != "" {
		locale = normalizeRetailerLocale(existing.Locale)
	}
	if phone == "" && hasExisting {
		phone = existing.PhoneNumber
	}

	otpCode, err := generateSixDigitOTP()
	if err != nil {
		s.log.Warn("telegram otp generation failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "otp_generation_failed"})
		return
	}

	now := s.now().UTC()
	expiresAt := now.Add(telegramOTPTTL)
	rec := TelegramLinkRecord{
		RetailerID:  retailerID,
		UserID:      userID,
		PhoneNumber: phone,
		OtpHash:     hashTelegramOTP(otpCode),
		Locale:      locale,
		ExpiresAt:   &expiresAt,
		UpdatedAt:   now,
	}
	if hasExisting {
		rec.TelegramChatID = existing.TelegramChatID
		rec.VerifiedAt = existing.VerifiedAt
	}

	if err := s.upsertTelegramLink(r.Context(), rec); err != nil {
		s.log.Warn("telegram link upsert failed", "retailer_id", retailerID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "telegram_link_save_failed"})
		return
	}

	botUsername := strings.TrimPrefix(strings.TrimSpace(os.Getenv("TELEGRAM_BOT_USERNAME")), "@")
	if botUsername == "" {
		botUsername = defaultTelegramBotName
	}
	deepLink := fmt.Sprintf("https://t.me/%s?start=%s", botUsername, otpCode)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":            true,
		"otp_code":      otpCode,
		"expires_at":    expiresAt.Format(time.RFC3339Nano),
		"bot_deep_link": deepLink,
		"deep_link":     deepLink,
		"locale":        locale,
	})
}

// HandleTelegramStatus serves GET /v1/retailer/telegram/status (retailer JWT).
func (s *Service) HandleTelegramStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	retailerID, userID, _, err := retailerActorFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}

	rec, found, err := s.getTelegramLinkByKey(r.Context(), retailerID, userID)
	if err != nil {
		s.log.Warn("telegram status read failed", "retailer_id", retailerID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "telegram_status_failed"})
		return
	}
	if !found {
		rec, found, err = s.findLatestVerifiedTelegramLinkForRetailer(r.Context(), retailerID)
		if err != nil {
			s.log.Warn("telegram retailer link read failed", "retailer_id", retailerID, "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "telegram_status_failed"})
			return
		}
	}

	if !found {
		writeJSON(w, http.StatusOK, map[string]any{
			"linked":      false,
			"retailer_id": retailerID,
			"user_id":     userID,
			"locale":      "uz",
		})
		return
	}

	linked := rec.VerifiedAt != nil && strings.TrimSpace(rec.TelegramChatID) != ""
	resp := map[string]any{
		"linked":           linked,
		"retailer_id":      rec.RetailerID,
		"user_id":          rec.UserID,
		"phone_number":     rec.PhoneNumber,
		"telegram_chat_id": rec.TelegramChatID,
		"locale":           normalizeRetailerLocale(rec.Locale),
	}
	if rec.VerifiedAt != nil {
		resp["verified_at"] = rec.VerifiedAt.UTC().Format(time.RFC3339Nano)
		resp["linked_at"] = rec.VerifiedAt.UTC().Format(time.RFC3339Nano)
	}
	if rec.ExpiresAt != nil && rec.OtpHash != "" {
		resp["pending_otp_expires_at"] = rec.ExpiresAt.UTC().Format(time.RFC3339Nano)
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleTelegramVerifyLink serves POST /v1/retailer/telegram/verify-link (bot secret OR retailer JWT).
func (s *Service) HandleTelegramVerifyLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}

	hasBotHdr := strings.TrimSpace(r.Header.Get(telegramWebhookSecretHdr)) != ""
	var jwtRetailerID string
	var jwtUserID string
	if hasBotHdr {
		ok, code, errCode := s.validateBotWebhookSecret(r)
		if !ok {
			writeJSON(w, code, map[string]string{"error": errCode})
			return
		}
	} else if claims, ok := auth.FromContext(r.Context()); ok && claims.Role == auth.RoleRetailer {
		rid, uid, _, rErr := retailerActorFromRequest(r)
		if rErr != nil {
			writeRetailerIdentityError(w, rErr)
			return
		}
		jwtRetailerID = rid
		jwtUserID = uid
	} else {
		_, code, errCode := s.validateBotWebhookSecret(r)
		writeJSON(w, code, map[string]string{"error": errCode})
		return
	}

	body, ok := readLimitedBody(w, r, 32*1024)
	if !ok {
		return
	}
	var req struct {
		TelegramChatID string `json:"telegram_chat_id"`
		OTPCode        string `json:"otp_code"`
		PhoneNumber    string `json:"phone_number"`
		Locale         string `json:"locale"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	chatID := strings.TrimSpace(req.TelegramChatID)
	otpCode := strings.TrimSpace(req.OTPCode)
	phone := NormalizeUZPhone(strings.TrimSpace(req.PhoneNumber))
	if chatID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "telegram_chat_id_required"})
		return
	}
	if otpCode == "" && phone == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "otp_code_or_phone_required"})
		return
	}

	now := s.now().UTC()
	var matched TelegramLinkRecord
	var found bool
	var err error

	if otpCode != "" {
		matched, found, err = s.findTelegramLinkByOTPHash(r.Context(), hashTelegramOTP(otpCode))
		if err != nil {
			s.log.Warn("telegram verify otp lookup failed", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "verify_link_failed"})
			return
		}
		if !found {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_or_expired_otp"})
			return
		}
		if matched.ExpiresAt == nil || !matched.ExpiresAt.After(now) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "otp_expired"})
			return
		}
		if jwtRetailerID != "" && matched.RetailerID != jwtRetailerID {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
	} else {
		matched, found, err = s.findTelegramLinkOrIdentityByPhone(r.Context(), phone, jwtRetailerID, jwtUserID)
		if err != nil {
			s.log.Warn("telegram verify phone lookup failed", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "verify_link_failed"})
			return
		}
		if !found {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "phone_not_registered"})
			return
		}
	}

	if reqLocale := strings.TrimSpace(req.Locale); reqLocale != "" {
		if !isSupportedLocale(reqLocale) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_locale"})
			return
		}
		matched.Locale = strings.ToLower(reqLocale)
	} else if matched.Locale == "" {
		matched.Locale = "uz"
	} else {
		matched.Locale = normalizeRetailerLocale(matched.Locale)
	}
	if phone != "" && matched.PhoneNumber == "" {
		matched.PhoneNumber = phone
	}

	matched.TelegramChatID = chatID
	matched.OtpHash = ""
	matched.ExpiresAt = nil
	matched.VerifiedAt = &now
	matched.UpdatedAt = now

	if err := s.bindTelegramChat(r.Context(), matched); err != nil {
		s.log.Warn("telegram bind save failed", "retailer_id", matched.RetailerID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "verify_link_save_failed"})
		return
	}

	retailerName := s.lookupRetailerDisplayName(r.Context(), matched.RetailerID)
	writeJSON(w, http.StatusOK, map[string]any{
		"linked":           true,
		"retailer_id":      matched.RetailerID,
		"retailer_name":    retailerName,
		"user_id":          matched.UserID,
		"phone_number":     matched.PhoneNumber,
		"telegram_chat_id": matched.TelegramChatID,
		"locale":           matched.Locale,
		"linked_at":        now.Format(time.RFC3339Nano),
		"verified_at":      now.Format(time.RFC3339Nano),
	})
}

// HandleTelegramChatContext serves GET /v1/retailer/telegram/chat/{chatId}/context (X-Bot-Webhook-Secret).
func (s *Service) HandleTelegramChatContext(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if ok, code, errCode := s.validateBotWebhookSecret(r); !ok {
		writeJSON(w, code, map[string]string{"error": errCode})
		return
	}
	chatID := strings.TrimSpace(chi.URLParam(r, "chatId"))
	if chatID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "chat_id_required"})
		return
	}
	rec, found, err := s.findTelegramLinkByChatID(r.Context(), chatID)
	if err != nil {
		s.log.Warn("telegram context link read failed", "chat_id", chatID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "telegram_context_failed"})
		return
	}
	if !found || rec.VerifiedAt == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "telegram_chat_not_linked"})
		return
	}

	fh, fhErr := s.buildFastHome(r.Context(), rec.RetailerID, rec.UserID, true)
	if fhErr != nil {
		s.log.Warn("telegram context fast-home read failed", "retailer_id", rec.RetailerID, "err", fhErr)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "telegram_context_failed"})
		return
	}
	orders, ordErr := s.listTelegramChatOrders(r.Context(), rec.RetailerID, fh)
	if ordErr != nil {
		s.log.Warn("telegram context orders read failed", "retailer_id", rec.RetailerID, "err", ordErr)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "telegram_context_failed"})
		return
	}
	retailerName := s.lookupRetailerDisplayName(r.Context(), rec.RetailerID)
	writeJSON(w, http.StatusOK, map[string]any{
		"linked":             true,
		"retailer_id":        rec.RetailerID,
		"retailer_name":      retailerName,
		"user_id":            rec.UserID,
		"locale":             normalizeRetailerLocale(rec.Locale),
		"balance_owed_minor": fh.BalanceOwedMinor,
		"overdue_minor":      fh.OverdueMinor,
		"days_overdue":       fh.DaysOverdue,
		"orders":             orders,
	})
}

// HandleTelegramUpdateLocale serves PATCH /v1/retailer/telegram/chat/{chatId}/locale (X-Bot-Webhook-Secret).
func (s *Service) HandleTelegramUpdateLocale(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if ok, code, errCode := s.validateBotWebhookSecret(r); !ok {
		writeJSON(w, code, map[string]string{"error": errCode})
		return
	}
	chatID := strings.TrimSpace(chi.URLParam(r, "chatId"))
	if chatID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "chat_id_required"})
		return
	}
	body, ok := readLimitedBody(w, r, 4*1024)
	if !ok {
		return
	}
	var req struct {
		Locale string `json:"locale"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if !isSupportedLocale(req.Locale) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_locale"})
		return
	}
	locale := strings.ToLower(strings.TrimSpace(req.Locale))

	rec, found, err := s.findTelegramLinkByChatID(r.Context(), chatID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "telegram_locale_update_failed"})
		return
	}
	if !found || rec.VerifiedAt == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "telegram_chat_not_linked"})
		return
	}
	rec.Locale = locale
	rec.UpdatedAt = s.now().UTC()
	if err := s.upsertTelegramLink(r.Context(), rec); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "telegram_locale_update_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"locale": locale})
}

// HandleTelegramPreorderConfirm serves POST /v1/retailer/telegram/chat/{chatId}/preorder/{orderId}/confirm.
func (s *Service) HandleTelegramPreorderConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if ok, code, errCode := s.validateBotWebhookSecret(r); !ok {
		writeJSON(w, code, map[string]string{"error": errCode})
		return
	}
	chatID := strings.TrimSpace(chi.URLParam(r, "chatId"))
	orderID := strings.TrimSpace(chi.URLParam(r, "orderId"))
	if chatID == "" || orderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "chat_id_and_order_id_required"})
		return
	}
	body, ok := readLimitedBody(w, r, 16*1024)
	if !ok {
		return
	}
	if s.guardIdempotency(w, r, body) {
		return
	}
	if s.orders == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "order_lifecycle_unavailable"})
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	rec, found, err := s.findTelegramLinkByChatID(r.Context(), chatID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "telegram_lookup_failed"})
		return
	}
	if !found || rec.VerifiedAt == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "telegram_chat_not_linked"})
		return
	}

	resp, err := s.orders.ConfirmAIOrder(r.Context(), rec.RetailerID, order.ConfirmAIOrderRequest{OrderID: orderID})
	if errors.Is(err, order.ErrInvalidStatusTransition) {
		resp, err = s.orders.ConfirmPreorder(r.Context(), rec.RetailerID, order.ConfirmPreorderRequest{OrderID: orderID})
	}
	if err != nil {
		writeRetailerOrderLifecycleError(w, err)
		return
	}

	status := string(resp.ConfirmationStatus)
	if status == "" {
		status = string(resp.Status)
	}
	outBytes, marshalErr := json.Marshal(map[string]any{
		"order_id":            resp.OrderID,
		"status":              status,
		"order_status":        resp.Status,
		"confirmation_status": resp.ConfirmationStatus,
	})
	if marshalErr != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "encode_response_failed"})
		return
	}
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, outBytes)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, outBytes)
}

// HandleTelegramPreorderReject serves POST /v1/retailer/telegram/chat/{chatId}/preorder/{orderId}/reject.
func (s *Service) HandleTelegramPreorderReject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if ok, code, errCode := s.validateBotWebhookSecret(r); !ok {
		writeJSON(w, code, map[string]string{"error": errCode})
		return
	}
	chatID := strings.TrimSpace(chi.URLParam(r, "chatId"))
	orderID := strings.TrimSpace(chi.URLParam(r, "orderId"))
	if chatID == "" || orderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "chat_id_and_order_id_required"})
		return
	}
	body, ok := readLimitedBody(w, r, 16*1024)
	if !ok {
		return
	}
	if s.guardIdempotency(w, r, body) {
		return
	}
	if s.orders == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "order_lifecycle_unavailable"})
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	var req struct {
		Reason string `json:"reason"`
	}
	if len(strings.TrimSpace(string(body))) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
	}

	rec, found, err := s.findTelegramLinkByChatID(r.Context(), chatID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "telegram_lookup_failed"})
		return
	}
	if !found || rec.VerifiedAt == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "telegram_chat_not_linked"})
		return
	}

	resp, err := s.orders.RejectAIOrder(r.Context(), rec.RetailerID, order.RejectAIOrderRequest{
		OrderID: orderID,
		Reason:  strings.TrimSpace(req.Reason),
	})
	if errors.Is(err, order.ErrInvalidStatusTransition) {
		resp, err = s.orders.RejectPreorder(r.Context(), rec.RetailerID, order.RejectPreorderRequest{
			OrderID: orderID,
			Reason:  strings.TrimSpace(req.Reason),
		})
	}
	if err != nil {
		writeRetailerOrderLifecycleError(w, err)
		return
	}

	status := string(resp.ConfirmationStatus)
	if status == "" {
		status = string(resp.Status)
	}
	outBytes, marshalErr := json.Marshal(map[string]any{
		"order_id":            resp.OrderID,
		"status":              status,
		"order_status":        resp.Status,
		"confirmation_status": resp.ConfirmationStatus,
	})
	if marshalErr != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "encode_response_failed"})
		return
	}
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, outBytes)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, outBytes)
}

func (s *Service) listTelegramChatOrders(ctx context.Context, retailerID string, fh *FastHomeResponse) ([]TelegramChatOrderSummary, error) {
	if s.spannerClient != nil {
		stmt := spanner.Statement{
			SQL: `SELECT OrderId, Status, ConfirmationStatus, TotalMinor, Currency, RequestedDeliveryDate
				FROM Orders
				WHERE RetailerId = @rid
				ORDER BY UpdatedAt DESC
				LIMIT 10`,
			Params: map[string]any{"rid": retailerID},
		}
		iter := s.spannerClient.Single().Query(ctx, stmt)
		defer iter.Stop()
		out := make([]TelegramChatOrderSummary, 0, 10)
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, err
			}
			var ord TelegramChatOrderSummary
			var reqDel spanner.NullTime
			if err := row.Columns(&ord.OrderID, &ord.Status, &ord.ConfirmationStatus, &ord.TotalAmountMinor, &ord.Currency, &reqDel); err != nil {
				return nil, err
			}
			ord.Currency = firstNonEmptyFastHome(ord.Currency, "UZS")
			if reqDel.Valid {
				ord.RequestedDeliveryDate = reqDel.Time.UTC().Format("2006-01-02")
			}
			out = append(out, ord)
		}
		return out, nil
	}

	out := make([]TelegramChatOrderSummary, 0)
	seen := make(map[string]bool)
	if fh != nil && fh.PendingProposal != nil && fh.PendingProposal.OrderID != "" {
		seen[fh.PendingProposal.OrderID] = true
		out = append(out, TelegramChatOrderSummary{
			OrderID:               fh.PendingProposal.OrderID,
			Status:                "PENDING",
			ConfirmationStatus:    fh.PendingProposal.ConfirmationStatus,
			TotalAmountMinor:      fh.PendingProposal.TotalMinor,
			Currency:              fh.PendingProposal.Currency,
			RequestedDeliveryDate: fh.PendingProposal.RequestedDeliveryDate,
		})
	}
	if s.orders != nil {
		if preds, err := s.orders.ListRetailerAIPredictions(ctx, retailerID, 10); err == nil {
			for _, p := range preds {
				if seen[p.OrderID] {
					continue
				}
				seen[p.OrderID] = true
				out = append(out, TelegramChatOrderSummary{
					OrderID:               p.OrderID,
					Status:                string(p.Source),
					ConfirmationStatus:    string(p.ConfirmationStatus),
					TotalAmountMinor:      p.TotalMinor,
					Currency:              firstNonEmptyFastHome(p.Currency, "UZS"),
					RequestedDeliveryDate: p.RequestedDeliveryDate,
				})
			}
		}
	}
	if s.repo != nil {
		if tracking, err := s.repo.ListTrackingOrders(ctx, retailerID, 10, 0); err == nil {
			for _, ord := range tracking {
				if seen[ord.OrderID] {
					continue
				}
				seen[ord.OrderID] = true
				reqDate := ""
				if ord.DeliveryExpectation != nil && ord.DeliveryExpectation.TargetDate != nil {
					reqDate = *ord.DeliveryExpectation.TargetDate
				}
				out = append(out, TelegramChatOrderSummary{
					OrderID:               ord.OrderID,
					Status:                ord.Status,
					TotalAmountMinor:      ord.TotalMinor,
					Currency:              firstNonEmptyFastHome(ord.Currency, "UZS"),
					RequestedDeliveryDate: reqDate,
				})
			}
		}
	}
	return out, nil
}

func telegramLinkMemKey(retailerID, userID string) string {
	return retailerID + "|" + userID
}

func (s *Service) getTelegramLinkByKey(ctx context.Context, retailerID, userID string) (TelegramLinkRecord, bool, error) {
	if s.spannerClient != nil {
		stmt := spanner.Statement{
			SQL: `SELECT RetailerId, UserId, COALESCE(PhoneNumber, ''), COALESCE(TelegramChatId, ''),
				       COALESCE(OtpHash, ''), Locale, VerifiedAt, ExpiresAt, UpdatedAt
				FROM RetailerTelegramLinks
				WHERE RetailerId = @rid AND UserId = @uid
				LIMIT 1`,
			Params: map[string]any{"rid": retailerID, "uid": userID},
		}
		return s.scanSingleTelegramLink(ctx, stmt)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.telegramLinksMem[telegramLinkMemKey(retailerID, userID)]
	return rec, ok, nil
}

func (s *Service) findLatestVerifiedTelegramLinkForRetailer(ctx context.Context, retailerID string) (TelegramLinkRecord, bool, error) {
	if s.spannerClient != nil {
		stmt := spanner.Statement{
			SQL: `SELECT RetailerId, UserId, COALESCE(PhoneNumber, ''), COALESCE(TelegramChatId, ''),
				       COALESCE(OtpHash, ''), Locale, VerifiedAt, ExpiresAt, UpdatedAt
				FROM RetailerTelegramLinks
				WHERE RetailerId = @rid
				ORDER BY VerifiedAt DESC, UpdatedAt DESC
				LIMIT 1`,
			Params: map[string]any{"rid": retailerID},
		}
		return s.scanSingleTelegramLink(ctx, stmt)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var best TelegramLinkRecord
	var found bool
	for _, rec := range s.telegramLinksMem {
		if rec.RetailerID != retailerID {
			continue
		}
		if !found || (rec.VerifiedAt != nil && best.VerifiedAt == nil) || rec.UpdatedAt.After(best.UpdatedAt) {
			best = rec
			found = true
		}
	}
	return best, found, nil
}

func (s *Service) findTelegramLinkByOTPHash(ctx context.Context, otpHash string) (TelegramLinkRecord, bool, error) {
	if s.spannerClient != nil {
		stmt := spanner.Statement{
			SQL: `SELECT RetailerId, UserId, COALESCE(PhoneNumber, ''), COALESCE(TelegramChatId, ''),
				       COALESCE(OtpHash, ''), Locale, VerifiedAt, ExpiresAt, UpdatedAt
				FROM RetailerTelegramLinks
				WHERE OtpHash = @hash
				ORDER BY UpdatedAt DESC
				LIMIT 1`,
			Params: map[string]any{"hash": otpHash},
		}
		return s.scanSingleTelegramLink(ctx, stmt)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, rec := range s.telegramLinksMem {
		if rec.OtpHash == otpHash && otpHash != "" {
			return rec, true, nil
		}
	}
	return TelegramLinkRecord{}, false, nil
}

func (s *Service) findTelegramLinkByChatID(ctx context.Context, chatID string) (TelegramLinkRecord, bool, error) {
	if s.spannerClient != nil {
		stmt := spanner.Statement{
			SQL: `SELECT RetailerId, UserId, COALESCE(PhoneNumber, ''), COALESCE(TelegramChatId, ''),
				       COALESCE(OtpHash, ''), Locale, VerifiedAt, ExpiresAt, UpdatedAt
				FROM RetailerTelegramLinks
				WHERE TelegramChatId = @chatId
				LIMIT 1`,
			Params: map[string]any{"chatId": chatID},
		}
		return s.scanSingleTelegramLink(ctx, stmt)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, rec := range s.telegramLinksMem {
		if rec.TelegramChatID == chatID {
			return rec, true, nil
		}
	}
	return TelegramLinkRecord{}, false, nil
}

func (s *Service) findTelegramLinkOrIdentityByPhone(ctx context.Context, normPhone, scopeRetailerID, scopeUserID string) (TelegramLinkRecord, bool, error) {
	if normPhone == "" {
		return TelegramLinkRecord{}, false, nil
	}
	if s.spannerClient != nil {
		stmt := spanner.Statement{
			SQL: `SELECT RetailerId, UserId, COALESCE(PhoneNumber, ''), COALESCE(TelegramChatId, ''),
				       COALESCE(OtpHash, ''), Locale, VerifiedAt, ExpiresAt, UpdatedAt
				FROM RetailerTelegramLinks
				WHERE PhoneNumber = @phone
				ORDER BY UpdatedAt DESC
				LIMIT 1`,
			Params: map[string]any{"phone": normPhone},
		}
		rec, found, err := s.scanSingleTelegramLink(ctx, stmt)
		if err != nil {
			return TelegramLinkRecord{}, false, err
		}
		if found && (scopeRetailerID == "" || rec.RetailerID == scopeRetailerID) {
			return rec, true, nil
		}
		// Fallback to registered RetailerUsers or Retailers by normalized phone.
		uStmt := spanner.Statement{
			SQL: `SELECT RetailerId, UserId, Phone
				FROM RetailerUsers
				WHERE Phone = @phone AND IsActive = TRUE
				ORDER BY IsOwner DESC, UpdatedAt DESC
				LIMIT 1`,
			Params: map[string]any{"phone": normPhone},
		}
		uIter := s.spannerClient.Single().Query(ctx, uStmt)
		if row, uErr := uIter.Next(); uErr == nil {
			var rid, uid, uPhone string
			if colErr := row.Columns(&rid, &uid, &uPhone); colErr == nil {
				uIter.Stop()
				if scopeRetailerID != "" && rid != scopeRetailerID {
					return TelegramLinkRecord{}, false, nil
				}
				return TelegramLinkRecord{RetailerID: rid, UserID: uid, PhoneNumber: NormalizeUZPhone(uPhone), Locale: "uz"}, true, nil
			}
		}
		uIter.Stop()

		rStmt := spanner.Statement{
			SQL: `SELECT RetailerId, Phone FROM Retailers WHERE Phone = @phone LIMIT 1`,
			Params: map[string]any{"phone": normPhone},
		}
		rIter := s.spannerClient.Single().Query(ctx, rStmt)
		defer rIter.Stop()
		if row, rErr := rIter.Next(); rErr == nil {
			var rid, rPhone string
			if colErr := row.Columns(&rid, &rPhone); colErr == nil {
				if scopeRetailerID != "" && rid != scopeRetailerID {
					return TelegramLinkRecord{}, false, nil
				}
				uid := scopeUserID
				if uid == "" {
					uid = rid
				}
				return TelegramLinkRecord{RetailerID: rid, UserID: uid, PhoneNumber: NormalizeUZPhone(rPhone), Locale: "uz"}, true, nil
			}
		}
		return TelegramLinkRecord{}, false, nil
	}

	// Memory store lookup.
	s.mu.RLock()
	for _, rec := range s.telegramLinksMem {
		if NormalizeUZPhone(rec.PhoneNumber) == normPhone && (scopeRetailerID == "" || rec.RetailerID == scopeRetailerID) {
			s.mu.RUnlock()
			return rec, true, nil
		}
	}
	for rid, owner := range s.ownerByRetailer {
		if NormalizeUZPhone(owner.Phone) == normPhone && (scopeRetailerID == "" || rid == scopeRetailerID) {
			s.mu.RUnlock()
			return TelegramLinkRecord{RetailerID: rid, UserID: owner.UserID, PhoneNumber: normPhone, Locale: "uz"}, true, nil
		}
	}
	for rid, staff := range s.staffByRetailer {
		for _, u := range staff {
			if NormalizeUZPhone(u.Phone) == normPhone && (scopeRetailerID == "" || rid == scopeRetailerID) {
				s.mu.RUnlock()
				return TelegramLinkRecord{RetailerID: rid, UserID: u.UserID, PhoneNumber: normPhone, Locale: "uz"}, true, nil
			}
		}
	}
	s.mu.RUnlock()

	if s.repo != nil {
		if ret, found, err := s.repo.FindByPhone(ctx, normPhone); err == nil && found {
			if scopeRetailerID == "" || ret.RetailerID == scopeRetailerID {
				uid := scopeUserID
				if uid == "" {
					uid = ret.RetailerID
				}
				return TelegramLinkRecord{RetailerID: ret.RetailerID, UserID: uid, PhoneNumber: normPhone, Locale: "uz"}, true, nil
			}
		}
	}
	return TelegramLinkRecord{}, false, nil
}

func (s *Service) scanSingleTelegramLink(ctx context.Context, stmt spanner.Statement) (TelegramLinkRecord, bool, error) {
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return TelegramLinkRecord{}, false, nil
	}
	if err != nil {
		return TelegramLinkRecord{}, false, err
	}
	var rec TelegramLinkRecord
	var verifiedAt, expiresAt spanner.NullTime
	if err := row.Columns(
		&rec.RetailerID, &rec.UserID, &rec.PhoneNumber, &rec.TelegramChatID,
		&rec.OtpHash, &rec.Locale, &verifiedAt, &expiresAt, &rec.UpdatedAt,
	); err != nil {
		return TelegramLinkRecord{}, false, err
	}
	if verifiedAt.Valid {
		t := verifiedAt.Time.UTC()
		rec.VerifiedAt = &t
	}
	if expiresAt.Valid {
		t := expiresAt.Time.UTC()
		rec.ExpiresAt = &t
	}
	rec.Locale = normalizeRetailerLocale(rec.Locale)
	return rec, true, nil
}

func (s *Service) upsertTelegramLink(ctx context.Context, rec TelegramLinkRecord) error {
	if s.spannerClient != nil {
		_, err := s.spannerClient.Apply(ctx, []*spanner.Mutation{telegramLinkMutation(rec)})
		return err
	}
	s.mu.Lock()
	s.telegramLinksMem[telegramLinkMemKey(rec.RetailerID, rec.UserID)] = rec
	s.mu.Unlock()
	return nil
}

func (s *Service) bindTelegramChat(ctx context.Context, rec TelegramLinkRecord) error {
	if s.spannerClient != nil {
		_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			// Clear previous link claiming the same TelegramChatId (UQ_RetailerTelegramLinks_ChatId).
			stmt := spanner.Statement{
				SQL: `SELECT RetailerId, UserId FROM RetailerTelegramLinks
					WHERE TelegramChatId = @chatId
					  AND NOT (RetailerId = @rid AND UserId = @uid)`,
				Params: map[string]any{
					"chatId": rec.TelegramChatID,
					"rid":    rec.RetailerID,
					"uid":    rec.UserID,
				},
			}
			iter := txn.Query(ctx, stmt)
			var muts []*spanner.Mutation
			for {
				row, nErr := iter.Next()
				if nErr == iterator.Done {
					break
				}
				if nErr != nil {
					iter.Stop()
					return nErr
				}
				var oldRID, oldUID string
				if colErr := row.Columns(&oldRID, &oldUID); colErr != nil {
					iter.Stop()
					return colErr
				}
				muts = append(muts, spanner.UpdateMap("RetailerTelegramLinks", map[string]any{
					"RetailerId":     oldRID,
					"UserId":         oldUID,
					"TelegramChatId": spanner.NullString{},
					"VerifiedAt":     spanner.NullTime{},
					"UpdatedAt":      spanner.CommitTimestamp,
				}))
			}
			iter.Stop()
			muts = append(muts, telegramLinkMutation(rec))
			return txn.BufferWrite(muts)
		})
		return err
	}

	s.mu.Lock()
	targetKey := telegramLinkMemKey(rec.RetailerID, rec.UserID)
	for k, existing := range s.telegramLinksMem {
		if k != targetKey && existing.TelegramChatID == rec.TelegramChatID {
			existing.TelegramChatID = ""
			existing.VerifiedAt = nil
			s.telegramLinksMem[k] = existing
		}
	}
	s.telegramLinksMem[targetKey] = rec
	s.mu.Unlock()
	return nil
}

func telegramLinkMutation(rec TelegramLinkRecord) *spanner.Mutation {
	cols := map[string]any{
		"RetailerId":     rec.RetailerID,
		"UserId":         rec.UserID,
		"PhoneNumber":    nullStringOrValue(rec.PhoneNumber),
		"TelegramChatId": nullStringOrValue(rec.TelegramChatID),
		"OtpHash":        nullStringOrValue(rec.OtpHash),
		"Locale":         normalizeRetailerLocale(rec.Locale),
		"UpdatedAt":      spanner.CommitTimestamp,
	}
	if rec.VerifiedAt != nil {
		cols["VerifiedAt"] = rec.VerifiedAt.UTC()
	} else {
		cols["VerifiedAt"] = spanner.NullTime{}
	}
	if rec.ExpiresAt != nil {
		cols["ExpiresAt"] = rec.ExpiresAt.UTC()
	} else {
		cols["ExpiresAt"] = spanner.NullTime{}
	}
	return spanner.InsertOrUpdateMap("RetailerTelegramLinks", cols)
}

func nullStringOrValue(v string) any {
	if strings.TrimSpace(v) == "" {
		return spanner.NullString{}
	}
	return strings.TrimSpace(v)
}

func (s *Service) lookupRetailerUserPhone(ctx context.Context, retailerID, userID string) string {
	if s.spannerClient != nil {
		iter := s.spannerClient.Single().Query(ctx, spanner.Statement{
			SQL: `SELECT COALESCE((SELECT Phone FROM RetailerUsers WHERE RetailerId = @rid AND UserId = @uid LIMIT 1),
			               (SELECT Phone FROM Retailers WHERE RetailerId = @rid LIMIT 1))`,
			Params: map[string]any{"rid": retailerID, "uid": userID},
		})
		defer iter.Stop()
		if row, err := iter.Next(); err == nil {
			var p spanner.NullString
			if row.Columns(&p) == nil && p.Valid {
				return NormalizeUZPhone(p.StringVal)
			}
		}
	}
	if s.repo != nil {
		if ret, found, err := s.repo.GetRetailer(ctx, retailerID); err == nil && found && ret.Phone != "" {
			return NormalizeUZPhone(ret.Phone)
		}
	}
	return ""
}

func (s *Service) lookupRetailerDisplayName(ctx context.Context, retailerID string) string {
	if s.spannerClient != nil {
		iter := s.spannerClient.Single().Query(ctx, spanner.Statement{
			SQL:    `SELECT COALESCE(NULLIF(Name, ''), RetailerId) FROM Retailers WHERE RetailerId = @rid LIMIT 1`,
			Params: map[string]any{"rid": retailerID},
		})
		defer iter.Stop()
		if row, err := iter.Next(); err == nil {
			var name string
			if row.Columns(&name) == nil && strings.TrimSpace(name) != "" {
				return strings.TrimSpace(name)
			}
		}
	}
	if s.repo != nil {
		if ret, found, err := s.repo.GetRetailer(ctx, retailerID); err == nil && found && strings.TrimSpace(ret.Name) != "" {
			return strings.TrimSpace(ret.Name)
		}
	}
	return retailerID
}

func generateSixDigitOTP() (string, error) {
	var buf [4]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	n := binary.BigEndian.Uint32(buf[:]) % 1000000
	return fmt.Sprintf("%06d", n), nil
}

func hashTelegramOTP(otp string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(otp)))
	return hex.EncodeToString(sum[:])
}

func isSupportedLocale(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "uz", "ru", "en":
		return true
	default:
		return false
	}
}
