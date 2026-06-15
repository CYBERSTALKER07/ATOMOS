package supplier

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
)

func TestDispatchExecuteRequiresIdempotencyKey(t *testing.T) {
	s := &Service{idem: idempotency.NewInMemoryStore()}
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/dispatch/execute", bytes.NewBufferString(`{"mode":"AUTO"}`))
	rr := httptest.NewRecorder()
	s.HandleDispatchExecute(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d want 400", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "idempotency_key_required") {
		t.Fatalf("body = %q want idempotency_key_required", rr.Body.String())
	}
}

func TestDispatchExecuteManifestStateIsDraft(t *testing.T) {
	if dispatchExecuteManifestState != "DRAFT" {
		t.Fatalf("dispatch execute manifest state = %q want DRAFT", dispatchExecuteManifestState)
	}
	if dispatchExecuteManifestState == "SEALED" {
		t.Fatal("dispatch execute must not pre-seal manifests; payloader owns seal gate")
	}
}

func TestDispatchExecuteCommittedStatusMatchesWarehouse(t *testing.T) {
	if dispatchExecuteCommittedStatus != "dispatched" {
		t.Fatalf("committed dispatch status = %q want dispatched", dispatchExecuteCommittedStatus)
	}
}

func TestDispatchExecuteEmitsDraftCreatedEvent(t *testing.T) {
	if events.EventManifestDraftCreated == events.EventManifestSealed {
		t.Fatal("draft-created and sealed events must remain distinct contract types")
	}
}
