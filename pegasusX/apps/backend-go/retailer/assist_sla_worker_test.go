package retailer

import (
	"testing"
	"time"
)

func TestAssistSLAWorkerMemoryBreachOnce(t *testing.T) {
	t.Parallel()
	enabled := true
	svc := NewService(ServiceConfig{
		Now:              time.Now,
		NewID:            func() string { return "ticket-1" },
		AssistSLAEnabled: &enabled,
	})
	past := time.Now().UTC().Add(-time.Minute).Format(time.RFC3339Nano)
	ticket := AssistTicketDTO{
		TicketID:        "ticket-1",
		RetailerID:      "ret-assist",
		LocationID:      "loc-1",
		SectionID:       "sec-1",
		Note:            "need help",
		Status:          AssistOpen,
		CreatedByUserID: "user-1",
		SlaDueAt:        past,
	}
	svc.mu.Lock()
	if svc.assistTickets == nil {
		svc.assistTickets = map[string]AssistTicketDTO{}
	}
	svc.assistTickets[ticket.TicketID] = ticket
	svc.mu.Unlock()

	n, err := svc.ProcessAssistSLABreachesOnce(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("breaches=%d want 1", n)
	}
	svc.mu.RLock()
	got := svc.assistTickets[ticket.TicketID]
	svc.mu.RUnlock()
	if got.SlaBreachedAt == "" {
		t.Fatal("expected SlaBreachedAt set")
	}

	n2, err := svc.ProcessAssistSLABreachesOnce(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if n2 != 0 {
		t.Fatalf("second pass breaches=%d want 0", n2)
	}
}

func TestAssistSLAWorkerDisabledNoOp(t *testing.T) {
	t.Parallel()
	disabled := false
	svc := NewService(ServiceConfig{
		Now:              time.Now,
		NewID:            func() string { return "t" },
		AssistSLAEnabled: &disabled,
	})
	if svc.assistSLAEnabled() {
		t.Fatal("expected disabled")
	}
}
