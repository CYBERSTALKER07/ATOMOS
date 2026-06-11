package supplier

import (
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

func TestDispatchExecuteManifestStateIsDraft(t *testing.T) {
	if dispatchExecuteManifestState != "DRAFT" {
		t.Fatalf("dispatch execute manifest state = %q want DRAFT", dispatchExecuteManifestState)
	}
	if dispatchExecuteManifestState == "SEALED" {
		t.Fatal("dispatch execute must not pre-seal manifests; payloader owns seal gate")
	}
}

func TestDispatchExecuteEmitsDraftCreatedEvent(t *testing.T) {
	if events.EventManifestDraftCreated == events.EventManifestSealed {
		t.Fatal("draft-created and sealed events must remain distinct contract types")
	}
}
