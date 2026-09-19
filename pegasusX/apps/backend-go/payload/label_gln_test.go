package payload

import (
	"context"
	"testing"
)

func TestLabelGLNs_NilClientGraceful(t *testing.T) {
	svc := &Service{}
	from, to := svc.labelGLNs(context.Background(), "man-1")
	if from != "" || to != "" {
		t.Fatalf("expected empty GLNs for nil spanner client, got from=%s, to=%s", from, to)
	}
}
