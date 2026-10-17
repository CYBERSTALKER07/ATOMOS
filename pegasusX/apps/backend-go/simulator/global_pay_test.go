package simulator_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/simulator"
)

func setupTestRouter() (*chi.Mux, *simulator.Handler) {
	r := chi.NewRouter()
	sim := simulator.NewHandler("whsec-test-secret", "http://localhost:8080", slog.New(slog.NewTextHandler(io.Discard, nil)))
	r.Route("/sim/globalpay", func(sub chi.Router) {
		simulator.RegisterRoutes(sub, sim)
	})
	return r, sim
}

func TestGlobalPay_Auth(t *testing.T) {
	r, _ := setupTestRouter()

	body, _ := json.Marshal(map[string]string{
		"username": "gp-merchant-test",
		"password": "secret-password",
	})
	req := httptest.NewRequest(http.MethodPost, "/sim/globalpay/v1/merchant/auth", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if !strings.HasPrefix(resp["access_token"], "sim-access-token-") {
		t.Fatalf("expected access token, got %v", resp)
	}
}

func TestGlobalPay_TokenCreationAndCheckoutProcess(t *testing.T) {
	r, _ := setupTestRouter()

	// 1. Create Token
	tokenReqBody, _ := json.Marshal(map[string]any{
		"service_id":   "svc-test",
		"order_id":     "ord-unit-1001",
		"amount_minor": 50000000, // 500,000 UZS
		"currency":     "UZS",
	})
	req := httptest.NewRequest(http.MethodPost, "/sim/globalpay/v1/user-service-tokens", bytes.NewReader(tokenReqBody))
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("token creation failed: %d: %s", w.Code, w.Body.String())
	}
	var tokenResp struct {
		Token           string `json:"token"`
		UserRedirectURL string `json:"userRedirectUrl"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &tokenResp)
	if tokenResp.Token == "" {
		t.Fatal("expected token in response")
	}

	// 2. Load Checkout HTML UI
	uiReq := httptest.NewRequest(http.MethodGet, "/sim/globalpay/checkout?token="+tokenResp.Token, nil)
	uiW := httptest.NewRecorder()
	r.ServeHTTP(uiW, uiReq)
	if uiW.Code != http.StatusOK {
		t.Fatalf("checkout UI failed: %d", uiW.Code)
	}
	if !strings.Contains(uiW.Body.String(), "GlobalPay") || !strings.Contains(uiW.Body.String(), "500 000") {
		t.Fatalf("checkout UI missing GlobalPay branding or formatted amount")
	}

	// 3. Process Checkout with Card & OTP
	formData := url.Values{}
	formData.Set("token", tokenResp.Token)
	formData.Set("card_id", "sim-card-01")
	formData.Set("otp_code", "123456")

	procReq := httptest.NewRequest(http.MethodPost, "/sim/globalpay/checkout/process", strings.NewReader(formData.Encode()))
	procReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	procW := httptest.NewRecorder()
	r.ServeHTTP(procW, procReq)

	if procW.Code != http.StatusOK {
		t.Fatalf("checkout process failed: %d: %s", procW.Code, procW.Body.String())
	}
	if !strings.Contains(procW.Body.String(), "To'lov Muvaffaqiyatli Amalga Oshirildi") {
		t.Fatalf("expected successful payment receipt in Uzbek")
	}
}

func TestGlobalPay_P2PMoneyTransfer(t *testing.T) {
	r, _ := setupTestRouter()

	// 1. Initiate P2P Transfer: UzCard (sim-card-01) -> Humo (9860123456780002)
	// Amount: 1,000,000 UZS (100,000,000 tiyin)
	// Fee 0.5%: 5,000 UZS (500,000 tiyin)
	// Total: 1,005,000 UZS
	transferInitBody, _ := json.Marshal(map[string]any{
		"sender_card_id": "sim-card-01",
		"recipient_pan":  "9860123456780002",
		"amount_minor":   100000000,
		"currency":       "UZS",
	})
	initReq := httptest.NewRequest(http.MethodPost, "/sim/globalpay/v1/transfers/p2p", bytes.NewReader(transferInitBody))
	initW := httptest.NewRecorder()
	r.ServeHTTP(initW, initReq)

	if initW.Code != http.StatusOK {
		t.Fatalf("p2p init failed: %d: %s", initW.Code, initW.Body.String())
	}
	var initResp struct {
		TransferID  string `json:"transfer_id"`
		FeeMinor    int64  `json:"fee_minor"`
		TotalMinor  int64  `json:"total_minor"`
		OTPCodeHint string `json:"otp_hint"`
	}
	_ = json.Unmarshal(initW.Body.Bytes(), &initResp)
	if initResp.TransferID == "" {
		t.Fatal("expected transfer_id")
	}
	if initResp.FeeMinor != 500000 {
		t.Fatalf("expected 500000 tiyin fee (0.5%% of 100M), got %d", initResp.FeeMinor)
	}
	if initResp.TotalMinor != 100500000 {
		t.Fatalf("expected 100500000 total minor, got %d", initResp.TotalMinor)
	}

	// 2. Confirm P2P Transfer with OTP
	confirmBody, _ := json.Marshal(map[string]string{
		"transfer_id": initResp.TransferID,
		"otp_code":    "123456",
	})
	confReq := httptest.NewRequest(http.MethodPost, "/sim/globalpay/v1/transfers/p2p/confirm", bytes.NewReader(confirmBody))
	confW := httptest.NewRecorder()
	r.ServeHTTP(confW, confReq)

	if confW.Code != http.StatusOK {
		t.Fatalf("p2p confirm failed: %d: %s", confW.Code, confW.Body.String())
	}
	var confResp struct {
		Status        string `json:"status"`
		RRN           string `json:"rrn"`
		SenderBalance int64  `json:"sender_balance"`
	}
	_ = json.Unmarshal(confW.Body.Bytes(), &confResp)
	if confResp.Status != "COMPLETED" {
		t.Fatalf("expected COMPLETED status, got %s", confResp.Status)
	}
	// Initial balance: 1,000,000,000 tiyin - 100,500,000 tiyin = 899,500,000 tiyin
	if confResp.SenderBalance != 899500000 {
		t.Fatalf("expected sender balance 899500000, got %d", confResp.SenderBalance)
	}

	// 3. Query receipt
	getReq := httptest.NewRequest(http.MethodGet, "/sim/globalpay/v1/transfers/"+initResp.TransferID, nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("p2p get failed: %d", getW.Code)
	}
}

func TestGlobalPay_CardLookup(t *testing.T) {
	r, _ := setupTestRouter()

	testCases := []struct {
		pan         string
		wantScheme  string
		wantBank    string
	}{
		{"8600 1234 5678 0001", "UZCARD", "O'zsanoatqurilishbank"},
		{"9860 1234 5678 0002", "HUMO", "Ipak Yo'li Bank"},
		{"4242 4242 4242 4242", "VISA", "Kapitalbank"},
		{"5555 5555 5555 5555", "MASTERCARD", "NBU (Milliy Bank)"},
	}

	for _, tc := range testCases {
		body, _ := json.Marshal(map[string]string{"pan": tc.pan})
		req := httptest.NewRequest(http.MethodPost, "/sim/globalpay/v1/cards/lookup", bytes.NewReader(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("lookup failed for %s: %d", tc.pan, w.Code)
		}
		var resp struct {
			Scheme   string `json:"scheme"`
			BankName string `json:"bank_name"`
			IsValid  bool   `json:"is_valid"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Scheme != tc.wantScheme {
			t.Errorf("pan %s: expected scheme %s, got %s", tc.pan, tc.wantScheme, resp.Scheme)
		}
		if resp.BankName != tc.wantBank {
			t.Errorf("pan %s: expected bank %s, got %s", tc.pan, tc.wantBank, resp.BankName)
		}
		if !resp.IsValid {
			t.Errorf("pan %s: expected is_valid = true", tc.pan)
		}
	}
}

func TestGlobalPay_CardTokenizationAndDirectPay(t *testing.T) {
	r, _ := setupTestRouter()

	// 1. Bind Card session
	newReqBody, _ := json.Marshal(map[string]string{
		"pan":    "8600123456780001",
		"expiry": "12/28",
	})
	newReq := httptest.NewRequest(http.MethodPost, "/sim/globalpay/v1/cards/new", bytes.NewReader(newReqBody))
	newW := httptest.NewRecorder()
	r.ServeHTTP(newW, newReq)

	if newW.Code != http.StatusOK {
		t.Fatalf("cards/new failed: %d: %s", newW.Code, newW.Body.String())
	}
	var newResp struct {
		BindingID string `json:"binding_id"`
	}
	_ = json.Unmarshal(newW.Body.Bytes(), &newResp)

	// 2. Confirm Bind with OTP
	bindBody, _ := json.Marshal(map[string]string{
		"binding_id": newResp.BindingID,
		"otp_code":   "123456",
	})
	bindReq := httptest.NewRequest(http.MethodPost, "/sim/globalpay/v1/cards/bind", bytes.NewReader(bindBody))
	bindW := httptest.NewRecorder()
	r.ServeHTTP(bindW, bindReq)

	if bindW.Code != http.StatusOK {
		t.Fatalf("cards/bind failed: %d: %s", bindW.Code, bindW.Body.String())
	}
	var bindResp struct {
		CardToken string `json:"card_token"`
		Scheme    string `json:"scheme"`
	}
	_ = json.Unmarshal(bindW.Body.Bytes(), &bindResp)
	if !strings.HasPrefix(bindResp.CardToken, "gp_tok_") {
		t.Fatalf("expected card token, got %v", bindResp)
	}

	// 3. Direct Charge via Token
	chargeBody, _ := json.Marshal(map[string]any{
		"card_token":   bindResp.CardToken,
		"order_id":     "ord-tok-999",
		"amount_minor": 20000000, // 200,000 UZS
		"currency":     "UZS",
	})
	chargeReq := httptest.NewRequest(http.MethodPost, "/sim/globalpay/v1/payments/token", bytes.NewReader(chargeBody))
	chargeW := httptest.NewRecorder()
	r.ServeHTTP(chargeW, chargeReq)

	if chargeW.Code != http.StatusOK {
		t.Fatalf("payments/token failed: %d: %s", chargeW.Code, chargeW.Body.String())
	}
	var chargeResp struct {
		Status       string `json:"status"`
		IsSuccessful bool   `json:"isSuccessful"`
	}
	_ = json.Unmarshal(chargeW.Body.Bytes(), &chargeResp)
	if chargeResp.Status != "SUCCESS" || !chargeResp.IsSuccessful {
		t.Fatalf("expected successful token charge, got %v", chargeResp)
	}
}

func TestGlobalPay_DeclinedCard(t *testing.T) {
	r, _ := setupTestRouter()

	// Create token
	tokenReqBody, _ := json.Marshal(map[string]any{
		"service_id":   "svc-test",
		"order_id":     "ord-decline-1",
		"amount_minor": 10000000,
		"currency":     "UZS",
	})
	req := httptest.NewRequest(http.MethodPost, "/sim/globalpay/v1/user-service-tokens", bytes.NewReader(tokenReqBody))
	req.Header.Set("Authorization", "Bearer valid")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var tokenResp struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &tokenResp)

	// Process with declined card (sim-card-03)
	formData := url.Values{}
	formData.Set("token", tokenResp.Token)
	formData.Set("card_id", "sim-card-03")
	formData.Set("otp_code", "123456")

	procReq := httptest.NewRequest(http.MethodPost, "/sim/globalpay/checkout/process", strings.NewReader(formData.Encode()))
	procReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	procW := httptest.NewRecorder()
	r.ServeHTTP(procW, procReq)

	if procW.Code != http.StatusOK {
		t.Fatalf("expected 200 result page, got %d", procW.Code)
	}
	if !strings.Contains(procW.Body.String(), "To'lov Rad Etildi") {
		t.Fatalf("expected declined receipt in Uzbek")
	}
}
