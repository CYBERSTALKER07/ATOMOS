package retailer

import (
	"context"
	"testing"
	"time"
)

func assistOn() *bool {
	v := true
	return &v
}

func assistOff() *bool {
	v := false
	return &v
}

type memNotifWriter struct {
	calls []string
}

func (m *memNotifWriter) CreateNotification(_ context.Context, recipientID, recipientRole, eventType, title, body, deepLink string) error {
	m.calls = append(m.calls, eventType+":"+recipientID+":"+title)
	return nil
}

// NotificationReader stubs (CreateNotification is via NotificationWriter assertion).
func (m *memNotifWriter) ListForRecipient(context.Context, string, int, int) ([]any, error) {
	return nil, nil
}
func (m *memNotifWriter) MarkRead(context.Context, string, []string) error { return nil }
func (m *memNotifWriter) MarkAllRead(context.Context, string) error         { return nil }
func (m *memNotifWriter) UnreadCount(context.Context, string) (int64, error) {
	return 0, nil
}

func TestSweepAssistSLA_DisabledNoop(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{
		Now: time.Now, NewID: func() string { return "a" },
		AssistSLAEnabled: assistOff(),
	})
	n, err := svc.SweepAssistSLA(context.Background())
	if err != nil || n != 0 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}

func TestSweepAssistSLA_NotifiesOnce(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	w := &memNotifWriter{}
	svc := NewService(ServiceConfig{
		Now:              func() time.Time { return now },
		NewID:            func() string { return "t-sla-1" },
		AssistSLAEnabled: assistOn(),
		NotifSvc:         w,
	})
	// OPEN ticket past due
	svc.mu.Lock()
	svc.assistTickets["t-sla-1"] = AssistTicketDTO{
		TicketID: "t-sla-1", RetailerID: "ret-a", LocationID: "loc-1", SectionID: "sec-1",
		Note: "customer needs help", Status: AssistOpen, CreatedByUserID: "u1",
		CreatedAt: now.Add(-20 * time.Minute).Format(time.RFC3339Nano),
		SlaDueAt:  now.Add(-5 * time.Minute).Format(time.RFC3339Nano),
	}
	// OPEN ticket still within SLA
	svc.assistTickets["t-ok"] = AssistTicketDTO{
		TicketID: "t-ok", RetailerID: "ret-a", LocationID: "loc-1", SectionID: "sec-1",
		Note: "fresh", Status: AssistOpen, CreatedByUserID: "u1",
		CreatedAt: now.Format(time.RFC3339Nano),
		SlaDueAt:  now.Add(10 * time.Minute).Format(time.RFC3339Nano),
	}
	// CLAIMED overdue should not fire
	svc.assistTickets["t-claimed"] = AssistTicketDTO{
		TicketID: "t-claimed", RetailerID: "ret-a", Status: AssistClaimed,
		SlaDueAt: now.Add(-time.Hour).Format(time.RFC3339Nano),
	}
	svc.mu.Unlock()

	n, err := svc.SweepAssistSLA(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("notified=%d want 1 (only open past SLA)", n)
	}
	if len(w.calls) == 0 {
		t.Fatal("expected notification calls")
	}
	svc.mu.RLock()
	notifiedAt := svc.assistTickets["t-sla-1"].SlaBreachNotifiedAt
	svc.mu.RUnlock()
	if notifiedAt == "" {
		t.Fatal("expected SlaBreachNotifiedAt set")
	}

	// Second sweep: idempotent
	w2 := &memNotifWriter{}
	svc.notifSvc = w2
	n2, err := svc.SweepAssistSLA(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n2 != 0 {
		t.Fatalf("second sweep notified=%d want 0", n2)
	}
}

func TestAssistSLAMinutes_Default15(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{Now: time.Now, NewID: func() string { return "x" }})
	if m := svc.assistSLAMinutes(context.Background(), "any"); m != 15 {
		t.Fatalf("sla=%d want 15", m)
	}
}
