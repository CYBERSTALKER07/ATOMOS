package retailerroutes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"backend-go/auth"
	"backend-go/order"
	"backend-go/payment"
	"backend-go/ws"
)

type retailerCardConfirmRequest struct {
	CardToken string `json:"card_token"`
	OTPCode   string `json:"otp_code"`
}

type retailerCardMutateRequest struct {
	TokenID string `json:"token_id"`
}

func handleRetailerCardInitiate(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if d.CardsClient == nil {
			http.Error(w, "card tokenization not configured", http.StatusServiceUnavailable)
			return
		}

		claims, ok := retailerClaims(r)
		if !ok {
			http.Error(w, "retailer identity missing from token", http.StatusUnauthorized)
			return
		}

		var req struct {
			Gateway string `json:"gateway"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Gateway == "" {
			req.Gateway = "GLOBAL_PAY"
		}

		phone, err := d.Order.LookupRetailerPhone(r.Context(), claims.UserID)
		if err != nil || phone == "" {
			http.Error(w, "retailer phone number required for card tokenization", http.StatusBadRequest)
			return
		}

		creds, err := payment.ResolveGlobalPayCredentials("", "", "")
		if err != nil {
			http.Error(w, "payment gateway credentials not configured", http.StatusServiceUnavailable)
			return
		}

		result, err := d.CardsClient.InitiateCardSave(r.Context(), creds, phone)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, result)
	}
}

func handleRetailerCashCheckout(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			OrderID string `json:"order_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == "" {
			http.Error(w, "order_id required", http.StatusBadRequest)
			return
		}

		resp, err := d.Order.CashCheckout(r.Context(), req.OrderID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		d.DriverHub.PushToDriver(resp.DriverID, map[string]interface{}{
			"type":     ws.EventCashCollectionRequired,
			"order_id": resp.OrderID,
			"amount":   resp.Amount,
			"message":  resp.Message,
		})

		writeJSON(w, resp)
	}
}

func handleRetailerCardCheckout(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			OrderID string `json:"order_id"`
			Gateway string `json:"gateway"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == "" || req.Gateway == "" {
			http.Error(w, "order_id and gateway required", http.StatusBadRequest)
			return
		}

		resp, err := d.Order.CardCheckout(r.Context(), req.OrderID, req.Gateway, requestBaseURL(r))
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		writeJSON(w, resp)
	}
}

func handleRetailerCardConfirm(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if d.CardsClient == nil {
			http.Error(w, "card tokenization not configured", http.StatusServiceUnavailable)
			return
		}

		claims, ok := retailerClaims(r)
		if !ok {
			http.Error(w, "retailer identity missing from token", http.StatusUnauthorized)
			return
		}

		var req retailerCardConfirmRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CardToken == "" || req.OTPCode == "" {
			http.Error(w, "card_token and otp_code required", http.StatusBadRequest)
			return
		}

		creds, err := payment.ResolveGlobalPayCredentials("", "", "")
		if err != nil {
			http.Error(w, "payment gateway credentials not configured", http.StatusServiceUnavailable)
			return
		}

		result, err := d.CardsClient.ConfirmCardOTP(r.Context(), creds, req.CardToken, req.OTPCode)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		if !result.Confirmed {
			http.Error(w, "OTP confirmation failed", http.StatusUnprocessableEntity)
			return
		}

		tokenID, err := d.CardTokenSvc.SaveCard(r.Context(), claims.UserID, "GLOBAL_PAY", req.CardToken, result.CardLast4, result.CardType)
		if err != nil {
			http.Error(w, "card confirmed but failed to save: "+err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, map[string]interface{}{
			"token_id":   tokenID,
			"card_last4": result.CardLast4,
			"card_type":  result.CardType,
			"confirmed":  true,
		})
	}
}

func handleRetailerCards(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := retailerClaims(r)
		if !ok {
			http.Error(w, "retailer identity missing from token", http.StatusUnauthorized)
			return
		}

		cards, err := d.CardTokenSvc.ListCards(r.Context(), claims.UserID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if cards == nil {
			cards = []payment.RetailerCardToken{}
		}

		writeJSON(w, map[string]interface{}{"cards": cards})
	}
}

func handleRetailerCardDeactivate(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := retailerClaims(r)
		if !ok {
			http.Error(w, "retailer identity missing from token", http.StatusUnauthorized)
			return
		}

		var req retailerCardMutateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TokenID == "" {
			http.Error(w, "token_id required", http.StatusBadRequest)
			return
		}

		if err := d.CardTokenSvc.DeactivateCard(r.Context(), req.TokenID, claims.UserID); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		writeJSON(w, map[string]string{"status": "deactivated"})
	}
}

func handleRetailerCardDefault(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := retailerClaims(r)
		if !ok {
			http.Error(w, "retailer identity missing from token", http.StatusUnauthorized)
			return
		}

		var req retailerCardMutateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TokenID == "" {
			http.Error(w, "token_id required", http.StatusBadRequest)
			return
		}

		if err := d.CardTokenSvc.SetDefaultCard(r.Context(), req.TokenID, claims.UserID); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		writeJSON(w, map[string]string{"status": "default_set"})
	}
}

func handleRetailerPendingPayments(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := retailerClaims(r)
		if !ok {
			http.Error(w, "retailer identity missing from token", http.StatusUnauthorized)
			return
		}

		sessions, err := d.SessionSvc.GetPendingSessionsByRetailer(r.Context(), claims.UserID)
		if err != nil {
			http.Error(w, "failed to retrieve pending payments", http.StatusInternalServerError)
			return
		}
		if sessions == nil {
			sessions = []payment.PaymentSession{}
		}

		writeJSON(w, map[string]interface{}{
			"pending_payments": sessions,
			"count":            len(sessions),
		})
	}
}

func handleRetailerActiveFulfillment(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := retailerClaims(r)
		if !ok {
			http.Error(w, "retailer identity missing from token", http.StatusUnauthorized)
			return
		}

		items, err := d.Order.ActiveFulfillments(r.Context(), claims.UserID)
		if err != nil {
			http.Error(w, "failed to retrieve active fulfillments", http.StatusInternalServerError)
			return
		}
		if items == nil {
			items = []order.ActiveFulfillmentItem{}
		}

		writeJSON(w, map[string]interface{}{
			"fulfillments": items,
			"count":        len(items),
		})
	}
}

func retailerClaims(r *http.Request) (*auth.PegasusClaims, bool) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims == nil || claims.UserID == "" {
		return nil, false
	}
	return claims, true
}

func writeJSON(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func requestBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwarded != "" {
		scheme = strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	host := strings.TrimSpace(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = r.Host
	}
	if host == "" {
		return ""
	}
	return fmt.Sprintf("%s://%s", scheme, host)
}
