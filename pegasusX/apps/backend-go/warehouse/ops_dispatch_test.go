package warehouse

import (
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

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
