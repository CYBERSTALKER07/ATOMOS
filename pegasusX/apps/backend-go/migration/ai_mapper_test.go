package migration

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/migration/parser"
)

func TestAIMapper_GeminiPIIRedactionAndStubSuccess(t *testing.T) {
	var capturedPrompt atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		capturedPrompt.Store(string(body))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{
				{
					"content": map[string]any{
						"parts": []map[string]string{
							{"text": `{"mappings":[{"source_column":"Kl_Nomi","target_field":"customer_name","confidence":0.91,"reason":"Klient nomi = customer name"},{"source_column":"Col_X","target_field":"inn","confidence":0.89,"reason":"9-digit tax id"}]}`},
						},
					},
				},
			},
			"usageMetadata": map[string]int{"totalTokenCount": 420},
		})
	}))
	defer srv.Close()

	mapper := NewAIMapper(GeminiMapperConfig{
		Enabled:          true,
		Endpoint:         srv.URL,
		APIKey:           "test-key",
		Model:            "gemini-2.5-flash",
		TargetTimeout:    2 * time.Second,
		HardTimeout:      5 * time.Second,
		DailyTokenBudget: 5000,
	})

	samples := []parser.ParsedRow{
		{
			RowIndex: 0,
			Fields: map[string]any{
				"Kl_Nomi": "Baraka Trade",
				"Col_X":   "301234567",
				"Tel_Col": "+998901234567",
				"Mail":    "owner@baraka.uz",
			},
		},
	}

	doc := mapper.ProposeMappings(context.Background(), "sup-1", "cell-uz", "sess-1", PackCustomers,
		[]string{"Kl_Nomi", "Col_X", "Tel_Col", "Mail"}, samples, nil)

	if doc.AIStatus != "GEMINI_PROPOSED" || !doc.AllRequiredMapped || doc.TokensUsed != 420 {
		t.Fatalf("expected GEMINI_PROPOSED ready doc, got %+v", doc)
	}
	sentPrompt, _ := capturedPrompt.Load().(string)
	for _, secret := range []string{"301234567", "+998901234567", "owner@baraka.uz"} {
		if strings.Contains(sentPrompt, secret) {
			t.Fatalf("raw PII %q leaked into Gemini prompt: %s", secret, sentPrompt)
		}
	}
	if !strings.Contains(sentPrompt, "REDACTED_INN:") || !strings.Contains(sentPrompt, "REDACTED_PHONE:") || !strings.Contains(sentPrompt, "REDACTED_EMAIL:") {
		t.Fatalf("expected redacted hashes in Gemini prompt, got: %s", sentPrompt)
	}
}

func TestAIMapper_TimeoutQuotaAndCellToggleFallbacks(t *testing.T) {
	slowSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(150 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer slowSrv.Close()

	// 1. Forced timeout -> clean fallback without failing request.
	mapper := NewAIMapper(GeminiMapperConfig{
		Enabled:          true,
		Endpoint:         slowSrv.URL,
		APIKey:           "test-key",
		TargetTimeout:    25 * time.Millisecond,
		HardTimeout:      50 * time.Millisecond,
		DailyTokenBudget: 10000,
	})
	doc := mapper.ProposeMappings(context.Background(), "sup-1", "cell-uz", "s1", PackCustomers,
		[]string{"Customer", "Col_9Digit", "Col_Tel", "Fld_ZZZ"},
		[]parser.ParsedRow{{Fields: map[string]any{"Customer": "Alpomish", "Col_9Digit": "301234567", "Col_Tel": "+998911234567", "Fld_ZZZ": "XYZ"}}},
		nil)
	if doc.AIStatus != "FALLBACK_TIMEOUT" {
		t.Fatalf("expected FALLBACK_TIMEOUT, got %s", doc.AIStatus)
	}
	if !doc.AllRequiredMapped {
		t.Fatalf("expected heuristic value inference to cover required customer fields on timeout, got %+v", doc)
	}

	// 2. Disabled cell toggle -> FALLBACK_DISABLED
	mapperDisabledCell := NewAIMapper(GeminiMapperConfig{
		Enabled:       true,
		DisabledCells: map[string]bool{"cell-airgap": true},
		Endpoint:      slowSrv.URL,
		APIKey:        "test-key",
	})
	docCell := mapperDisabledCell.ProposeMappings(context.Background(), "sup-1", "cell-airgap", "s2", PackCustomers, []string{"Col1"}, nil, nil)
	if docCell.AIStatus != "FALLBACK_DISABLED" {
		t.Fatalf("expected FALLBACK_DISABLED for blocked cell, got %s", docCell.AIStatus)
	}

	// 3. Daily tenant token budget exhaustion -> FALLBACK_QUOTA
	mapperQuota := NewAIMapper(GeminiMapperConfig{
		Enabled:          true,
		Endpoint:         slowSrv.URL,
		APIKey:           "test-key",
		DailyTokenBudget: 100, // less than ~400 est tokens
	})
	docQuota := mapperQuota.ProposeMappings(context.Background(), "sup-1", "cell-uz", "s3", PackCustomers, []string{"Col1"}, nil, nil)
	if docQuota.AIStatus != "FALLBACK_QUOTA" {
		t.Fatalf("expected FALLBACK_QUOTA when daily budget exceeded, got %s", docQuota.AIStatus)
	}
}

func TestConfirmSessionMapping_AcceptAboveAndLibraryHitAndBlockIncomplete(t *testing.T) {
	repo := newMemIngestRepo()
	repo.sessions["sess-1"] = Session{
		SupplierID: "sup-1",
		SessionID:  "sess-1",
		Status:     SessionStatusMappingProposed,
		Pack:       PackProductsPrices,
	}

	// Proposed headers where SKU & Name have high confidence, unit is 0.70 (below 0.80), price missing.
	mapper := NewAIMapper(GeminiMapperConfig{Enabled: false})
	proposed := mapper.ProposeMappings(context.Background(), "sup-1", "cell-uz", "sess-1", PackProductsPrices,
		[]string{"Артикул", "Наименование", "PriceCol"}, nil, nil)
	if err := repo.SaveSessionMapping(context.Background(), "sup-1", "sess-1", SessionStatusMappingProposed, proposed); err != nil {
		t.Fatal(err)
	}

	// Strict confirm failing when PriceCol isn't mapped if required (actually products_prices requires name + (sku|barcode)),
	// so test missing SKU on a session where SKU was unmapped.
	repo.sessions["sess-bad"] = Session{SupplierID: "sup-1", SessionID: "sess-bad", Status: SessionStatusMappingRequired, Pack: PackLots}
	if err := repo.SaveSessionMapping(context.Background(), "sup-1", "sess-bad", SessionStatusMappingRequired, SessionMappingDocument{
		SessionID: "sess-bad",
		Pack:      PackLots,
		Mappings: []ColumnMappingProposal{
			{SourceColumn: "A", TargetField: "sku", Confidence: 0.90, Confirmed: true},
			{SourceColumn: "B", TargetField: "lot_number", Confidence: 0.72, Confirmed: false}, // < 0.80!
		},
	}); err != nil {
		t.Fatal(err)
	}

	// Confirm with accept_above_confidence=0.80 strictly -> blocked because lot_number is 0.72 and expiry/qty missing!
	thresh := 0.80
	_, _, err := ConfirmSessionMapping(context.Background(), repo, "sup-1", "sess-bad", "admin@sup", ConfirmMappingRequest{
		AcceptAboveConfidence: &thresh,
		Strict:                true,
	}, time.Now().UTC())
	if !errors.Is(err, ErrIncompleteRequiredMappings) {
		t.Fatalf("expected ErrIncompleteRequiredMappings on strict confirm < 0.80, got %v", err)
	}

	// Now confirm sess-1 with accept_above_confidence=0.80 + manual override for PriceCol -> writes MappingLibrary!
	updatedSess, confirmedDoc, err := ConfirmSessionMapping(context.Background(), repo, "sup-1", "sess-1", "admin@sup", ConfirmMappingRequest{
		AcceptAboveConfidence: &thresh,
		Mappings: []ColumnMappingProposal{
			{SourceColumn: "PriceCol", TargetField: "price"},
		},
		Strict: true,
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("ConfirmSessionMapping sess-1: %v", err)
	}
	if updatedSess.Status != SessionStatusDiscovered || !confirmedDoc.AllRequiredMapped {
		t.Fatalf("expected DISCOVERED & ready doc, got status=%s doc=%+v", updatedSess.Status, confirmedDoc)
	}

	// Upload a SECOND file with identical header fingerprint -> instant LIBRARY_HIT -> DISCOVERED!
	repo.sessions["sess-2"] = Session{
		SupplierID: "sup-1",
		ProjectID:  "proj-1",
		BundleID:   "bnd-repeat",
		SessionID:  "sess-2",
		Status:     SessionStatusMappingRequired,
		Pack:       PackProductsPrices,
	}
	_, _ = repo.SaveStagedRowsChunk(context.Background(), "sup-1", "sess-2", []StagedRow{
		{SupplierID: "sup-1", SessionID: "sess-2", RowIndex: 0, RawData: json.RawMessage(`{"Артикул":"S1","Наименование":"Milk","PriceCol":"9000"}`)},
	})
	docs, err := ProposeMappingsForBundle(context.Background(), repo, mapper, "sup-1", "cell-uz", "proj-1", "bnd-repeat")
	if err != nil {
		t.Fatalf("ProposeMappingsForBundle repeat: %v", err)
	}
	if len(docs) != 1 || docs[0].AIStatus != "LIBRARY_HIT" || docs[0].Mappings[0].Confidence != 0.99 {
		t.Fatalf("expected repeat fingerprint LIBRARY_HIT, got %+v", docs)
	}
	sess2, _ := repo.GetSession(context.Background(), "sup-1", "sess-2")
	if sess2.Status != SessionStatusDiscovered {
		t.Fatalf("expected sess-2 auto-advanced to DISCOVERED on LIBRARY_HIT, got %s", sess2.Status)
	}
}
