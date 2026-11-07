package bootstrap

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMaxBytesMiddleware(t *testing.T) {
	limit := int64(1024) // 1KB limit for test
	mw := MaxBytesMiddleware(limit)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))

	// Case 1: Under limit (500 bytes) -> 200 OK
	smallBody := bytes.Repeat([]byte("a"), 500)
	req1 := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(smallBody))
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for small body, got %d", rec1.Code)
	}

	// Case 2: Exceeding limit (2048 bytes) -> 413 Request Entity Too Large
	largeBody := bytes.Repeat([]byte("a"), 2048)
	req2 := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(largeBody))
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 for oversized body, got %d", rec2.Code)
	}
}
