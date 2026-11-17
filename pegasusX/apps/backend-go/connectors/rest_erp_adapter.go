package connectors

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type restERPConfig struct {
	provider       string
	baseEnvName    string
	tokenEnvName   string
	healthPath     string
	healthEnvelope []string
	pullPaths      map[string]string
	pullEnvelopes  map[string][]string
	pushPaths      map[string]string
	defaultPushKey string
	mapRow         func(stream string, row map[string]any) (CanonicalRecord, bool)
	extractPushID  func(body []byte, prefix string, now time.Time) string
}

type restERPAdapter struct {
	cfg        restERPConfig
	httpClient *http.Client
	secrets    SecretResolver
	envBaseURL func() string
	envToken   func() string
	now        func() time.Time
}

func newRESTERPAdapter(cfg restERPConfig, secrets SecretResolver, httpClient *http.Client) *restERPAdapter {
	if secrets == nil {
		secrets = NewDefaultSecretResolver(nil)
	}
	baseEnv := cfg.baseEnvName
	tokEnv := cfg.tokenEnvName
	return &restERPAdapter{
		cfg:        cfg,
		httpClient: httpClient,
		secrets:    secrets,
		envBaseURL: func() string { return os.Getenv(baseEnv) },
		envToken:   func() string { return os.Getenv(tokEnv) },
		now:        func() time.Time { return time.Now().UTC() },
	}
}

func (a *restERPAdapter) Provider() string {
	return a.cfg.provider
}

func (a *restERPAdapter) resolveConfig(ctx context.Context, cred ExternalCredential) (string, string, error) {
	return resolveEndpointConfig(
		ctx,
		cred,
		a.secrets,
		a.envBaseURL,
		a.envToken,
		a.cfg.provider,
		a.cfg.baseEnvName,
		a.cfg.tokenEnvName,
		a != nil && a.httpClient != nil,
	)
}

func (a *restERPAdapter) currentNow() time.Time {
	if a != nil && a.now != nil {
		return a.now()
	}
	return time.Now().UTC()
}

func (a *restERPAdapter) Health(ctx context.Context, cred ExternalCredential) (HealthResult, error) {
	started := a.currentNow()
	res := HealthResult{
		Provider:     a.cfg.provider,
		AccountLabel: normalizeAccountLabel(cred.AccountLabel),
		CheckedAt:    started,
	}
	baseURL, token, err := a.resolveConfig(ctx, cred)
	if err != nil {
		res.Message = err.Error()
		return res, err
	}

	statusCode, body, callErr := doHTTPRequest(
		ctx, a.httpClient, a.cfg.provider, http.MethodGet, baseURL+a.cfg.healthPath,
		cred.AuthType, token, "", nil, nil,
	)
	res.LatencyMs = a.currentNow().Sub(started).Milliseconds()
	res.StatusCode = statusCode
	if callErr != nil {
		res.Message = scrubSecrets(callErr.Error(), token)
		return res, callErr
	}
	if _, parseErr := extractJSONEnvelopeRows(body, a.cfg.provider, a.cfg.healthEnvelope...); parseErr != nil {
		res.Message = parseErr.Error()
		return res, parseErr
	}
	res.Healthy = true
	res.Message = a.cfg.provider + " api reachable"
	return res, nil
}

func (a *restERPAdapter) Pull(ctx context.Context, cred ExternalCredential, req PullRequest) (PullResult, error) {
	baseURL, token, err := a.resolveConfig(ctx, cred)
	if err != nil {
		return PullResult{}, err
	}

	stream := strings.ToLower(strings.TrimSpace(req.Stream))
	if stream == "" {
		stream = StreamAll
	}
	targets := []string{stream}
	if stream == StreamAll {
		targets = []string{StreamProducts, StreamStocks, StreamShops, StreamOrders, StreamClients}
	}

	out := PullResult{
		Provider:     a.cfg.provider,
		AccountLabel: normalizeAccountLabel(cred.AccountLabel),
		Stream:       stream,
		Streams:      make(map[string][]map[string]any, len(targets)),
		StreamCounts: make(map[string]int64, len(targets)),
	}

	for _, st := range targets {
		endpoint, ok := a.cfg.pullPaths[st]
		if !ok {
			return PullResult{}, fmt.Errorf("%w: unsupported %s stream %q", ErrInvalidInput, a.cfg.provider, st)
		}
		if req.Limit > 0 {
			sep := "?"
			if strings.Contains(endpoint, "?") {
				sep = "&"
			}
			endpoint = fmt.Sprintf("%s%slimit=%d", endpoint, sep, req.Limit)
		}
		_, body, callErr := doHTTPRequest(
			ctx, a.httpClient, a.cfg.provider, http.MethodGet, baseURL+endpoint,
			cred.AuthType, token, "", nil, nil,
		)
		if callErr != nil {
			return PullResult{}, callErr
		}
		envKeys := a.cfg.pullEnvelopes[st]
		rows, parseErr := extractJSONEnvelopeRows(body, a.cfg.provider, envKeys...)
		if parseErr != nil {
			return PullResult{}, parseErr
		}
		out.Streams[st] = rows
		out.StreamCounts[st] = int64(len(rows))
		out.RawCount += int64(len(rows))

		mapped, mapErr := a.MapSchema(ctx, st, rows)
		if mapErr != nil {
			return PullResult{}, mapErr
		}
		out.Canonical = append(out.Canonical, mapped...)
	}
	out.RecordsCount = int64(len(out.Canonical))
	return out, nil
}

func (a *restERPAdapter) Push(ctx context.Context, cred ExternalCredential, req PushRequest) (PushResult, error) {
	baseURL, token, err := a.resolveConfig(ctx, cred)
	if err != nil {
		return PushResult{}, err
	}

	stream := strings.ToLower(strings.TrimSpace(req.Stream))
	if stream == "" || stream == StreamAll {
		stream = a.cfg.defaultPushKey
	}
	endpoint, ok := a.cfg.pushPaths[stream]
	if !ok {
		return PushResult{}, fmt.Errorf("%w: unsupported %s push stream %q", ErrInvalidInput, a.cfg.provider, req.Stream)
	}

	body, count, err := prepareJSONPushBody(a.cfg.provider, cred, req, stream)
	if err != nil {
		return PushResult{}, err
	}
	headers := map[string]string{}
	if aid := strings.TrimSpace(req.ApprovalID); aid != "" {
		headers["X-Atomos-Approval-Id"] = aid
	}
	_, respBody, callErr := doHTTPRequest(
		ctx, a.httpClient, a.cfg.provider, http.MethodPost, baseURL+endpoint,
		cred.AuthType, token, "application/json", body, headers,
	)
	if callErr != nil {
		return PushResult{}, callErr
	}

	now := a.currentNow()
	docID := ""
	if a.cfg.extractPushID != nil {
		docID = a.cfg.extractPushID(respBody, a.cfg.provider+"-push", now)
	} else {
		docID = extractExternalDocID(respBody, a.cfg.provider+"-push", now)
	}

	return PushResult{
		Provider:      a.cfg.provider,
		AccountLabel:  normalizeAccountLabel(cred.AccountLabel),
		Stream:        stream,
		RecordsPushed: count,
		ExternalDocID: docID,
		Status:        RunStatusCompleted,
		Message:       fmt.Sprintf("write-back pushed to %s", a.cfg.provider),
	}, nil
}

func (a *restERPAdapter) MapSchema(_ context.Context, stream string, rawRows []map[string]any) ([]CanonicalRecord, error) {
	st := strings.ToLower(strings.TrimSpace(stream))
	if st == "" {
		st = StreamAll
	}
	if _, ok := a.cfg.pullPaths[st]; !ok {
		return nil, fmt.Errorf("%w: unsupported %s stream %q", ErrInvalidInput, a.cfg.provider, stream)
	}
	out := make([]CanonicalRecord, 0, len(rawRows))
	for _, row := range rawRows {
		if row == nil {
			continue
		}
		rec, ok := a.cfg.mapRow(st, row)
		if !ok {
			continue
		}
		rec.ContentHash = canonicalContentHash(rec)
		out = append(out, rec)
	}
	return out, nil
}
