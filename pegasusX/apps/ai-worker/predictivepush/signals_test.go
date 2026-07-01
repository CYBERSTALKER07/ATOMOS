package predictivepush

import (
	"testing"
	"time"
)

func TestCompositeSignalProviderCollectEmpty(t *testing.T) {
	p := &CompositeSignalProvider{}
	out, err := p.Collect(nil, "sup-1", testDay())
	if err != nil {
		t.Fatal(err)
	}
	if out == nil {
		t.Fatal("expected non-nil slice")
	}
}

func testDay() time.Time {
	return time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
}
