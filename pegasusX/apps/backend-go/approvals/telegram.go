package approvals

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

const (
	headerTelegramSecretToken = "X-Telegram-Bot-Api-Secret-Token"
	headerBotWebhookSecret    = "X-Bot-Webhook-Secret"
	headerTelegramSignature   = "X-Telegram-Signature"
)

// TelegramApproverBinding maps a linked Telegram chat ID to a supplier approver identity.
type TelegramApproverBinding struct {
	TenantID  string `json:"tenant_id"`
	ActorID   string `json:"actor_id"`
	Role      string `json:"role"`
	ActorType string `json:"actor_type,omitempty"`
}

type telegramWebhookPayload struct {
	UpdateID      int64 `json:"update_id,omitempty"`
	CallbackQuery *struct {
		ID   string `json:"id"`
		From struct {
			ID       int64  `json:"id"`
			Username string `json:"username"`
			IsBot    bool   `json:"is_bot"`
		} `json:"from"`
		Message *struct {
			Chat struct {
				ID int64 `json:"id"`
			} `json:"chat"`
		} `json:"message,omitempty"`
		Data string `json:"data"`
	} `json:"callback_query,omitempty"`
	CallbackData   string `json:"callback_data,omitempty"`
	TelegramChatID string `json:"telegram_chat_id,omitempty"`
	ApproverID     string `json:"approver_id,omitempty"`
	ApproverRole   string `json:"approver_role,omitempty"`
	TenantID       string `json:"tenant_id,omitempty"`
	Comment        string `json:"comment,omitempty"`
	PayloadHash    string `json:"payload_hash,omitempty"`
	MFAVerified    bool   `json:"mfa_verified,omitempty"`
}

func (s *Service) resolveTelegramSecret() string {
	if s != nil && strings.TrimSpace(s.telegramSecret) != "" {
		return strings.TrimSpace(s.telegramSecret)
	}
	for _, k := range []string{
		"TELEGRAM_APPROVAL_WEBHOOK_SECRET",
		"TELEGRAM_BOT_WEBHOOK_SECRET",
		"BOT_WEBHOOK_SECRET",
	} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

func verifyTelegramWebhookAuth(r *http.Request, body []byte, expectedSecret string) bool {
	if expectedSecret == "" {
		return false
	}
	if tok := strings.TrimSpace(r.Header.Get(headerTelegramSecretToken)); tok != "" {
		if subtle.ConstantTimeCompare([]byte(tok), []byte(expectedSecret)) == 1 {
			return true
		}
	}
	if tok := strings.TrimSpace(r.Header.Get(headerBotWebhookSecret)); tok != "" {
		if subtle.ConstantTimeCompare([]byte(tok), []byte(expectedSecret)) == 1 {
			return true
		}
	}
	if sig := strings.TrimSpace(r.Header.Get(headerTelegramSignature)); sig != "" {
		sigHex := strings.TrimPrefix(strings.ToLower(sig), "sha256=")
		sigBytes, err := hex.DecodeString(sigHex)
		if err != nil {
			return false
		}
		mac := hmac.New(sha256.New, []byte(expectedSecret))
		_, _ = mac.Write(body)
		return hmac.Equal(sigBytes, mac.Sum(nil))
	}
	return false
}

// ParseTelegramCallbackData parses "approve:<requestId>" or "reject:<requestId>".
func ParseTelegramCallbackData(data string) (decision, requestID string, err error) {
	parts := strings.SplitN(strings.TrimSpace(data), ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("%w: expected callback format approve:<id> or reject:<id>", ErrInvalidInput)
	}
	act := strings.ToLower(strings.TrimSpace(parts[0]))
	reqID := strings.TrimSpace(parts[1])
	if reqID == "" {
		return "", "", fmt.Errorf("%w: callback request id is empty", ErrInvalidInput)
	}
	switch act {
	case "approve":
		return DecisionApprove, reqID, nil
	case "reject":
		return DecisionReject, reqID, nil
	default:
		return "", "", fmt.Errorf("%w: unsupported callback action %q", ErrInvalidInput, parts[0])
	}
}

func resolveTelegramBindingFromEnv(chatID string) (TelegramApproverBinding, bool) {
	raw := strings.TrimSpace(os.Getenv("TELEGRAM_APPROVAL_CHAT_MAP"))
	if raw == "" {
		return TelegramApproverBinding{}, false
	}
	var m map[string]TelegramApproverBinding
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return TelegramApproverBinding{}, false
	}
	b, ok := m[strings.TrimSpace(chatID)]
	return b, ok
}

// HandleTelegramWebhook serves POST /v1/webhooks/telegram/approvals.
// Verifies Telegram secret token or HMAC signature fail-closed, parses inline
// callback actions ("approve:<id>" / "reject:<id>"), resolves the linked human
// approver, enforces SoD/MFA/PayloadHash, and records AuditEvents.
func (s *Service) HandleTelegramWebhook(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "approvals_unavailable", "approvals service unavailable")
		return
	}
	if r.Body == nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "request body required")
		return
	}
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 256*1024))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "read_body_failed", "unable to read body")
		return
	}

	if !verifyTelegramWebhookAuth(r, body, s.resolveTelegramSecret()) {
		writeDomainError(w, ErrWebhookUnauthorized)
		return
	}

	var p telegramWebhookPayload
	if err := json.Unmarshal(body, &p); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return
	}

	callbackData := strings.TrimSpace(p.CallbackData)
	chatID := strings.TrimSpace(p.TelegramChatID)
	username := ""
	isBot := false
	if p.CallbackQuery != nil {
		if callbackData == "" {
			callbackData = strings.TrimSpace(p.CallbackQuery.Data)
		}
		if p.CallbackQuery.From.IsBot {
			isBot = true
		}
		username = strings.TrimSpace(p.CallbackQuery.From.Username)
		if chatID == "" {
			if p.CallbackQuery.Message != nil && p.CallbackQuery.Message.Chat.ID != 0 {
				chatID = strconv.FormatInt(p.CallbackQuery.Message.Chat.ID, 10)
			} else if p.CallbackQuery.From.ID != 0 {
				chatID = strconv.FormatInt(p.CallbackQuery.From.ID, 10)
			}
		}
	}
	if isBot {
		writeDomainError(w, ErrAgentApprovalForbidden)
		return
	}

	decision, reqID, err := ParseTelegramCallbackData(callbackData)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	tenantID := strings.TrimSpace(p.TenantID)
	approverID := strings.TrimSpace(p.ApproverID)
	approverRole := strings.ToUpper(strings.TrimSpace(p.ApproverRole))
	actorType := auth.ActorTypeHuman

	if chatID != "" && s.telegramResolver != nil {
		rTenant, rActor, rRole, rType, ok, rErr := s.telegramResolver.ResolveTelegramApprover(r.Context(), chatID, username)
		if rErr != nil {
			writeDomainError(w, rErr)
			return
		}
		if ok {
			tenantID = coalesceNonEmpty(rTenant, tenantID)
			approverID = coalesceNonEmpty(rActor, approverID)
			approverRole = coalesceNonEmpty(strings.ToUpper(rRole), approverRole)
			if strings.TrimSpace(rType) != "" {
				actorType = strings.ToLower(strings.TrimSpace(rType))
			}
		}
	}
	if approverID == "" && chatID != "" {
		if b, ok := resolveTelegramBindingFromEnv(chatID); ok {
			tenantID = coalesceNonEmpty(b.TenantID, tenantID)
			approverID = strings.TrimSpace(b.ActorID)
			approverRole = coalesceNonEmpty(strings.ToUpper(b.Role), approverRole)
			if strings.TrimSpace(b.ActorType) != "" {
				actorType = strings.ToLower(strings.TrimSpace(b.ActorType))
			}
		}
	}
	if approverID == "" {
		writeJSONError(w, http.StatusForbidden, "telegram_approver_not_linked", "telegram chat is not linked to an authorized approver")
		return
	}
	if actorType == auth.ActorTypeAgent || approverRole == auth.SupplierRoleAgent {
		writeDomainError(w, ErrAgentApprovalForbidden)
		return
	}
	if approverRole == auth.SupplierRoleAuditor {
		writeDomainError(w, ErrAuditorReadOnly)
		return
	}

	comment := strings.TrimSpace(p.Comment)
	if comment == "" {
		if chatID != "" {
			comment = "telegram_inline:" + chatID
		} else {
			comment = "telegram_inline"
		}
	}

	res, err := s.Decide(r.Context(), DecideInput{
		TenantID:     tenantID,
		RequestID:    reqID,
		ApproverID:   approverID,
		ActorType:    actorType,
		ApproverRole: approverRole,
		Decision:     decision,
		Comment:      comment,
		ExpectedHash: p.PayloadHash,
		MFAVerified:  p.MFAVerified,
		RoutePath:    r.URL.Path,
		TraceID:      strings.TrimSpace(r.Header.Get("X-Trace-Id")),
		IP:           r.RemoteAddr,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   res.Request.TenantID,
		Action:     "approvals.telegram." + strings.ToLower(decision),
		ObjectType: "ApprovalRequest",
		ObjectID:   reqID,
		ApprovalID: reqID,
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"request":  res.Request,
		"decision": res.Decision,
	})
}
