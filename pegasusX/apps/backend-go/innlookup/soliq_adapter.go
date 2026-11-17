package innlookup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// SoliqHTTPAdapter queries a configured Uzbekistan taxpayer registry endpoint
// (e.g. Soliq / GNK proxy) and fails closed (StatusUnavailable) on any transport
// error, timeout, missing configuration, or upstream 5xx response.
type SoliqHTTPAdapter struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
}

// SoliqAdapterFromEnv constructs a SoliqHTTPAdapter from INN_REGISTRY_BASE_URL,
// INN_REGISTRY_API_KEY, and INN_REGISTRY_TIMEOUT_MS environment variables.
// When unconfigured, lookups fail closed with ErrRegistryUnavailable.
func SoliqAdapterFromEnv() *SoliqHTTPAdapter {
	timeoutMs := 6000
	if raw := strings.TrimSpace(os.Getenv("INN_REGISTRY_TIMEOUT_MS")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			timeoutMs = parsed
		}
	}
	return &SoliqHTTPAdapter{
		BaseURL: strings.TrimRight(strings.TrimSpace(os.Getenv("INN_REGISTRY_BASE_URL")), "/"),
		APIKey:  strings.TrimSpace(os.Getenv("INN_REGISTRY_API_KEY")),
		Client:  &http.Client{Timeout: time.Duration(timeoutMs) * time.Millisecond},
	}
}

type soliqRegistryPayload struct {
	Found         *bool  `json:"found,omitempty"`
	Status        string `json:"status,omitempty"`
	LegalName     string `json:"legal_name,omitempty"`
	Name          string `json:"name,omitempty"`
	FullName      string `json:"full_name,omitempty"`
	Address       string `json:"address,omitempty"`
	LegalAddress  string `json:"legal_address,omitempty"`
	RegistryState string `json:"registry_state,omitempty"`
}

// Lookup calls the configured tax registry endpoint and maps its payload into LookupResult.
func (a *SoliqHTTPAdapter) Lookup(ctx context.Context, homeCell, normalizedINN string) (LookupResult, error) {
	if a == nil || strings.TrimSpace(a.BaseURL) == "" {
		return LookupResult{Status: StatusUnavailable, NormalizedINN: normalizedINN}, ErrRegistryUnavailable
	}
	client := a.Client
	if client == nil {
		client = &http.Client{Timeout: 6 * time.Second}
	}
	endpoint := fmt.Sprintf("%s?inn=%s&home_cell=%s",
		strings.TrimRight(a.BaseURL, "/"),
		url.QueryEscape(normalizedINN),
		url.QueryEscape(homeCell),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return LookupResult{Status: StatusUnavailable, NormalizedINN: normalizedINN}, ErrRegistryUnavailable
	}
	req.Header.Set("Accept", "application/json")
	if a.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+a.APIKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return LookupResult{Status: StatusUnavailable, NormalizedINN: normalizedINN}, fmt.Errorf("%w: %v", ErrRegistryUnavailable, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if err != nil {
		return LookupResult{Status: StatusUnavailable, NormalizedINN: normalizedINN}, fmt.Errorf("%w: read response: %v", ErrRegistryUnavailable, err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return LookupResult{
			Status:        StatusNotFound,
			NormalizedINN: normalizedINN,
			RawJSON:       validRawJSONOrNil(body),
		}, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return LookupResult{Status: StatusUnavailable, NormalizedINN: normalizedINN}, fmt.Errorf("%w: upstream status %d", ErrRegistryUnavailable, resp.StatusCode)
	}

	var parsed soliqRegistryPayload
	if err := json.Unmarshal(body, &parsed); err != nil {
		return LookupResult{Status: StatusUnavailable, NormalizedINN: normalizedINN}, fmt.Errorf("%w: invalid upstream json: %v", ErrRegistryUnavailable, err)
	}

	legalName := firstNonEmpty(parsed.LegalName, parsed.Name, parsed.FullName)
	address := firstNonEmpty(parsed.Address, parsed.LegalAddress)
	statusUpper := strings.ToUpper(strings.TrimSpace(parsed.Status))
	if (parsed.Found != nil && !*parsed.Found) || statusUpper == StatusNotFound || (parsed.Found == nil && legalName == "" && statusUpper == "") {
		return LookupResult{
			Status:        StatusNotFound,
			NormalizedINN: normalizedINN,
			RawJSON:       validRawJSONOrNil(body),
		}, nil
	}
	if statusUpper == StatusUnavailable {
		return LookupResult{Status: StatusUnavailable, NormalizedINN: normalizedINN}, ErrRegistryUnavailable
	}

	return LookupResult{
		Status:        StatusFound,
		NormalizedINN: normalizedINN,
		LegalName:     legalName,
		Address:       address,
		RegistryState: parsed.RegistryState,
		RawJSON:       validRawJSONOrNil(body),
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func validRawJSONOrNil(raw []byte) json.RawMessage {
	if len(raw) == 0 || !json.Valid(raw) {
		return nil
	}
	cp := make([]byte, len(raw))
	copy(cp, raw)
	return json.RawMessage(cp)
}
