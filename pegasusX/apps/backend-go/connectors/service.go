package connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

// Service orchestrates M1.9 / Wave 4A external connector adapters, credential persistence,
// health/pull/approval-gated push operations, ShadowCompare diffing, outbox emission, audit, and ontology lineage.
type Service struct {
	mu              sync.RWMutex
	repo            Repository
	secrets         SecretResolver
	httpClient      *http.Client
	connectors      map[string]Connector
	audit           AuditRecorder
	approvalChecker PushApprovalChecker
	ontologyHook    OntologyLineageHook
	now             func() time.Time
}

// NewService constructs a Spanner-backed connector spine service with all 5 adapters registered.
func NewService(client *spanner.Client) *Service {
	var repo Repository
	if client != nil {
		repo = NewSpannerRepository(client)
	}
	return NewServiceWithRepository(repo)
}

// NewServiceWithRepository constructs a connector spine service backed by a custom Repository.
func NewServiceWithRepository(repo Repository) *Service {
	secrets := NewDefaultSecretResolver(nil)
	s := &Service{
		repo:       repo,
		secrets:    secrets,
		connectors: make(map[string]Connector),
		now:        func() time.Time { return time.Now().UTC() },
	}
	s.mu.Lock()
	s.registerBuiltinConnectorsLocked(secrets, nil)
	s.mu.Unlock()
	return s
}

func (s *Service) registerBuiltinConnectorsLocked(sr SecretResolver, client *http.Client) {
	if s.connectors == nil {
		s.connectors = make(map[string]Connector)
	}
	s.connectors[ProviderBillz] = NewBillzConnector(sr, client)
	s.connectors[ProviderOnec] = NewOnecConnector(sr, client)
	s.connectors[ProviderMoySklad] = NewMoySkladConnector(sr, client)
	s.connectors[ProviderSmartup] = NewSmartupConnector(sr, client)
	s.connectors[ProviderIiko] = NewIikoConnector(sr, client)
}

// Repo exposes the underlying Repository.
func (s *Service) Repo() Repository {
	if s == nil {
		return nil
	}
	return s.repo
}

// RegisterConnector registers or replaces a Connector adapter by Provider().
func (s *Service) RegisterConnector(c Connector) {
	if s == nil || c == nil {
		return
	}
	prov := normalizeProvider(c.Provider())
	if prov == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.connectors == nil {
		s.connectors = make(map[string]Connector)
	}
	s.connectors[prov] = c
}

// SetSecretResolver sets a custom SecretResolver (e.g. GCP Secret Manager hook) and rebinds built-in adapters.
func (s *Service) SetSecretResolver(sr SecretResolver) {
	if s == nil || sr == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.secrets = sr
	s.registerBuiltinConnectorsLocked(sr, s.httpClient)
}

// SetHTTPClient sets a custom http.Client for built-in HTTP adapters (useful for timeout/retry/test control).
func (s *Service) SetHTTPClient(client *http.Client) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.httpClient = client
	s.registerBuiltinConnectorsLocked(s.secrets, client)
}

// SetApprovalChecker attaches the approval verification hook required for 2-way Push write-backs.
func (s *Service) SetApprovalChecker(checker PushApprovalChecker) {
	if s != nil {
		s.mu.Lock()
		s.approvalChecker = checker
		s.mu.Unlock()
	}
}

// SetAuditRecorder attaches an immutable AuditEvents recorder.
func (s *Service) SetAuditRecorder(rec AuditRecorder) {
	if s != nil {
		s.mu.Lock()
		s.audit = rec
		s.mu.Unlock()
	}
}

// SetOntologyLineageHook attaches an optional ontology source_system lineage callback.
func (s *Service) SetOntologyLineageHook(hook OntologyLineageHook) {
	if s != nil {
		s.mu.Lock()
		s.ontologyHook = hook
		s.mu.Unlock()
	}
}

func (s *Service) currentNow() time.Time {
	if s != nil && s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}

func (s *Service) getConnector(provider string) (Connector, error) {
	prov := normalizeProvider(provider)
	if prov == "" {
		return nil, fmt.Errorf("%w: provider is required", ErrInvalidInput)
	}
	s.mu.RLock()
	c, ok := s.connectors[prov]
	s.mu.RUnlock()
	if !ok || c == nil {
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedProvider, prov)
	}
	return c, nil
}

// ListAdapters returns metadata descriptors for all registered connectors.
func (s *Service) ListAdapters() []AdapterDescriptor {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]AdapterDescriptor, 0, len(s.connectors))
	for prov := range s.connectors {
		switch prov {
		case ProviderBillz:
			out = append(out, AdapterDescriptor{
				Provider:     ProviderBillz,
				DisplayName:  "BILLZ POS & Inventory",
				Capabilities: []string{"health", "pull", "push", "shadow_compare", "products", "stocks", "shops", "orders", "clients", "goods_receipt", "inventory_adjustment"},
				ReadOnly:     false,
			})
		case ProviderOnec:
			out = append(out, AdapterDescriptor{
				Provider:     ProviderOnec,
				DisplayName:  "1C:Enterprise (CommerceML & OData Multi-Account)",
				Capabilities: []string{"health", "pull", "push", "shadow_compare", "commerceml", "odata", "journals", "multi_account"},
				ReadOnly:     false,
			})
		case ProviderMoySklad:
			out = append(out, AdapterDescriptor{
				Provider:     ProviderMoySklad,
				DisplayName:  "MoySklad Cloud ERP",
				Capabilities: []string{"health", "pull", "push", "shadow_compare", "products", "stocks", "shops", "orders", "clients", "goods_receipt", "inventory_adjustment"},
				ReadOnly:     false,
			})
		case ProviderSmartup:
			out = append(out, AdapterDescriptor{
				Provider:     ProviderSmartup,
				DisplayName:  "Smartup Distribution ERP",
				Capabilities: []string{"health", "pull", "push", "shadow_compare", "products", "stocks", "shops", "orders", "clients", "goods_receipt", "inventory_adjustment", "journals"},
				ReadOnly:     false,
			})
		case ProviderIiko:
			out = append(out, AdapterDescriptor{
				Provider:     ProviderIiko,
				DisplayName:  "iiko HORECA POS & Inventory",
				Capabilities: []string{"health", "pull", "push", "shadow_compare", "products", "stocks", "shops", "orders", "clients", "goods_receipt", "inventory_adjustment"},
				ReadOnly:     false,
			})
		default:
			out = append(out, AdapterDescriptor{
				Provider:     prov,
				DisplayName:  strings.ToUpper(prov),
				Capabilities: []string{"health", "pull", "push", "shadow_compare"},
				ReadOnly:     false,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Provider < out[j].Provider })
	return out
}

// Overview aggregates registered adapters, masked credentials, and latest sync runs for a tenant.
type Overview struct {
	TenantID    string                       `json:"tenant_id"`
	Adapters    []AdapterDescriptor          `json:"adapters"`
	Credentials []ConnectorCredentialSummary `json:"credentials"`
	LatestRuns  []ConnectorSyncRun           `json:"latest_runs"`
}

func (s *Service) GetOverview(ctx context.Context, tenantID string) (Overview, error) {
	if s == nil || s.repo == nil {
		return Overview{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	creds, err := s.repo.ListCredentials(ctx, tid, "")
	if err != nil {
		return Overview{}, err
	}
	runs, err := s.repo.ListLatestRunsByProvider(ctx, tid)
	if err != nil {
		return Overview{}, err
	}
	summaries := make([]ConnectorCredentialSummary, 0, len(creds))
	for _, c := range creds {
		summaries = append(summaries, c.Summary())
	}
	if runs == nil {
		runs = []ConnectorSyncRun{}
	}
	return Overview{
		TenantID:    tid,
		Adapters:    s.ListAdapters(),
		Credentials: summaries,
		LatestRuns:  runs,
	}, nil
}

// UpsertCredential validates and stores a tenant connector credential (with SecretRef handle only).
func (s *Service) UpsertCredential(ctx context.Context, in UpsertCredentialInput) (ConnectorCredentialSummary, error) {
	if s == nil || s.repo == nil {
		return ConnectorCredentialSummary{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(in.TenantID)
	prov := normalizeProvider(in.Provider)
	if tid == "" || prov == "" {
		return ConnectorCredentialSummary{}, fmt.Errorf("%w: tenant_id and provider are required", ErrInvalidInput)
	}
	if _, err := s.getConnector(prov); err != nil {
		return ConnectorCredentialSummary{}, err
	}
	secretRef := strings.TrimSpace(in.SecretRef)
	if err := ValidateSecretRefForTenant(tid, secretRef); err != nil {
		return ConnectorCredentialSummary{}, err
	}

	metaJSON := strings.TrimSpace(in.MetadataJSON)
	if metaJSON != "" {
		var generic any
		if err := decodeJSONWithNumbers([]byte(metaJSON), &generic); err != nil {
			return ConnectorCredentialSummary{}, fmt.Errorf("%w: invalid metadata_json: %v", ErrInvalidInput, err)
		}
		if err := validateNoFloats(generic); err != nil {
			return ConnectorCredentialSummary{}, err
		}
	} else if len(in.Metadata) > 0 {
		if err := validateNoFloats(in.Metadata); err != nil {
			return ConnectorCredentialSummary{}, err
		}
		raw, err := json.Marshal(in.Metadata)
		if err != nil {
			return ConnectorCredentialSummary{}, fmt.Errorf("%w: marshal metadata: %v", ErrInvalidInput, err)
		}
		metaJSON = string(raw)
	}

	status := strings.ToUpper(strings.TrimSpace(in.Status))
	if status == "" {
		status = CredStatusActive
	}
	if status != CredStatusActive && status != CredStatusDisabled {
		return ConnectorCredentialSummary{}, fmt.Errorf("%w: invalid credential status %q", ErrInvalidInput, status)
	}

	if strings.TrimSpace(in.BaseURL) != "" {
		s.mu.RLock()
		customClient := s.httpClient != nil
		s.mu.RUnlock()
		if err := ValidateConnectorBaseURL(in.BaseURL, customClient || allowPrivateConnectorHostsFromEnv()); err != nil {
			return ConnectorCredentialSummary{}, err
		}
	}

	cred := ExternalCredential{
		TenantID:     tid,
		CredentialID: strings.TrimSpace(in.CredentialID),
		Provider:     prov,
		AccountLabel: normalizeAccountLabel(in.AccountLabel),
		AuthType:     normalizeAuthType(in.AuthType),
		SecretRef:    secretRef,
		BaseURL:      strings.TrimSpace(in.BaseURL),
		MetadataJSON: metaJSON,
		Metadata:     in.Metadata,
		Status:       status,
		CreatedBy:    firstNonEmpty(strings.TrimSpace(in.CreatedBy), "system"),
	}
	saved, err := s.repo.UpsertCredential(ctx, cred)
	if err != nil {
		return ConnectorCredentialSummary{}, err
	}
	summary := saved.Summary()
	afterRaw, _ := json.Marshal(summary)
	s.recordAudit(ctx, AuditRecordInput{
		TenantID:   tid,
		ActorID:    firstNonEmpty(in.CreatedBy, "system"),
		ActorType:  firstNonEmpty(in.ActorType, auth.ActorTypeHuman),
		Role:       in.ActorRole,
		Action:     "connector.credential.upsert",
		ObjectType: "ExternalCredential",
		ObjectID:   summary.CredentialID,
		AfterJSON:  string(afterRaw),
		TraceID:    in.TraceID,
		IP:         in.IP,
		DeviceID:   in.DeviceID,
	})
	return summary, nil
}

func (s *Service) resolveCredentialForTrigger(ctx context.Context, tenantID, provider, accountLabel, credentialID string) (ExternalCredential, error) {
	prov := normalizeProvider(provider)
	cred, found, err := s.repo.GetCredential(ctx, tenantID, prov, accountLabel, credentialID)
	if err != nil {
		return ExternalCredential{}, err
	}
	if !found {
		if strings.TrimSpace(credentialID) != "" {
			return ExternalCredential{}, ErrCredentialNotFound
		}
		return ExternalCredential{
			TenantID:     tenantID,
			Provider:     prov,
			AccountLabel: normalizeAccountLabel(accountLabel),
			AuthType:     AuthTypeBearer,
			Status:       CredStatusActive,
		}, nil
	}
	if strings.EqualFold(cred.Status, CredStatusDisabled) {
		return ExternalCredential{}, fmt.Errorf("%w: credential %s is disabled", ErrConnectorNotConfigured, cred.CredentialID)
	}
	return cred, nil
}

// CheckHealth pings the external connector, persists a HEALTH ConnectorSyncRun + OutboxEvents, and audits.
func (s *Service) CheckHealth(ctx context.Context, in TriggerConnectorInput) (HealthResult, error) {
	if s == nil || s.repo == nil {
		return HealthResult{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(in.TenantID)
	prov := normalizeProvider(in.Provider)
	conn, err := s.getConnector(prov)
	if err != nil {
		return HealthResult{}, err
	}
	cred, err := s.resolveCredentialForTrigger(ctx, tid, prov, in.AccountLabel, in.CredentialID)
	if err != nil {
		return HealthResult{}, err
	}

	started := s.currentNow()
	res, callErr := conn.Health(ctx, cred)
	completed := s.currentNow()
	runID := "crun_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:24]
	status := RunStatusCompleted
	errMsg := ""
	evType := events.EventConnectorSyncCompleted
	if callErr != nil {
		status = RunStatusFailed
		errMsg = callErr.Error()
		evType = events.EventConnectorSyncFailed
	}

	run := ConnectorSyncRun{
		TenantID:     tid,
		RunID:        runID,
		Provider:     prov,
		AccountLabel: normalizeAccountLabel(cred.AccountLabel),
		Direction:    DirectionHealth,
		StreamName:   "health",
		Status:       status,
		ErrorMessage: errMsg,
		StartedAt:    started,
		CompletedAt:  &completed,
		CreatedAt:    completed,
	}
	ev := events.ConnectorSyncEvent{
		BaseEvent: events.BaseEvent{
			Type:      evType,
			Version:   1,
			Timestamp: completed.Format(time.RFC3339Nano),
		},
		TenantID:     tid,
		RunID:        runID,
		Provider:     prov,
		AccountLabel: run.AccountLabel,
		Direction:    DirectionHealth,
		StreamName:   "health",
		Status:       status,
		ErrorMessage: errMsg,
		TriggeredBy:  in.ActorID,
		StartedAt:    started.Format(time.RFC3339Nano),
		CompletedAt:  completed.Format(time.RFC3339Nano),
		OccurredAt:   completed.Format(time.RFC3339Nano),
	}
	if saveErr := s.repo.SaveSyncRunAndMaps(ctx, run, nil, []OutboxEmission{{
		AggregateType: events.AggregateConnectorSyncRun,
		AggregateID:   runID,
		Topic:         events.TopicConnectors,
		Event:         ev,
	}}); saveErr != nil && callErr == nil {
		return res, saveErr
	}

	s.recordAudit(ctx, AuditRecordInput{
		TenantID:   tid,
		ActorID:    firstNonEmpty(in.ActorID, "system"),
		ActorType:  firstNonEmpty(in.ActorType, auth.ActorTypeHuman),
		Role:       in.ActorRole,
		Action:     "connector.health",
		ObjectType: "ConnectorSyncRun",
		ObjectID:   runID,
		ReasonCode: status,
		TraceID:    in.TraceID,
		IP:         in.IP,
		DeviceID:   in.DeviceID,
	})
	res.Run = &run
	return res, callErr
}

// Pull fetches external streams via Connector.Pull, persists a PULL ConnectorSyncRun, and emits OutboxEvents.
func (s *Service) Pull(ctx context.Context, in TriggerConnectorInput) (PullResponse, error) {
	if s == nil || s.repo == nil {
		return PullResponse{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(in.TenantID)
	prov := normalizeProvider(in.Provider)
	conn, err := s.getConnector(prov)
	if err != nil {
		return PullResponse{}, err
	}
	if idem := strings.TrimSpace(in.IdempotencyKey); idem != "" {
		if existing, found, fErr := s.repo.FindRunByIdempotency(ctx, tid, idem); fErr != nil {
			return PullResponse{}, fErr
		} else if found {
			return PullResponse{
				Run:          existing,
				RecordsCount: existing.RecordsProcessed,
			}, nil
		}
	}

	cred, err := s.resolveCredentialForTrigger(ctx, tid, prov, in.AccountLabel, in.CredentialID)
	if err != nil {
		return PullResponse{}, err
	}

	stream := firstNonEmpty(strings.ToLower(strings.TrimSpace(in.Stream)), StreamAll)
	started := s.currentNow()
	pullRes, callErr := conn.Pull(ctx, cred, PullRequest{
		Stream:     stream,
		RawPayload: in.RawPayload,
	})
	completed := s.currentNow()
	runID := "crun_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:24]

	if callErr != nil {
		s.persistFailedRun(ctx, tid, runID, prov, cred.AccountLabel, DirectionPull, stream, in.IdempotencyKey, callErr.Error(), started, completed, in)
		return PullResponse{}, callErr
	}

	run := ConnectorSyncRun{
		TenantID:         tid,
		RunID:            runID,
		Provider:         prov,
		AccountLabel:     normalizeAccountLabel(cred.AccountLabel),
		Direction:        DirectionPull,
		StreamName:       stream,
		Status:           RunStatusCompleted,
		RecordsProcessed: pullRes.RecordsCount,
		IdempotencyKey:   strings.TrimSpace(in.IdempotencyKey),
		StartedAt:        started,
		CompletedAt:      &completed,
		CreatedAt:        completed,
	}
	ev := events.ConnectorSyncEvent{
		BaseEvent: events.BaseEvent{
			Type:      events.EventConnectorSyncCompleted,
			Version:   1,
			Timestamp: completed.Format(time.RFC3339Nano),
		},
		TenantID:         tid,
		RunID:            runID,
		Provider:         prov,
		AccountLabel:     run.AccountLabel,
		Direction:        DirectionPull,
		StreamName:       stream,
		Status:           RunStatusCompleted,
		RecordsProcessed: pullRes.RecordsCount,
		TriggeredBy:      in.ActorID,
		StartedAt:        started.Format(time.RFC3339Nano),
		CompletedAt:      completed.Format(time.RFC3339Nano),
		OccurredAt:       completed.Format(time.RFC3339Nano),
	}
	if err := s.repo.SaveSyncRunAndMaps(ctx, run, nil, []OutboxEmission{{
		AggregateType: events.AggregateConnectorSyncRun,
		AggregateID:   runID,
		Topic:         events.TopicConnectors,
		Event:         ev,
	}}); err != nil {
		return PullResponse{}, err
	}
	afterRaw, _ := json.Marshal(map[string]any{
		"run_id":        runID,
		"provider":      prov,
		"records_count": pullRes.RecordsCount,
		"stream_counts": pullRes.StreamCounts,
	})
	s.recordAudit(ctx, AuditRecordInput{
		TenantID:   tid,
		ActorID:    firstNonEmpty(in.ActorID, "system"),
		ActorType:  firstNonEmpty(in.ActorType, auth.ActorTypeHuman),
		Role:       in.ActorRole,
		Action:     "connector.pull",
		ObjectType: "ConnectorSyncRun",
		ObjectID:   runID,
		AfterJSON:  string(afterRaw),
		TraceID:    in.TraceID,
		IP:         in.IP,
		DeviceID:   in.DeviceID,
	})
	return PullResponse{
		Run:          run,
		RecordsCount: pullRes.RecordsCount,
		StreamCounts: pullRes.StreamCounts,
	}, nil
}

// Push executes an approval-gated 2-way write-back to an external ERP/POS connector,
// persisting a PUSH ConnectorSyncRun, emitting connector.push.completed on TopicConnectors,
// and recording an immutable AuditEvents row.
func (s *Service) Push(ctx context.Context, in TriggerConnectorInput) (PushResponse, error) {
	if s == nil || s.repo == nil {
		return PushResponse{}, ErrRepoUnavailable
	}
	if claims, ok := auth.FromContext(ctx); ok {
		if auth.IsReadOnlyAuditor(claims) {
			return PushResponse{}, ErrAuditorReadOnly
		}
		if auth.IsAgentActor(claims) {
			return PushResponse{}, ErrAgentForbidden
		}
	}
	if strings.EqualFold(strings.TrimSpace(in.ActorRole), auth.SupplierRoleAuditor) {
		return PushResponse{}, ErrAuditorReadOnly
	}
	if strings.EqualFold(strings.TrimSpace(in.ActorRole), auth.SupplierRoleAgent) ||
		strings.EqualFold(strings.TrimSpace(in.ActorType), auth.ActorTypeAgent) {
		return PushResponse{}, ErrAgentForbidden
	}

	tid := strings.TrimSpace(in.TenantID)
	prov := normalizeProvider(in.Provider)
	conn, err := s.getConnector(prov)
	if err != nil {
		return PushResponse{}, err
	}

	approvalID := strings.TrimSpace(in.ApprovalID)
	if approvalID == "" {
		return PushResponse{}, fmt.Errorf("%w: approval_id is required for connector push", ErrApprovalRequired)
	}

	s.mu.RLock()
	checker := s.approvalChecker
	s.mu.RUnlock()
	if checker == nil {
		return PushResponse{}, fmt.Errorf("%w: push approval checker is not wired", ErrApprovalRequired)
	}

	acctLabel := normalizeAccountLabel(in.AccountLabel)
	stream := firstNonEmpty(strings.ToLower(strings.TrimSpace(in.Stream)), defaultPushStreamForProvider(prov))
	approvalPayload := in.ApprovalPayload
	if approvalPayload == nil {
		approvalPayload = BuildPushApprovalPayloadFromRecords(prov, acctLabel, stream, in.Records, string(in.RawPayload))
	}
	if err := checker.VerifyPushApproval(ctx, PushApprovalInput{
		TenantID:   tid,
		Provider:   prov,
		ApprovalID: approvalID,
		Stream:     stream,
		Payload:    approvalPayload,
		ActorID:    in.ActorID,
		ActorType:  in.ActorType,
		ActorRole:  in.ActorRole,
	}); err != nil {
		return PushResponse{}, err
	}

	if idem := strings.TrimSpace(in.IdempotencyKey); idem != "" {
		if existing, found, fErr := s.repo.FindRunByIdempotency(ctx, tid, idem); fErr != nil {
			return PushResponse{}, fErr
		} else if found {
			return PushResponse{
				Run: existing,
				PushResult: PushResult{
					Provider:      existing.Provider,
					AccountLabel:  existing.AccountLabel,
					Stream:        existing.StreamName,
					RecordsPushed: existing.RecordsProcessed,
					Status:        existing.Status,
					Message:       "idempotent replay",
				},
				ApprovalID: approvalID,
			}, nil
		}
	}

	cred, err := s.resolveCredentialForTrigger(ctx, tid, prov, in.AccountLabel, in.CredentialID)
	if err != nil {
		return PushResponse{}, err
	}

	started := s.currentNow()
	pushRes, callErr := conn.Push(ctx, cred, PushRequest{
		Stream:     stream,
		ApprovalID: approvalID,
		Records:    in.Records,
		RawPayload: in.RawPayload,
	})
	completed := s.currentNow()
	runID := "crun_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:24]
	if callErr != nil {
		s.persistFailedRun(ctx, tid, runID, prov, cred.AccountLabel, DirectionPush, stream, in.IdempotencyKey, callErr.Error(), started, completed, in)
		return PushResponse{}, callErr
	}

	run := ConnectorSyncRun{
		TenantID:         tid,
		RunID:            runID,
		Provider:         prov,
		AccountLabel:     normalizeAccountLabel(cred.AccountLabel),
		Direction:        DirectionPush,
		StreamName:       firstNonEmpty(pushRes.Stream, stream),
		Status:           RunStatusCompleted,
		RecordsProcessed: pushRes.RecordsPushed,
		IdempotencyKey:   strings.TrimSpace(in.IdempotencyKey),
		StartedAt:        started,
		CompletedAt:      &completed,
		CreatedAt:        completed,
	}

	ts := completed.Format(time.RFC3339Nano)
	ev := events.ConnectorPushCompletedEvent{
		BaseEvent: events.BaseEvent{
			Type:      events.EventConnectorPushCompleted,
			Version:   1,
			Timestamp: ts,
		},
		TenantID:         tid,
		RunID:            runID,
		Provider:         prov,
		AccountLabel:     run.AccountLabel,
		Direction:        DirectionPush,
		StreamName:       run.StreamName,
		Status:           RunStatusCompleted,
		RecordsProcessed: pushRes.RecordsPushed,
		ApprovalID:       approvalID,
		ExternalRef:      pushRes.ExternalDocID,
		TriggeredBy:      in.ActorID,
		CompletedAt:      ts,
		OccurredAt:       ts,
	}
	if err := s.repo.SaveSyncRunAndMaps(ctx, run, nil, []OutboxEmission{{
		AggregateType: events.AggregateConnectorSyncRun,
		AggregateID:   runID,
		Topic:         events.TopicConnectors,
		Event:         ev,
	}}); err != nil {
		return PushResponse{}, err
	}

	afterRaw, _ := json.Marshal(map[string]any{
		"run_id":          runID,
		"provider":        prov,
		"account_label":   run.AccountLabel,
		"stream":          run.StreamName,
		"approval_id":     approvalID,
		"external_doc_id": pushRes.ExternalDocID,
		"records_pushed":  pushRes.RecordsPushed,
	})
	s.recordAudit(ctx, AuditRecordInput{
		TenantID:   tid,
		ActorID:    firstNonEmpty(in.ActorID, "system"),
		ActorType:  firstNonEmpty(in.ActorType, auth.ActorTypeHuman),
		Role:       in.ActorRole,
		Action:     "connector.push",
		ObjectType: "ConnectorSyncRun",
		ObjectID:   runID,
		AfterJSON:  string(afterRaw),
		ReasonCode: approvalID,
		TraceID:    in.TraceID,
		IP:         in.IP,
		DeviceID:   in.DeviceID,
	})

	return PushResponse{
		Run:        run,
		PushResult: pushRes,
		ApprovalID: approvalID,
	}, nil
}

func defaultPushStreamForProvider(provider string) string {
	switch normalizeProvider(provider) {
	case ProviderBillz, ProviderMoySklad:
		return "inventory_adjustment"
	case ProviderOnec:
		return "journals"
	case ProviderSmartup, ProviderIiko:
		return "goods_receipt"
	default:
		return "inventory_adjustment"
	}
}

// BuildPushApprovalPayloadFromRecords constructs the canonical zero-float approval payload map for a connector push.
func BuildPushApprovalPayloadFromRecords(provider, accountLabel, stream string, records []CanonicalRecord, rawPayload string) map[string]any {
	out := map[string]any{
		"provider":      normalizeProvider(provider),
		"account_label": normalizeAccountLabel(accountLabel),
		"stream":        firstNonEmpty(strings.ToLower(strings.TrimSpace(stream)), defaultPushStreamForProvider(provider)),
	}
	if len(records) > 0 {
		out["records"] = records
	}
	if strings.TrimSpace(rawPayload) != "" {
		out["raw_payload"] = rawPayload
	}
	return out
}

// ShadowCompare pulls external snapshot, diffs against Spanner, persists run & maps, emits events, audits, and notifies ontology.
func (s *Service) ShadowCompare(ctx context.Context, in TriggerConnectorInput) (ShadowCompareResult, error) {
	if s == nil || s.repo == nil {
		return ShadowCompareResult{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(in.TenantID)
	prov := normalizeProvider(in.Provider)
	conn, err := s.getConnector(prov)
	if err != nil {
		return ShadowCompareResult{}, err
	}
	if idem := strings.TrimSpace(in.IdempotencyKey); idem != "" {
		if existing, found, fErr := s.repo.FindRunByIdempotency(ctx, tid, idem); fErr != nil {
			return ShadowCompareResult{}, fErr
		} else if found {
			var sum ShadowCompareSummary
			if existing.Summary != nil {
				sum = *existing.Summary
			}
			mismatches := existing.Mismatches
			if mismatches == nil {
				mismatches = []ShadowCompareMismatch{}
			}
			return ShadowCompareResult{
				Run:        existing,
				Summary:    sum,
				Mismatches: mismatches,
			}, nil
		}
	}

	cred, err := s.resolveCredentialForTrigger(ctx, tid, prov, in.AccountLabel, in.CredentialID)
	if err != nil {
		return ShadowCompareResult{}, err
	}

	started := s.currentNow()
	pullRes, callErr := conn.Pull(ctx, cred, PullRequest{
		Stream:     StreamAll,
		RawPayload: in.RawPayload,
	})
	if callErr != nil {
		completed := s.currentNow()
		runID := "crun_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:24]
		s.persistFailedRun(ctx, tid, runID, prov, cred.AccountLabel, DirectionShadowCompare, StreamAll, in.IdempotencyKey, callErr.Error(), started, completed, in)
		return ShadowCompareResult{}, callErr
	}

	snap, snapErr := s.repo.LoadInternalSnapshot(ctx, tid, prov)
	if snapErr != nil {
		completed := s.currentNow()
		runID := "crun_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:24]
		s.persistFailedRun(ctx, tid, runID, prov, cred.AccountLabel, DirectionShadowCompare, StreamAll, in.IdempotencyKey, snapErr.Error(), started, completed, in)
		return ShadowCompareResult{}, snapErr
	}

	completed := s.currentNow()
	runID := "crun_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:24]
	summary, mismatches, entityMaps := ComputeShadowCompare(tid, prov, cred.AccountLabel, runID, completed, pullRes.Canonical, snap)

	envBytes, err := json.Marshal(ShadowCompareEnvelope{
		Summary:    summary,
		Mismatches: mismatches,
	})
	if err != nil {
		return ShadowCompareResult{}, fmt.Errorf("marshal shadow compare summary: %w", err)
	}
	runStatus := RunStatusCompleted
	if summary.TotalMismatches > 0 {
		runStatus = RunStatusMismatchFound
	}

	run := ConnectorSyncRun{
		TenantID:            tid,
		RunID:               runID,
		Provider:            prov,
		AccountLabel:        normalizeAccountLabel(cred.AccountLabel),
		Direction:           DirectionShadowCompare,
		StreamName:          StreamAll,
		Status:              runStatus,
		RecordsProcessed:    pullRes.RecordsCount,
		MismatchCount:       summary.TotalMismatches,
		MismatchSummaryJSON: string(envBytes),
		Summary:             &summary,
		Mismatches:          mismatches,
		IdempotencyKey:      strings.TrimSpace(in.IdempotencyKey),
		StartedAt:           started,
		CompletedAt:         &completed,
		CreatedAt:           completed,
	}

	ts := completed.Format(time.RFC3339Nano)
	emissions := []OutboxEmission{{
		AggregateType: events.AggregateConnectorSyncRun,
		AggregateID:   runID,
		Topic:         events.TopicConnectors,
		Event: events.ConnectorSyncEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventConnectorSyncCompleted,
				Version:   1,
				Timestamp: ts,
			},
			TenantID:             tid,
			RunID:                runID,
			Provider:             prov,
			AccountLabel:         run.AccountLabel,
			Direction:            DirectionShadowCompare,
			StreamName:           StreamAll,
			Status:               runStatus,
			RecordsProcessed:     pullRes.RecordsCount,
			MismatchCount:        summary.TotalMismatches,
			MissingExternalCount: summary.MissingExternal,
			MissingInternalCount: summary.MissingInternal,
			StockDeltaCount:      summary.StockDeltaCount,
			BalanceDeltaCount:    summary.BalanceDeltaCount,
			TriggeredBy:          in.ActorID,
			StartedAt:            started.Format(time.RFC3339Nano),
			CompletedAt:          ts,
			OccurredAt:           ts,
		},
	}}
	if summary.TotalMismatches > 0 {
		emissions = append(emissions, OutboxEmission{
			AggregateType: events.AggregateConnectorSyncRun,
			AggregateID:   runID,
			Topic:         events.TopicConnectors,
			Event: events.ConnectorShadowMismatchEvent{
				BaseEvent: events.BaseEvent{
					Type:      events.EventConnectorShadowMismatch,
					Version:   1,
					Timestamp: ts,
				},
				TenantID:             tid,
				RunID:                runID,
				Provider:             prov,
				AccountLabel:         run.AccountLabel,
				Direction:            DirectionShadowCompare,
				StreamName:           StreamAll,
				Status:               runStatus,
				RecordsProcessed:     pullRes.RecordsCount,
				MismatchCount:        summary.TotalMismatches,
				MissingExternalCount: summary.MissingExternal,
				MissingInternalCount: summary.MissingInternal,
				StockDeltaCount:      summary.StockDeltaCount,
				BalanceDeltaCount:    summary.BalanceDeltaCount,
				BalanceDeltaMinor:    summary.NetBalanceDeltaMinor,
				SummaryJSON:          string(envBytes),
				OccurredAt:           ts,
			},
		})
	}

	if err := s.repo.SaveSyncRunAndMaps(ctx, run, entityMaps, emissions); err != nil {
		return ShadowCompareResult{}, err
	}

	auditAfter, _ := json.Marshal(summary)
	s.recordAudit(ctx, AuditRecordInput{
		TenantID:   tid,
		ActorID:    firstNonEmpty(in.ActorID, "system"),
		ActorType:  firstNonEmpty(in.ActorType, auth.ActorTypeHuman),
		Role:       in.ActorRole,
		Action:     "connector.shadow_compare",
		ObjectType: "ConnectorSyncRun",
		ObjectID:   runID,
		AfterJSON:  string(auditAfter),
		ReasonCode: runStatus,
		TraceID:    in.TraceID,
		IP:         in.IP,
		DeviceID:   in.DeviceID,
	})

	s.mu.RLock()
	ontHook := s.ontologyHook
	s.mu.RUnlock()
	if ontHook != nil && len(entityMaps) > 0 {
		_ = ontHook(ctx, OntologyLineageInput{
			TenantID:   tid,
			Provider:   prov,
			RunID:      runID,
			SyncedAt:   completed,
			EntityMaps: entityMaps,
		})
	}

	return ShadowCompareResult{
		Run:        run,
		Summary:    summary,
		Mismatches: mismatches,
	}, nil
}

func (s *Service) ListRuns(ctx context.Context, tenantID, provider string, limit int) ([]ConnectorSyncRun, error) {
	if s == nil || s.repo == nil {
		return nil, ErrRepoUnavailable
	}
	runs, err := s.repo.ListSyncRuns(ctx, tenantID, provider, limit)
	if err != nil {
		return nil, err
	}
	if runs == nil {
		runs = []ConnectorSyncRun{}
	}
	return runs, nil
}

func (s *Service) GetRun(ctx context.Context, tenantID, runID string) (ConnectorSyncRun, error) {
	if s == nil || s.repo == nil {
		return ConnectorSyncRun{}, ErrRepoUnavailable
	}
	run, found, err := s.repo.GetSyncRun(ctx, tenantID, runID)
	if err != nil {
		return ConnectorSyncRun{}, err
	}
	if !found {
		return ConnectorSyncRun{}, ErrRunNotFound
	}
	return run, nil
}

func (s *Service) persistFailedRun(
	ctx context.Context,
	tenantID, runID, provider, accountLabel, direction, stream, idemKey, errMsg string,
	started, completed time.Time,
	in TriggerConnectorInput,
) {
	if s.repo == nil {
		return
	}
	run := ConnectorSyncRun{
		TenantID:       tenantID,
		RunID:          runID,
		Provider:       provider,
		AccountLabel:   normalizeAccountLabel(accountLabel),
		Direction:      direction,
		StreamName:     stream,
		Status:         RunStatusFailed,
		ErrorMessage:   errMsg,
		IdempotencyKey: "", // Do not burn idempotency key on transient/config failures so retry can succeed.
		StartedAt:      started,
		CompletedAt:    &completed,
		CreatedAt:      completed,
	}
	ts := completed.Format(time.RFC3339Nano)
	_ = s.repo.SaveSyncRunAndMaps(ctx, run, nil, []OutboxEmission{{
		AggregateType: events.AggregateConnectorSyncRun,
		AggregateID:   runID,
		Topic:         events.TopicConnectors,
		Event: events.ConnectorSyncEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventConnectorSyncFailed,
				Version:   1,
				Timestamp: ts,
			},
			TenantID:     tenantID,
			RunID:        runID,
			Provider:     provider,
			AccountLabel: run.AccountLabel,
			Direction:    direction,
			StreamName:   stream,
			Status:       RunStatusFailed,
			ErrorMessage: errMsg,
			TriggeredBy:  in.ActorID,
			StartedAt:    started.Format(time.RFC3339Nano),
			CompletedAt:  ts,
			OccurredAt:   ts,
		},
	}})
	s.recordAudit(ctx, AuditRecordInput{
		TenantID:   tenantID,
		ActorID:    firstNonEmpty(in.ActorID, "system"),
		ActorType:  firstNonEmpty(in.ActorType, auth.ActorTypeHuman),
		Role:       in.ActorRole,
		Action:     "connector." + strings.ToLower(direction),
		ObjectType: "ConnectorSyncRun",
		ObjectID:   runID,
		ReasonCode: RunStatusFailed,
		TraceID:    in.TraceID,
		IP:         in.IP,
		DeviceID:   in.DeviceID,
	})
	_ = idemKey
}

func (s *Service) recordAudit(ctx context.Context, ev AuditRecordInput) {
	s.mu.RLock()
	rec := s.audit
	s.mu.RUnlock()
	if rec == nil {
		return
	}
	_ = rec.RecordEvent(ctx, ev)
}
