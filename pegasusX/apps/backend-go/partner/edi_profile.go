package partner

import (
	"context"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
)

// Default EDI pack: all core lifecycle docs enabled (backward compatible).
const EdiPackEdifactLiteV1 = "edifact_lite_v1"

// EdiProfile is per-tenant message/GLN policy (G5.A).
type EdiProfile struct {
	TenantType      string    `json:"tenant_type"`
	TenantID        string    `json:"tenant_id"`
	PackName        string    `json:"pack_name"`
	OurGLN          string    `json:"our_gln,omitempty"`
	TheirGLN        string    `json:"their_gln,omitempty"`
	EnabledDocTypes []string  `json:"enabled_doc_types"`
	RequireCONTRL   bool      `json:"require_contrl"`
	RequireAPERAK   bool      `json:"require_aperak"`
	AsnAsDesadv     bool      `json:"asn_as_desadv"`
	Transport       string    `json:"transport,omitempty"` // LOCAL|SFTP|AS2|ANY
	UpdatedAt       time.Time `json:"updated_at,omitempty"`
}

// EdiProfileRepository loads/stores tenant EDI profiles.
type EdiProfileRepository interface {
	Get(ctx context.Context, tenantType, tenantID string) (EdiProfile, bool, error)
	Upsert(ctx context.Context, p EdiProfile) error
}

// DefaultEdiProfile enables the full EDI-lite lifecycle set.
func DefaultEdiProfile(tenantType, tenantID string) EdiProfile {
	return EdiProfile{
		TenantType: strings.ToUpper(strings.TrimSpace(tenantType)),
		TenantID:   strings.TrimSpace(tenantID),
		PackName:   EdiPackEdifactLiteV1,
		EnabledDocTypes: []string{
			EdiDocORDERS, EdiDocORDRSP, EdiDocDESADV, EdiDocINVOIC,
			EdiDocCONTRL, EdiDocAPERAK, EdiDocPRICAT, EdiDocINVRPT,
			EdiDocSLSRPT, EdiDocRECADV, EdiDocORDCHG, EdiDocDELFOR, EdiDocREMADV,
		},
		RequireCONTRL: true,
		RequireAPERAK: true,
		AsnAsDesadv:   true,
		Transport:     "ANY",
		UpdatedAt:     time.Now().UTC(),
	}
}

// DocEnabled reports whether docType is allowed for the profile (case-insensitive).
func (p EdiProfile) DocEnabled(docType string) bool {
	want := strings.ToUpper(strings.TrimSpace(docType))
	if want == "" {
		return false
	}
	if len(p.EnabledDocTypes) == 0 {
		return true // empty list = no restriction (legacy)
	}
	for _, d := range p.EnabledDocTypes {
		if strings.ToUpper(strings.TrimSpace(d)) == want {
			return true
		}
	}
	return false
}

// MemoryEdiProfiles is an in-process profile store (tests + non-Spanner).
type MemoryEdiProfiles struct {
	mu   sync.RWMutex
	rows map[string]EdiProfile
}

func NewMemoryEdiProfiles() *MemoryEdiProfiles {
	return &MemoryEdiProfiles{rows: map[string]EdiProfile{}}
}

func ediProfileKey(tt, tid string) string {
	return strings.ToUpper(strings.TrimSpace(tt)) + "|" + strings.TrimSpace(tid)
}

func (m *MemoryEdiProfiles) Get(_ context.Context, tenantType, tenantID string) (EdiProfile, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.rows[ediProfileKey(tenantType, tenantID)]
	return p, ok, nil
}

func (m *MemoryEdiProfiles) Upsert(_ context.Context, p EdiProfile) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p.TenantType = strings.ToUpper(strings.TrimSpace(p.TenantType))
	p.TenantID = strings.TrimSpace(p.TenantID)
	p.UpdatedAt = time.Now().UTC()
	if p.PackName == "" {
		p.PackName = EdiPackEdifactLiteV1
	}
	m.rows[ediProfileKey(p.TenantType, p.TenantID)] = p
	return nil
}

// SpannerEdiProfiles persists PartnerEdiProfiles.
type SpannerEdiProfiles struct {
	Client *spanner.Client
}

func (s *SpannerEdiProfiles) Get(ctx context.Context, tenantType, tenantID string) (EdiProfile, bool, error) {
	if s == nil || s.Client == nil {
		return EdiProfile{}, false, nil
	}
	tt := strings.ToUpper(strings.TrimSpace(tenantType))
	tid := strings.TrimSpace(tenantID)
	row, err := s.Client.Single().ReadRow(ctx, "PartnerEdiProfiles", spanner.Key{tt, tid},
		[]string{"TenantType", "TenantId", "PackName", "OurGln", "TheirGln", "EnabledDocTypes",
			"RequireContrl", "RequireAperak", "AsnAsDesadv", "Transport", "UpdatedAt"})
	if err != nil {
		if spanner.ErrCode(err) == 5 {
			return EdiProfile{}, false, nil
		}
		// Table missing.
		if strings.Contains(err.Error(), "PartnerEdiProfiles") {
			return EdiProfile{}, false, nil
		}
		return EdiProfile{}, false, err
	}
	var p EdiProfile
	var our, their, enabled, transport spanner.NullString
	var updated time.Time
	if err := row.Columns(&p.TenantType, &p.TenantID, &p.PackName, &our, &their, &enabled,
		&p.RequireCONTRL, &p.RequireAPERAK, &p.AsnAsDesadv, &transport, &updated); err != nil {
		return EdiProfile{}, false, err
	}
	if our.Valid {
		p.OurGLN = our.StringVal
	}
	if their.Valid {
		p.TheirGLN = their.StringVal
	}
	if enabled.Valid && enabled.StringVal != "" {
		for _, part := range strings.Split(enabled.StringVal, ",") {
			if t := strings.TrimSpace(part); t != "" {
				p.EnabledDocTypes = append(p.EnabledDocTypes, strings.ToUpper(t))
			}
		}
	}
	if transport.Valid {
		p.Transport = transport.StringVal
	}
	p.UpdatedAt = updated
	return p, true, nil
}

func (s *SpannerEdiProfiles) Upsert(ctx context.Context, p EdiProfile) error {
	if s == nil || s.Client == nil {
		return nil
	}
	p.TenantType = strings.ToUpper(strings.TrimSpace(p.TenantType))
	p.TenantID = strings.TrimSpace(p.TenantID)
	if p.PackName == "" {
		p.PackName = EdiPackEdifactLiteV1
	}
	docs := strings.Join(p.EnabledDocTypes, ",")
	now := time.Now().UTC()
	_, err := s.Client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("PartnerEdiProfiles", map[string]interface{}{
			"TenantType":      p.TenantType,
			"TenantId":        p.TenantID,
			"PackName":        p.PackName,
			"OurGln":          nullStr(p.OurGLN),
			"TheirGln":        nullStr(p.TheirGLN),
			"EnabledDocTypes": docs,
			"RequireContrl":   p.RequireCONTRL,
			"RequireAperak":   p.RequireAPERAK,
			"AsnAsDesadv":     p.AsnAsDesadv,
			"Transport":       nullStr(p.Transport),
			"UpdatedAt":       now,
		}),
	})
	if err != nil && strings.Contains(err.Error(), "PartnerEdiProfiles") {
		return nil
	}
	return err
}

func nullStr(s string) interface{} {
	s = strings.TrimSpace(s)
	if s == "" {
		return spanner.NullString{}
	}
	return s
}

// ResolveEdiProfile returns stored profile or DefaultEdiProfile.
func ResolveEdiProfile(ctx context.Context, repo EdiProfileRepository, tenantType, tenantID string) EdiProfile {
	if repo != nil {
		if p, ok, err := repo.Get(ctx, tenantType, tenantID); err == nil && ok {
			return p
		}
	}
	return DefaultEdiProfile(tenantType, tenantID)
}
