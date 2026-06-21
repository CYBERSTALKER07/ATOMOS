package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func clientPostAuthorized(ctx context.Context, client *http.Client, url string, body []byte, authorization, idempotencyKey string) (int, []byte, http.Header, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authorization)
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	return resp.StatusCode, respBody, resp.Header, nil
}

func sliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func clientGet(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	status, body, _, err := clientDo(ctx, client, http.MethodGet, url, nil, "", "")
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GET %s status %d body %s", url, status, string(body))
	}
	return body, nil
}

func clientPost(ctx context.Context, client *http.Client, url string, body []byte, bearer, idempotencyKey string) (int, []byte, http.Header, error) {
	return clientDo(ctx, client, http.MethodPost, url, body, bearer, idempotencyKey)
}

func clientPostRetry(ctx context.Context, client *http.Client, url string, body []byte, bearer, idempotencyKey string) (int, []byte, http.Header, error) {
	return clientDoRetry(ctx, client, http.MethodPost, url, body, bearer, idempotencyKey)
}

func clientDoRetry(ctx context.Context, client *http.Client, method, url string, body []byte, bearerOrCookie, idempotencyKey string) (int, []byte, http.Header, error) {
	var lastStatus int
	var lastBody []byte
	var lastHdrs http.Header
	for attempt := 0; attempt < 12; attempt++ {
		status, respBody, hdrs, err := clientDo(ctx, client, method, url, body, bearerOrCookie, idempotencyKey)
		if err != nil {
			return 0, nil, nil, err
		}
		if status != http.StatusTooManyRequests {
			return status, respBody, hdrs, nil
		}
		lastStatus, lastBody, lastHdrs = status, respBody, hdrs
		wait := retryAfterSeconds(hdrs, 2+attempt)
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return lastStatus, lastBody, lastHdrs, ctx.Err()
		case <-timer.C:
		}
	}
	return lastStatus, lastBody, lastHdrs, nil
}

func retryAfterSeconds(hdrs http.Header, fallback int) time.Duration {
	if hdrs == nil {
		return time.Duration(fallback) * time.Second
	}
	raw := strings.TrimSpace(hdrs.Get("Retry-After"))
	if raw == "" {
		return time.Duration(fallback) * time.Second
	}
	if secs, err := strconv.Atoi(raw); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	return time.Duration(fallback) * time.Second
}

func clientDo(ctx context.Context, client *http.Client, method, url string, body []byte, bearerOrCookie, idempotencyKey string) (int, []byte, http.Header, error) {
	contentType := "application/json"
	if body == nil {
		contentType = ""
	}
	return clientDoContentType(ctx, client, method, url, body, contentType, bearerOrCookie, idempotencyKey)
}

func clientDoContentType(ctx context.Context, client *http.Client, method, url string, body []byte, contentType, bearerOrCookie, idempotencyKey string) (int, []byte, http.Header, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return 0, nil, nil, err
	}
	if body != nil && strings.TrimSpace(contentType) != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}
	if strings.HasPrefix(bearerOrCookie, auth.CookieName+"=") || strings.Contains(bearerOrCookie, "=") {
		req.Header.Set("Cookie", bearerOrCookie)
	} else if bearerOrCookie != "" {
		req.Header.Set("Authorization", "Bearer "+bearerOrCookie)
	}
	if secret := strings.TrimSpace(envOr("LOAD_BOOTSTRAP_SECRET", "")); secret != "" {
		req.Header.Set("X-PegasusX-Load-Bootstrap", secret)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	return resp.StatusCode, respBody, resp.Header, nil
}

func sessionCookie(hdrs http.Header) string {
	for _, c := range hdrs["Set-Cookie"] {
		if strings.HasPrefix(c, auth.CookieName+"=") {
			part := strings.SplitN(c, ";", 2)[0]
			return part
		}
	}
	return ""
}

func supplierIDFromJWT(cookieHeader, secret string) string {
	token := strings.TrimPrefix(cookieHeader, auth.CookieName+"=")
	claims, err := auth.Parse(token, secret)
	if err != nil {
		return ""
	}
	return claims.SupplierID
}
