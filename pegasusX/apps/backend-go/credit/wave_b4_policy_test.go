package credit

import (
	"context"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// capturingPolicyRepo wraps MemoryPolicyRepository and records emit payloads.
type capturingPolicyRepo struct {
	*MemoryPolicyRepository
	lastTypes []string
}

func (r *capturingPolicyRepo) UpsertProgram(ctx context.Context, p SupplierCreditProgram, emit func(outbox.TxnBuffer) error) error {
	if emit != nil {
		cap := &capturingTxnBuffer{types: &r.lastTypes}
		if err := emit(cap); err != nil {
			return err
		}
	}
	return r.MemoryPolicyRepository.UpsertProgram(ctx, p, nil)
}

func (r *capturingPolicyRepo) UpsertTerms(ctx context.Context, t RetailerPaymentTerms, emit func(outbox.TxnBuffer) error) error {
	if emit != nil {
		cap := &capturingTxnBuffer{types: &r.lastTypes}
		if err := emit(cap); err != nil {
			return err
		}
	}
	return r.MemoryPolicyRepository.UpsertTerms(ctx, t, nil)
}

type capturingTxnBuffer struct {
	types *[]string
}

func (b *capturingTxnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	// Payload is JSON; peek type via lightweight decode of common field.
	// EmitJSON sets AggregateType; event type is inside Payload.
	// We store AggregateType + a sentinel when payload non-empty.
	if len(e.Payload) > 0 {
		*b.types = append(*b.types, string(e.Payload))
	}
	return nil
}

func (b *capturingTxnBuffer) BufferAudit(context.Context, outbox.AuditEntry) error { return nil }

func TestEnableProgram_EmitsOutbox(t *testing.T) {
	repo := &capturingPolicyRepo{MemoryPolicyRepository: NewMemoryPolicyRepository()}
	svc := NewPolicyService(repo, nil)
	svc.SetNow(func() time.Time { return time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC) })
	ack := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	p, err := svc.EnableProgram(context.Background(), "sup-b4", "u1", "ADMIN", true, ack, nil)
	if err != nil {
		t.Fatalf("EnableProgram: %v", err)
	}
	if !p.ProgramEnabled {
		t.Fatal("expected program enabled")
	}
	if len(repo.lastTypes) == 0 {
		t.Fatal("expected outbox emit on program enable")
	}
	found := false
	for _, raw := range repo.lastTypes {
		if containsType(raw, events.EventSupplierCreditProgramChanged) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("payload missing SUPPLIER_CREDIT_PROGRAM_CHANGED: %v", repo.lastTypes)
	}
}

func TestAdminDisableProgram_EmitsOutbox(t *testing.T) {
	repo := &capturingPolicyRepo{MemoryPolicyRepository: NewMemoryPolicyRepository()}
	svc := NewPolicyService(repo, nil)
	svc.SetNow(func() time.Time { return time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC) })
	ack := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	if _, err := svc.EnableProgram(context.Background(), "sup-b4d", "u1", "ADMIN", true, ack, nil); err != nil {
		t.Fatalf("enable: %v", err)
	}
	repo.lastTypes = nil
	if err := svc.AdminDisableProgram(context.Background(), "sup-b4d", "u1", "ADMIN", "tkt-1", "risk"); err != nil {
		t.Fatalf("AdminDisableProgram: %v", err)
	}
	found := false
	for _, raw := range repo.lastTypes {
		if containsType(raw, events.EventSupplierCreditProgramChanged) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("disable missing program event: %v", repo.lastTypes)
	}
}

func TestEnableRelationship_TermsAndProfileEmit(t *testing.T) {
	repo := &capturingPolicyRepo{MemoryPolicyRepository: NewMemoryPolicyRepository()}
	creditRepo := &testCreditRepo{found: false}
	creditSvc := newTestService(creditRepo)
	svc := NewPolicyService(repo, creditSvc)
	svc.SetNow(func() time.Time { return time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC) })
	ack := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	if _, err := svc.EnableProgram(context.Background(), "sup-rel", "u1", "ADMIN", true, ack, &SupplierCreditProgram{
		GlobalTermsDays: 14, GlobalDefaultLimitMinor: 1_000_000,
	}); err != nil {
		t.Fatalf("enable program: %v", err)
	}
	repo.lastTypes = nil
	t1, err := svc.EnableRelationship(context.Background(), "sup-rel", "ret-1", "u1", "ADMIN", true, ack, 14, 0, 500_000, false)
	if err != nil {
		t.Fatalf("EnableRelationship: %v", err)
	}
	if !t1.CreditEnabled {
		t.Fatal("terms not enabled")
	}
	// Memory path: terms emit + profile via credit service (separate but both applied).
	foundTerms := false
	for _, raw := range repo.lastTypes {
		if containsType(raw, events.EventSupplierCreditTermsChanged) {
			foundTerms = true
			break
		}
	}
	if !foundTerms {
		t.Fatalf("missing terms event: %v", repo.lastTypes)
	}
	if creditRepo.lastProfile.RetailerID != "ret-1" || creditRepo.lastProfile.Status != StatusActive {
		t.Fatalf("profile not mirrored: %+v", creditRepo.lastProfile)
	}
}

func containsType(payload, eventType string) bool {
	return len(payload) > 0 && (stringIndex(payload, eventType) >= 0)
}

func stringIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
