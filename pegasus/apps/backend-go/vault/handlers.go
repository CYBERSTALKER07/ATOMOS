package vault

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"backend-go/auth"
	apierrors "backend-go/errors"
	"backend-go/payment"

	"cloud.google.com/go/spanner"
)

// HandlePaymentConfigs handles GET (list) and POST (upsert) for supplier payment configs.
// DELETE deactivates a config by config_id.
func HandlePaymentConfigs(client *spanner.Client) http.HandlerFunc {
	svc := &Service{Spanner: client}

	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		supplierId := claims.ResolveSupplierID()

		switch r.Method {
		case http.MethodGet:
			handleListConfigs(w, r, svc, supplierId)
		case http.MethodPost:
			handleUpsertConfig(w, r, svc, supplierId)
		case http.MethodDelete:
			handleDeactivateConfig(w, r, svc, supplierId)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

// paymentConfigResponse wraps configs with provider capability metadata
// so the frontend can render truthful connect vs manual states.
type paymentConfigResponse struct {
	Configs      []GatewayConfigSummary `json:"configs"`
	Capabilities []ProviderCapability   `json:"capabilities"`
}

func handleListConfigs(w http.ResponseWriter, r *http.Request, svc *Service, supplierId string) {
	configs, err := svc.ListConfigs(r.Context(), supplierId)
	if err != nil {
		slog.Error("vault.list_configs_failed", "supplier_id", supplierId, "err", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}
	if configs == nil {
		configs = []GatewayConfigSummary{}
	}
	resp := paymentConfigResponse{
		Configs:      configs,
		Capabilities: GetProviderCapabilities(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleUpsertConfig(w http.ResponseWriter, r *http.Request, svc *Service, supplierId string) {
	var req struct {
		GatewayName string `json:"gateway_name"`
		MerchantId  string `json:"merchant_id"`
		ServiceId   string `json:"service_id"`
		SecretKey   string `json:"secret_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		return
	}
	req.GatewayName = strings.ToUpper(req.GatewayName)

	result, err := svc.UpsertConfig(r.Context(), supplierId, req.GatewayName, req.MerchantId, req.ServiceId, req.SecretKey)
	if err != nil {
		slog.Error("vault.upsert_config_failed", "supplier_id", supplierId, "gateway", req.GatewayName, "err", err)
		if strings.Contains(err.Error(), "unsupported gateway") ||
			strings.Contains(err.Error(), "required") {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
			return
		}
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

func handleDeactivateConfig(w http.ResponseWriter, r *http.Request, svc *Service, supplierId string) {
	var req struct {
		ConfigId string `json:"config_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ConfigId == "" {
		http.Error(w, `{"error":"config_id required"}`, http.StatusBadRequest)
		return
	}

	if err := svc.DeactivateConfig(r.Context(), supplierId, req.ConfigId); err != nil {
		slog.Error("vault.deactivate_config_failed", "supplier_id", supplierId, "config_id", req.ConfigId, "err", err)
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "does not belong") {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusForbidden)
			return
		}
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "DEACTIVATED", "config_id": req.ConfigId})
}

// HandleRegisterRecipient registers the supplier as a Global Pay split-payment
// recipient and persists the returned RecipientId on SupplierPaymentConfigs.
func HandleRegisterRecipient(client *spanner.Client, gpDirect *payment.GlobalPayDirectClient) http.HandlerFunc {
	svc := &Service{Spanner: client}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			apierrors.MethodNotAllowed(w, r)
			return
		}
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims.UserID == "" {
			apierrors.Unauthorized(w, r, "missing or invalid authentication token")
			return
		}
		supplierId := claims.ResolveSupplierID()

		var req struct {
			Name         string `json:"name"`
			TIN          string `json:"tin"`
			BankAccount  string `json:"bank_account"`
			BankMFO      string `json:"bank_mfo"`
			ContactPhone string `json:"contact_phone"`
			ContactEmail string `json:"contact_email"`
			OKED         string `json:"oked"`
			LegalAddress string `json:"legal_address"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierrors.BadRequest(w, r, "invalid JSON body")
			return
		}

		// Resolve GP credentials from the supplier's existing vault config.
		cfg, err := svc.GetDecryptedConfig(r.Context(), supplierId, "GLOBAL_PAY")
		if err != nil {
			slog.Error("vault.register_recipient_missing_gateway_config", "supplier_id", supplierId, "err", err)
			apierrors.WriteOperational(w, r, apierrors.ProblemDetail{
				Type:       "error/payment/gateway-not-configured",
				Title:      "Payment Gateway Not Configured",
				Status:     http.StatusBadRequest,
				Detail:     "Global Pay credentials not configured — set up payment gateway first.",
				Code:       apierrors.CodeGatewayUnavailable,
				MessageKey: apierrors.MsgKeyGatewayDown,
				Action:     apierrors.ActionContactSupport,
			})
			return
		}

		creds := payment.GlobalPayCredentials{
			Username:  cfg.MerchantId,
			Password:  cfg.SecretKey,
			ServiceID: cfg.ServiceId,
		}

		result, err := gpDirect.RegisterRecipient(r.Context(), creds, payment.RecipientRegistration{
			Name:         req.Name,
			TIN:          req.TIN,
			BankAccount:  req.BankAccount,
			BankMFO:      req.BankMFO,
			ContactPhone: req.ContactPhone,
			ContactEmail: req.ContactEmail,
			OKED:         req.OKED,
			LegalAddress: req.LegalAddress,
		})
		if err != nil {
			slog.Error("vault.register_recipient_gateway_call_failed", "supplier_id", supplierId, "err", err)
			apierrors.WriteOperational(w, r, apierrors.ProblemDetail{
				Type:       "error/payment/recipient-registration-failed",
				Title:      "Recipient Registration Failed",
				Status:     http.StatusBadGateway,
				Detail:     "Global Pay rejected the recipient registration. Verify business details and try again.",
				Code:       apierrors.CodeGPRecipientFailed,
				MessageKey: apierrors.MsgKeyPaymentGeneric,
				Retryable:  true,
				Action:     apierrors.ActionRetry,
			})
			return
		}

		if err := svc.SetRecipientId(r.Context(), supplierId, "GLOBAL_PAY", result.RecipientID); err != nil {
			slog.Error("vault.register_recipient_persist_failed", "supplier_id", supplierId, "recipient_id", result.RecipientID, "err", err)
			apierrors.WriteOperational(w, r, apierrors.ProblemDetail{
				Type:       "error/payment/recipient-persist-failed",
				Title:      "Internal Error",
				Status:     http.StatusInternalServerError,
				Detail:     "Recipient registered but failed to persist — contact support.",
				Code:       apierrors.CodeGPRecipientFailed,
				MessageKey: apierrors.MsgKeyPaymentGeneric,
				Action:     apierrors.ActionContactSupport,
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(result)
	}
}
