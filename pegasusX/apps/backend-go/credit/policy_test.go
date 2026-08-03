package credit

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestEnableProgramRequiresWarningAck(t *testing.T) {
	svc := NewPolicyService(NewMemoryPolicyRepository(), nil)
	_, err := svc.EnableProgram(context.Background(), "sup-1", "u1", "ADMIN", false, time.Now(), nil)
	if !errors.Is(err, ErrWarningAckRequired) {
		t.Fatalf("got %v", err)
	}
}

func TestSelfServeDisableRejected(t *testing.T) {
	svc := NewPolicyService(NewMemoryPolicyRepository(), nil)
	if err := svc.RejectSelfServeDisable(); !errors.Is(err, ErrDisableRequiresSupport) {
		t.Fatalf("got %v", err)
	}
}

func TestEnableRelationshipIdempotent(t *testing.T) {
	repo := NewMemoryPolicyRepository()
	creditRepo := &testCreditRepo{found: false}
	creditSvc := newTestService(creditRepo)
	svc := NewPolicyService(repo, creditSvc)
	svc.SetNow(func() time.Time { return time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC) })
	ack := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	_, err := svc.EnableProgram(context.Background(), "sup-1", "u1", "ADMIN", true, ack, &SupplierCreditProgram{
		GlobalTermsDays: 14, GlobalDefaultLimitMinor: 1_000_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	t1, err := svc.EnableRelationship(context.Background(), "sup-1", "ret-1", "u1", "ADMIN", true, ack, 14, 0, 500_000, false)
	if err != nil {
		t.Fatal(err)
	}
	t2, err := svc.EnableRelationship(context.Background(), "sup-1", "ret-1", "u1", "ADMIN", true, ack, 14, 0, 500_000, false)
	if err != nil {
		t.Fatal(err)
	}
	if t1.Version != t2.Version {
		t.Fatalf("expected idempotent same version, got %d vs %d", t1.Version, t2.Version)
	}
}

func TestResolveDueAtUsesTermsDays(t *testing.T) {
	repo := NewMemoryPolicyRepository()
	svc := NewPolicyService(repo, nil)
	ack := time.Now().UTC()
	_, _ = svc.EnableProgram(context.Background(), "sup-1", "u1", "ADMIN", true, ack, &SupplierCreditProgram{
		GlobalTermsDays: 7, Timezone: "UTC",
	})
	_, _ = svc.EnableRelationship(context.Background(), "sup-1", "ret-1", "u1", "ADMIN", true, ack, 7, 0, 100, false)
	leave := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	due, days, err := svc.ResolveDueAt(context.Background(), "ret-1", "sup-1", leave)
	if err != nil {
		t.Fatal(err)
	}
	if days != 7 {
		t.Fatalf("days=%d", days)
	}
	if due.Before(leave) {
		t.Fatalf("due %v before leave %v", due, leave)
	}
}

func TestOverrideBeatsGlobal(t *testing.T) {
	repo := NewMemoryPolicyRepository()
	svc := NewPolicyService(repo, nil)
	ack := time.Now().UTC()
	_, _ = svc.EnableProgram(context.Background(), "sup-1", "u1", "ADMIN", true, ack, &SupplierCreditProgram{
		GlobalTermsDays: 30, GlobalDefaultLimitMinor: 100,
	})
	_, _ = svc.EnableRelationship(context.Background(), "sup-1", "ret-1", "u1", "ADMIN", true, ack, 10, 2, 999, false)
	resolved, err := svc.ResolveTermsFor(context.Background(), "ret-1", "sup-1")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.TermsDays != 10 || resolved.CreditLimitMinor != 999 {
		t.Fatalf("resolved=%+v", resolved)
	}
}

func TestAdminDisableKeepsCollectibleSemantics(t *testing.T) {
	repo := NewMemoryPolicyRepository()
	creditRepo := &testCreditRepo{
		found: true,
		profile: Profile{
			RetailerID: "ret-1", SupplierID: "sup-1",
			CreditLimitMinor: 1000, Status: StatusActive,
		},
	}
	creditSvc := newTestService(creditRepo)
	svc := NewPolicyService(repo, creditSvc)
	ack := time.Now().UTC()
	_, _ = svc.EnableProgram(context.Background(), "sup-1", "u1", "ADMIN", true, ack, nil)
	_, _ = svc.EnableRelationship(context.Background(), "sup-1", "ret-1", "u1", "ADMIN", true, ack, 30, 0, 1000, true)
	if err := svc.AdminDisableRelationship(context.Background(), "sup-1", "ret-1", "admin", "ADMIN", "T-1", "requested"); err != nil {
		t.Fatal(err)
	}
	ok, reason, _, err := svc.CreditPathAllowed(context.Background(), "ret-1", "sup-1")
	_ = reason
	if err != nil {
		t.Fatal(err)
	}
	// With flag off, CreditPathAllowed returns true; force-enable env for this test.
	t.Setenv("CREDIT_POLICY_V2_ENABLED", "true")
	ok, reason, _, err = svc.CreditPathAllowed(context.Background(), "ret-1", "sup-1")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatalf("expected disabled path blocked, reason=%s", reason)
	}
}

func TestCheckOrderFrozenCreditOnly(t *testing.T) {
	repo := &testCreditRepo{
		found: true,
		profile: Profile{
			RetailerID: "r", SupplierID: "s",
			CreditLimitMinor: 100_000, CurrentBalanceMinor: 0,
			Status: StatusFrozen,
		},
	}
	svc := newTestService(repo)
	res, err := svc.CheckOrder(context.Background(), "r", "s", 1000)
	if err != nil {
		t.Fatal(err)
	}
	if res.Allowed || res.Reason != "profile_frozen" {
		t.Fatalf("got %+v", res)
	}
}

func TestCheckOrderUsesReserved(t *testing.T) {
	repo := &testCreditRepo{
		found: true,
		profile: Profile{
			RetailerID: "r", SupplierID: "s",
			CreditLimitMinor: 10_000, CurrentBalanceMinor: 0, ReservedMinor: 9_000,
			Status: StatusActive,
		},
	}
	svc := newTestService(repo)
	res, err := svc.CheckOrder(context.Background(), "r", "s", 2_000)
	if err != nil {
		t.Fatal(err)
	}
	if res.Allowed {
		t.Fatalf("should breach with reserved: %+v", res)
	}
}
