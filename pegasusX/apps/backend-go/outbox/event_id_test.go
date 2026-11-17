package outbox

import (
	"strings"
	"testing"
)

func TestNewEventIDIsUUID(t *testing.T) {
	t.Parallel()
	id := newEventID()
	if len(id) != 36 || strings.Count(id, "-") != 4 {
		t.Fatalf("want UUID string, got %q", id)
	}
	id2 := newEventID()
	if id == id2 {
		t.Fatal("event ids must not collide across calls")
	}
}
