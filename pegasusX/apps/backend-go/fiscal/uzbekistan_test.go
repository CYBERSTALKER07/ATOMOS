package fiscal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/soliq"
)

func TestUzbekistanStrategy_FailClosedWhenUnconfigured(t *testing.T) {
	ctx := context.Background()

	unconfigured := NewUzbekistanStrategy()
	formatted, err := unconfigured.FormatDocument(ctx, "esf-1")
	if err != nil {
		t.Fatalf("FormatDocument: %v", err)
	}
	if err := unconfigured.SubmitDocument(ctx, "esf-1", formatted); !errors.Is(err, ErrSoliqGatewayNotConfigured) {
		t.Fatalf("SubmitDocument err = %v, want ErrSoliqGatewayNotConfigured", err)
	}
	if err := unconfigured.CancelDocument(ctx, "esf-1"); !errors.Is(err, ErrSoliqGatewayNotConfigured) {
		t.Fatalf("CancelDocument err = %v, want ErrSoliqGatewayNotConfigured", err)
	}

	noSignerClient := soliq.NewClient(soliq.SoliqConfig{BaseURL: "http://127.0.0.1:1", TIN: "123456789"})
	missingSigner := NewUzbekistanStrategyWithDeps(noSignerClient, nil, "123456789", "direct")
	if err := missingSigner.SubmitDocument(ctx, "esf-2", formatted); !errors.Is(err, ErrEDSSignerNotConfigured) {
		t.Fatalf("SubmitDocument missing signer err = %v, want ErrEDSSignerNotConfigured", err)
	}
	if err := missingSigner.CancelDocument(ctx, "esf-2"); !errors.Is(err, ErrEDSSignerNotConfigured) {
		t.Fatalf("CancelDocument missing signer err = %v, want ErrEDSSignerNotConfigured", err)
	}
}

func TestUzbekistanStrategy_SignedSubmitAndCancelDelegation(t *testing.T) {
	ctx := context.Background()
	signer, err := NewDevHMACSigner([]byte("test-uzbekistan-eds-key-32-bytes!"))
	if err != nil {
		t.Fatalf("NewDevHMACSigner: %v", err)
	}

	var mu sync.Mutex
	type capturedReq struct {
		path string
		idem string
		body map[string]any
	}
	var captured []capturedReq
	rejectNext := false

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var decoded map[string]any
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.UseNumber()
		_ = dec.Decode(&decoded)

		mu.Lock()
		captured = append(captured, capturedReq{
			path: r.URL.Path,
			idem: r.Header.Get("Idempotency-Key"),
			body: decoded,
		})
		shouldReject := rejectNext
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		if shouldReject {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write([]byte(`{"success":false,"error":{"code":"INVALID_DOC","message":"rejected by soliq"}}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true,"data":{"ehf_id":"ehf-uz-9001"}}`))
	}))
	defer srv.Close()

	client := soliq.NewClient(soliq.SoliqConfig{
		BaseURL:  srv.URL,
		Operator: "direct",
		APIKey:   "test-key",
		TIN:      "301234567",
	})
	strategy := NewUzbekistanStrategyWithDeps(client, signer, "301234567", "direct")

	formatted, err := strategy.FormatDocument(ctx, "esf-100")
	if err != nil {
		t.Fatalf("FormatDocument: %v", err)
	}
	if err := strategy.SubmitDocument(ctx, "esf-100", formatted); err != nil {
		t.Fatalf("SubmitDocument: %v", err)
	}
	if err := strategy.CancelDocument(ctx, "esf-100"); err != nil {
		t.Fatalf("CancelDocument: %v", err)
	}

	// Upstream rejection must propagate as error (never fake nil).
	mu.Lock()
	rejectNext = true
	mu.Unlock()
	if err := strategy.SubmitDocument(ctx, "esf-101", formatted); err == nil {
		t.Fatal("expected error on upstream 422 rejection, got nil")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(captured) != 3 {
		t.Fatalf("captured %d calls, want 3", len(captured))
	}
	for i, req := range captured[:2] {
		sig, _ := req.body["signature"].(string)
		if sig == "" {
			t.Fatalf("call %d missing signature field", i)
		}
		// Verify HMAC signature against canonical unsigned copy.
		unsigned := make(map[string]any, len(req.body))
		for k, v := range req.body {
			if k != "signature" {
				unsigned[k] = v
			}
		}
		canon, err := MarshalCanonical(unsigned)
		if err != nil {
			t.Fatalf("MarshalCanonical: %v", err)
		}
		if !signer.Verify(canon, sig) {
			t.Fatalf("call %d signature failed verification", i)
		}
	}
	if captured[0].idem != "uz-submit:esf-100" || captured[0].body["action"] != "SUBMIT" {
		t.Fatalf("unexpected submit request: %+v", captured[0])
	}
	if captured[1].idem != "uz-cancel:esf-100" || captured[1].body["action"] != "CANCEL" {
		t.Fatalf("unexpected cancel request: %+v", captured[1])
	}
}
