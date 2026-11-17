package exportpacks

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// Service orchestrates M1.10 auditor/bank/tax pack generation, checksum/watermark enforcement,
// artifact storage, and audit/outbox emission.
type Service struct {
	repo          Repository
	collector     DatasetCollector
	audit         AuditRecorder
	mu            sync.RWMutex
	storage       PackStorage
	signingSecret string
	now           func() time.Time
}

// NewService constructs a Spanner-backed ExportPacks service.
func NewService(client *spanner.Client) *Service {
	if client == nil {
		return &Service{now: utcNow}
	}
	return &Service{
		repo:      NewSpannerRepository(client),
		collector: NewSpannerCollector(client, defaultCollectorBatchSize),
		now:       utcNow,
	}
}

// NewServiceWithDeps constructs a Service with custom Repository, DatasetCollector, and PackStorage.
func NewServiceWithDeps(repo Repository, collector DatasetCollector, storage PackStorage) *Service {
	return &Service{
		repo:      repo,
		collector: collector,
		storage:   storage,
		now:       utcNow,
	}
}

func utcNow() time.Time { return time.Now().UTC() }

func (s *Service) currentNow() time.Time {
	if s != nil && s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}

// SetAuditRecorder attaches an immutable hash-chained AuditEvents recorder.
func (s *Service) SetAuditRecorder(rec AuditRecorder) {
	if s != nil {
		s.audit = rec
	}
}

// SetPackStorage overrides the artifact storage backend.
func (s *Service) SetPackStorage(st PackStorage) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.storage = st
	s.mu.Unlock()
}

// SetSigningSecret configures the HMAC signing key embedded in manifest.json.
func (s *Service) SetSigningSecret(secret string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.signingSecret = strings.TrimSpace(secret)
	s.mu.Unlock()
}

// SetBatchSize adjusts keyset pagination page size on SpannerCollector (for tests/tuning).
func (s *Service) SetBatchSize(n int) {
	if s == nil || n <= 0 {
		return
	}
	if sc, ok := s.collector.(*SpannerCollector); ok {
		sc.batchSize = n
	}
}

func (s *Service) ensureStorage(ctx context.Context) (PackStorage, error) {
	s.mu.RLock()
	st := s.storage
	s.mu.RUnlock()
	if st != nil {
		return st, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.storage != nil {
		return s.storage, nil
	}
	created, err := NewDefaultPackStorage(ctx)
	if err != nil {
		return nil, err
	}
	s.storage = created
	return s.storage, nil
}

func (s *Service) getSigningSecret() string {
	s.mu.RLock()
	sec := s.signingSecret
	s.mu.RUnlock()
	if sec != "" {
		return sec
	}
	if env := strings.TrimSpace(os.Getenv("EXPORT_PACKS_SIGNING_SECRET")); env != "" {
		return env
	}
	return strings.TrimSpace(os.Getenv("JWT_SECRET"))
}

// CreatePack collects unlimited rows across all 15 audit tables, builds real multi-sheet XLSX,
// normalized CSVs, 1C XML, signed JSON manifest, and ZIP bundle, stores artifacts with SHA-256
// + watermark metadata, and atomically writes ExportPacks, OutboxEvents, and AuditEvents.
func (s *Service) CreatePack(ctx context.Context, in CreatePackInput) (*ExportPack, error) {
	if s == nil || s.repo == nil || s.collector == nil {
		return nil, ErrRepoUnavailable
	}

	tid := strings.TrimSpace(in.TenantID)
	eid := strings.TrimSpace(in.EntityID)
	if eid == "" {
		eid = tid
	}
	if tid == "" {
		tid = eid
	}
	if tid == "" || eid == "" {
		return nil, fmt.Errorf("%w: tenant_id and entity_id required", ErrInvalidInput)
	}

	// AUDITOR & AGENT actors must never generate export packs.
	claims, hasClaims := auth.FromContext(ctx)
	actorRole := strings.TrimSpace(in.ActorRole)
	actorType := strings.TrimSpace(in.ActorType)
	if hasClaims {
		if actorRole == "" {
			actorRole = auth.EffectiveSupplierRole(claims)
		}
		if actorType == "" {
			actorType = strings.TrimSpace(claims.ActorType)
		}
		if auth.IsReadOnlyAuditor(claims) || auth.IsAgentActor(claims) {
			return nil, fmt.Errorf("%w: auditor or agent actor cannot create export packs", ErrForbidden)
		}
	}
	if strings.EqualFold(actorRole, auth.SupplierRoleAuditor) || strings.EqualFold(actorRole, auth.SupplierRoleAgent) || strings.EqualFold(actorType, auth.ActorTypeAgent) {
		return nil, fmt.Errorf("%w: role %q is read-only/restricted for export pack creation", ErrForbidden, actorRole)
	}

	packType, ok := NormalizePackType(in.PackType)
	if !ok {
		return nil, fmt.Errorf("%w: unsupported pack_type %q (expected AUDIT, BANK, or TAX)", ErrInvalidInput, in.PackType)
	}
	format, ok := NormalizeFormat(in.Format)
	if !ok {
		return nil, fmt.Errorf("%w: unsupported format %q (expected zip, xlsx, csv, xml, or json)", ErrInvalidInput, in.Format)
	}

	now := s.currentNow()
	ttlHours := in.TTLHours
	if ttlHours < 0 {
		return nil, fmt.Errorf("%w: ttl_hours must be positive", ErrInvalidInput)
	}
	if ttlHours == 0 {
		ttlHours = DefaultTTLHours
	}
	expiresAt := now.Add(time.Duration(ttlHours) * time.Hour)

	var from, to time.Time
	switch {
	case in.FromDate != nil && in.ToDate != nil:
		from = in.FromDate.UTC()
		to = in.ToDate.UTC()
	case in.FromDate != nil:
		from = in.FromDate.UTC()
		to = now
	case in.ToDate != nil:
		to = in.ToDate.UTC()
		from = to.AddDate(-1, 0, 0)
	default:
		to = now
		from = now.AddDate(-1, 0, 0)
	}
	if from.After(to) {
		return nil, fmt.Errorf("%w: from_date cannot be after to_date", ErrInvalidInput)
	}

	actor := strings.TrimSpace(in.ActorID)
	if actor == "" {
		actor = strings.TrimSpace(auth.ActorFromContext(ctx))
	}
	if actor == "" {
		actor = "finance-operator"
	}
	if actorType == "" {
		actorType = auth.ActorTypeHuman
	}

	storage, err := s.ensureStorage(ctx)
	if err != nil {
		return nil, err
	}

	var datasets *PackDatasets
	if packType == PackTypeBank {
		if bc, ok := s.collector.(BankDatasetCollector); ok {
			datasets, err = bc.CollectBank(ctx, eid, strings.TrimSpace(in.PeriodID), from, to)
		} else {
			datasets, err = s.collector.Collect(ctx, eid, strings.TrimSpace(in.PeriodID), from, to)
		}
	} else {
		datasets, err = s.collector.Collect(ctx, eid, strings.TrimSpace(in.PeriodID), from, to)
	}
	if err != nil {
		return nil, err
	}

	packID := "pack-" + uuid.NewString()
	dlToken, err := generateDownloadToken()
	if err != nil {
		return nil, fmt.Errorf("generate download token: %w", err)
	}
	watermark := formatWatermark(actor, tid, packID, expiresAt)

	pack := &ExportPack{
		TenantID:      tid,
		PackID:        packID,
		PackType:      packType,
		EntityID:      eid,
		PeriodID:      strings.TrimSpace(in.PeriodID),
		FromDate:      from,
		ToDate:        to,
		Format:        format,
		Status:        StatusReady,
		DownloadToken: dlToken,
		WatermarkText: watermark,
		RowCount:      datasets.Stats().Total(),
		CreatedBy:     actor,
		ExpiresAt:     expiresAt,
		CreatedAt:     now,
	}

	bundle, err := buildPackBundle(pack, datasets, s.getSigningSecret())
	if err != nil {
		return nil, err
	}
	if bundle.Manifest != nil {
		pack.RowCount = bundle.Manifest.TotalRows
	}

	orderedFormats := []string{FormatZip, FormatXLSX, FormatCSV, FormatXML, FormatJSON}
	storedMetas := make([]ArtifactMeta, 0, len(orderedFormats))
	for _, f := range orderedFormats {
		art := bundle.Artifacts[f]
		key := fmt.Sprintf("%s/%s/%s", tid, packID, art.Meta.Filename)
		uri, perr := storage.PutObject(ctx, key, art.Meta.ContentType, art.Data)
		if perr != nil {
			return nil, fmt.Errorf("store %s artifact: %w", f, perr)
		}
		art.Meta.StorageURI = uri
		storedMetas = append(storedMetas, art.Meta)
		if f == format {
			pack.Sha256Hex = art.Meta.Sha256Hex
		}
	}

	artifactsBytes, err := json.Marshal(storedMetas)
	if err != nil {
		return nil, fmt.Errorf("marshal artifacts metadata: %w", err)
	}
	pack.Manifest = bundle.Manifest
	pack.ManifestJSON = string(bundle.ManifestBytes)
	pack.Artifacts = storedMetas
	pack.ArtifactsJSON = string(artifactsBytes)
	pack.DownloadURL = "/v1/finance/packs/" + packID + "/download"

	auditFn := func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		if s.audit == nil {
			return nil
		}
		afterPayload, _ := json.Marshal(map[string]any{
			"pack_id":    pack.PackID,
			"pack_type":  pack.PackType,
			"entity_id":  pack.EntityID,
			"period_id":  pack.PeriodID,
			"format":     pack.Format,
			"row_count":  pack.RowCount,
			"sha256_hex": pack.Sha256Hex,
			"expires_at": pack.ExpiresAt.Format(time.RFC3339),
		})
		ev := AuditRecordInput{
			TenantID:   tid,
			ActorID:    actor,
			ActorType:  actorType,
			Role:       actorRole,
			Action:     "EXPORT_PACK_CREATED",
			ObjectType: "ExportPack",
			ObjectID:   pack.PackID,
			AfterJSON:  string(afterPayload),
			ReasonCode: pack.PackType,
			TraceID:    in.TraceID,
			IP:         in.IP,
			DeviceID:   in.DeviceID,
		}
		if txn != nil {
			return s.audit.RecordEventInTxn(txCtx, txn, ev)
		}
		return s.audit.RecordEvent(txCtx, ev)
	}

	if err := s.repo.CreatePack(ctx, pack, auditFn); err != nil {
		return nil, err
	}
	return pack, nil
}

// ListPacks returns packs scoped to tenantID/entityID ordered newest first.
func (s *Service) ListPacks(ctx context.Context, tenantID, entityID string, limit int) ([]*ExportPack, error) {
	if s == nil || s.repo == nil {
		return nil, ErrRepoUnavailable
	}
	packs, err := s.repo.ListPacks(ctx, tenantID, entityID, limit)
	if err != nil {
		return nil, err
	}
	now := s.currentNow()
	claims, _ := auth.FromContext(ctx)
	redactToken := auth.IsReadOnlyAuditor(claims)
	for _, p := range packs {
		s.enrichPackView(p, now, redactToken)
	}
	return packs, nil
}

// GetPack fetches a single pack's metadata and manifest by tenant and pack ID.
func (s *Service) GetPack(ctx context.Context, tenantID, packID string) (*ExportPack, error) {
	if s == nil || s.repo == nil {
		return nil, ErrRepoUnavailable
	}
	if strings.TrimSpace(packID) == "" {
		return nil, fmt.Errorf("%w: pack_id required", ErrInvalidInput)
	}
	var pack *ExportPack
	var err error
	if strings.TrimSpace(tenantID) != "" {
		pack, err = s.repo.GetPack(ctx, tenantID, packID)
	} else {
		pack, err = s.repo.GetPackByID(ctx, packID)
	}
	if err != nil {
		return nil, err
	}
	claims, _ := auth.FromContext(ctx)
	s.enrichPackView(pack, s.currentNow(), auth.IsReadOnlyAuditor(claims))
	return pack, nil
}

// DownloadPack validates expiry, authorization/download token, retrieves and checksum-verifies
// the requested format bytes, and records an immutable EXPORT_PACK_DOWNLOADED audit event.
func (s *Service) DownloadPack(ctx context.Context, in DownloadPackInput) (*DownloadResult, error) {
	if s == nil || s.repo == nil {
		return nil, ErrRepoUnavailable
	}
	pid := strings.TrimSpace(in.PackID)
	if pid == "" {
		return nil, fmt.Errorf("%w: pack_id required", ErrInvalidInput)
	}

	tid := strings.TrimSpace(in.TenantID)
	reqEntity := strings.TrimSpace(in.EntityID)
	dlToken := strings.TrimSpace(in.DownloadToken)

	var pack *ExportPack
	var err error
	if tid != "" {
		pack, err = s.repo.GetPack(ctx, tid, pid)
	} else {
		if dlToken == "" {
			return nil, fmt.Errorf("%w: authentication or download token required", ErrForbidden)
		}
		pack, err = s.repo.GetPackByID(ctx, pid)
	}
	if err != nil {
		return nil, err
	}

	if tid != "" && pack.TenantID != tid {
		return nil, fmt.Errorf("%w: tenant scope mismatch", ErrForbidden)
	}
	if reqEntity != "" && pack.EntityID != reqEntity && pack.TenantID != reqEntity {
		return nil, fmt.Errorf("%w: entity scope mismatch", ErrForbidden)
	}
	if dlToken != "" {
		if pack.DownloadToken == "" || subtle.ConstantTimeCompare([]byte(dlToken), []byte(pack.DownloadToken)) != 1 {
			return nil, fmt.Errorf("%w: download token mismatch", ErrInvalidDownloadToken)
		}
	} else if tid == "" {
		return nil, fmt.Errorf("%w: download token mismatch", ErrInvalidDownloadToken)
	}

	now := s.currentNow()
	if !pack.ExpiresAt.After(now) {
		return nil, fmt.Errorf("%w: export pack %s expired at %s", ErrPackExpired, pack.PackID, pack.ExpiresAt.UTC().Format(time.RFC3339))
	}

	reqFormat := strings.TrimSpace(in.Format)
	if reqFormat == "" {
		reqFormat = pack.Format
	}
	format, ok := NormalizeFormat(reqFormat)
	if !ok {
		return nil, fmt.Errorf("%w: unsupported download format %q", ErrInvalidInput, in.Format)
	}

	var chosen *ArtifactMeta
	for i := range pack.Artifacts {
		if pack.Artifacts[i].Format == format {
			chosen = &pack.Artifacts[i]
			break
		}
	}
	if chosen == nil {
		return nil, fmt.Errorf("%w: format %q not available in pack %s", ErrNotFound, format, pack.PackID)
	}

	storage, err := s.ensureStorage(ctx)
	if err != nil {
		return nil, err
	}
	data, err := storage.GetObject(ctx, chosen.StorageURI)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(data)
	actualHex := hex.EncodeToString(sum[:])
	if chosen.Sha256Hex != "" && !strings.EqualFold(actualHex, chosen.Sha256Hex) {
		return nil, fmt.Errorf("export pack checksum mismatch for %s (%s): got %s want %s", pack.PackID, format, actualHex, chosen.Sha256Hex)
	}

	actor := strings.TrimSpace(in.ActorID)
	if actor == "" {
		actor = strings.TrimSpace(auth.ActorFromContext(ctx))
	}
	if actor == "" && dlToken != "" {
		actor = "token-holder"
	}
	actorType := strings.TrimSpace(in.ActorType)
	if actorType == "" {
		actorType = auth.ActorTypeHuman
	}

	if s.audit != nil {
		afterPayload, _ := json.Marshal(map[string]any{
			"pack_id":    pack.PackID,
			"format":     format,
			"sha256_hex": actualHex,
			"watermark":  pack.WatermarkText,
			"via_token":  dlToken != "",
		})
		if aerr := s.audit.RecordEvent(ctx, AuditRecordInput{
			TenantID:   pack.TenantID,
			ActorID:    actor,
			ActorType:  actorType,
			Role:       in.ActorRole,
			Action:     "EXPORT_PACK_DOWNLOADED",
			ObjectType: "ExportPack",
			ObjectID:   pack.PackID,
			AfterJSON:  string(afterPayload),
			ReasonCode: format,
			TraceID:    in.TraceID,
			IP:         in.IP,
			DeviceID:   in.DeviceID,
		}); aerr != nil {
			return nil, fmt.Errorf("record EXPORT_PACK_DOWNLOADED audit event: %w", aerr)
		}
	}

	return &DownloadResult{
		Pack:        pack,
		Artifact:    *chosen,
		Data:        data,
		ContentType: chosen.ContentType,
		Filename:    chosen.Filename,
		Watermark:   pack.WatermarkText,
		Sha256Hex:   actualHex,
	}, nil
}

func (s *Service) enrichPackView(p *ExportPack, now time.Time, redactToken bool) {
	if p == nil {
		return
	}
	p.Expired = !p.ExpiresAt.After(now)
	if p.Expired {
		p.Status = StatusExpired
	}
	p.DownloadURL = "/v1/finance/packs/" + p.PackID + "/download"
	if redactToken {
		p.DownloadToken = ""
	}
}

func generateDownloadToken() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}
