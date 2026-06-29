package supplier

import "testing"

func TestDispatchPartialCommitError(t *testing.T) {
	err := &dispatchPartialCommitError{
		CommittedRoutes: 50,
		FailedChunk:     2,
		TotalChunks:     3,
		TotalRoutes:     120,
		Cause:           errTestCause{},
	}
	msg := err.Error()
	if msg == "" {
		t.Fatal("expected non-empty error message")
	}
	if err.Unwrap() == nil {
		t.Fatal("expected unwrap cause")
	}
}

type errTestCause struct{}

func (errTestCause) Error() string { return "chunk failed" }
