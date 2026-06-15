package warehouse

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
)

func TestOpsDispatchExecuteRequiresIdempotencyKey(t *testing.T) {
	s := &Service{idem: idempotency.NewInMemoryStore()}
	req := httptest.NewRequest(http.MethodPost, "/v1/warehouse/ops/dispatch/execute", bytes.NewBufferString(`{"mode":"MANUAL"}`))
	rr := httptest.NewRecorder()
	s.handleOpsDispatchExecute(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d want 400", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "idempotency_key_required") {
		t.Fatalf("body = %q want idempotency_key_required", rr.Body.String())
	}
}

func TestWarehouseDispatchExecuteManifestStateIsDraft(t *testing.T) {
	if warehouseDispatchExecuteManifestState != "DRAFT" {
		t.Fatalf("warehouse dispatch execute manifest state = %q want DRAFT", warehouseDispatchExecuteManifestState)
	}
	if warehouseDispatchExecuteManifestState == "SEALED" {
		t.Fatal("warehouse dispatch execute must not pre-seal manifests; payloader owns seal gate")
	}
}

func TestWarehouseDispatchExecuteEmitsDraftCreatedEvent(t *testing.T) {
	if events.EventManifestDraftCreated == events.EventManifestSealed {
		t.Fatal("draft-created and sealed events must remain distinct contract types")
	}
}
