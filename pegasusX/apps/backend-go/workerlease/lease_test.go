package workerlease

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSpannerManager_NilClient_FailsClosed(t *testing.T) {
	ctx := context.Background()
	var nilMgr *SpannerManager
	if _, err := nilMgr.TryAcquireOrRenew(ctx, "sweeper:test", "pod-1", 10*time.Second); !errors.Is(err, ErrSpannerClientNil) {
		t.Fatalf("expected ErrSpannerClientNil on nil receiver, got %v", err)
	}

	mgr := NewSpannerManager(nil)
	if _, err := mgr.TryAcquireOrRenew(ctx, "sweeper:test", "pod-1", 10*time.Second); !errors.Is(err, ErrSpannerClientNil) {
		t.Fatalf("expected ErrSpannerClientNil on TryAcquireOrRenew, got %v", err)
	}
	if err := mgr.Release(ctx, "sweeper:test", "pod-1"); !errors.Is(err, ErrSpannerClientNil) {
		t.Fatalf("expected ErrSpannerClientNil on Release, got %v", err)
	}
	if held, err := mgr.IsHeld(ctx, "sweeper:test"); held || !errors.Is(err, ErrSpannerClientNil) {
		t.Fatalf("expected (false, ErrSpannerClientNil) on IsHeld, got (%v, %v)", held, err)
	}
	if _, _, err := mgr.GetLease(ctx, "sweeper:test"); !errors.Is(err, ErrSpannerClientNil) {
		t.Fatalf("expected ErrSpannerClientNil on GetLease, got %v", err)
	}
}

func TestMemoryManager_AcquireRenewExpireFencedRelease(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC)
	var mu sync.Mutex
	clock := func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return now
	}
	advance := func(d time.Duration) {
		mu.Lock()
		now = now.Add(d)
		mu.Unlock()
	}

	mgr := NewMemoryManager().WithClock(clock)

	// Pod A acquires.
	ok, err := mgr.TryAcquireOrRenew(ctx, "sweeper:preorder", "pod-a", 30*time.Second)
	if err != nil || !ok {
		t.Fatalf("pod-a acquire failed: ok=%v err=%v", ok, err)
	}
	if held, err := mgr.IsHeld(ctx, "sweeper:preorder"); err != nil || !held {
		t.Fatalf("expected lease held, got held=%v err=%v", held, err)
	}

	// Pod B is rejected while lease is active.
	ok, err = mgr.TryAcquireOrRenew(ctx, "sweeper:preorder", "pod-b", 30*time.Second)
	if err != nil || ok {
		t.Fatalf("pod-b should not acquire active lease: ok=%v err=%v", ok, err)
	}

	// Pod A renews at +15s; expiry moves to +45s.
	advance(15 * time.Second)
	ok, err = mgr.TryAcquireOrRenew(ctx, "sweeper:preorder", "pod-a", 30*time.Second)
	if err != nil || !ok {
		t.Fatalf("pod-a renew failed: ok=%v err=%v", ok, err)
	}

	// Pod B tries fenced release on Pod A's lease -> must NOT delete it.
	if err := mgr.Release(ctx, "sweeper:preorder", "pod-b"); err != nil {
		t.Fatalf("fenced release unexpected err: %v", err)
	}
	if held, err := mgr.IsHeld(ctx, "sweeper:preorder"); err != nil || !held {
		t.Fatalf("pod-b release must not revoke pod-a lease: held=%v err=%v", held, err)
	}

	// At +44s, still held by Pod A.
	advance(29 * time.Second)
	ok, err = mgr.TryAcquireOrRenew(ctx, "sweeper:preorder", "pod-b", 30*time.Second)
	if err != nil || ok {
		t.Fatalf("pod-b acquired before expiry: ok=%v err=%v", ok, err)
	}

	// At +45s (ExpiresAt == now), lease is expired so Pod B takes over.
	advance(1 * time.Second)
	ok, err = mgr.TryAcquireOrRenew(ctx, "sweeper:preorder", "pod-b", 30*time.Second)
	if err != nil || !ok {
		t.Fatalf("pod-b takeover after expiry failed: ok=%v err=%v", ok, err)
	}

	// Pod A cannot release Pod B's lease.
	if err := mgr.Release(ctx, "sweeper:preorder", "pod-a"); err != nil {
		t.Fatalf("pod-a fenced release err: %v", err)
	}
	if held, err := mgr.IsHeld(ctx, "sweeper:preorder"); err != nil || !held {
		t.Fatalf("pod-b lease lost after pod-a release: held=%v err=%v", held, err)
	}

	// Pod B releases cleanly.
	if err := mgr.Release(ctx, "sweeper:preorder", "pod-b"); err != nil {
		t.Fatalf("pod-b release failed: %v", err)
	}
	if held, err := mgr.IsHeld(ctx, "sweeper:preorder"); err != nil || held {
		t.Fatalf("expected lease cleared after pod-b release: held=%v err=%v", held, err)
	}
}

func TestValidation(t *testing.T) {
	ctx := context.Background()
	mgr := NewMemoryManager()
	longStr := strings.Repeat("x", MaxLeaseNameLen+1)

	if _, err := mgr.TryAcquireOrRenew(ctx, "", "h", time.Second); !errors.Is(err, ErrInvalidLeaseName) {
		t.Fatalf("expected ErrInvalidLeaseName, got %v", err)
	}
	if _, err := mgr.TryAcquireOrRenew(ctx, longStr, "h", time.Second); !errors.Is(err, ErrInvalidLeaseName) {
		t.Fatalf("expected ErrInvalidLeaseName for >128 chars, got %v", err)
	}
	if _, err := mgr.TryAcquireOrRenew(ctx, "lease", "", time.Second); !errors.Is(err, ErrInvalidHolderID) {
		t.Fatalf("expected ErrInvalidHolderID, got %v", err)
	}
	if _, err := mgr.TryAcquireOrRenew(ctx, "lease", longStr, time.Second); !errors.Is(err, ErrInvalidHolderID) {
		t.Fatalf("expected ErrInvalidHolderID for >128 chars, got %v", err)
	}
	if _, err := mgr.TryAcquireOrRenew(ctx, "lease", "h", 0); !errors.Is(err, ErrInvalidTTL) {
		t.Fatalf("expected ErrInvalidTTL, got %v", err)
	}
}

func TestDefaultHolderID(t *testing.T) {
	id1 := DefaultHolderID("api-sweeper")
	id2 := DefaultHolderID("api-sweeper")
	if id1 == "" || id2 == "" || id1 == id2 {
		t.Fatalf("expected unique non-empty holder IDs, got %q and %q", id1, id2)
	}
	if len(id1) > MaxHolderIDLen || len(id2) > MaxHolderIDLen {
		t.Fatalf("holder ID exceeds %d bytes: %d / %d", MaxHolderIDLen, len(id1), len(id2))
	}
}

type scriptableManager struct {
	mu          sync.Mutex
	calls       int
	acquireFn   func(call int, leaseName, holderID string, ttl time.Duration) (bool, error)
	releaseHits int
	held        bool
}

func (s *scriptableManager) TryAcquireOrRenew(_ context.Context, leaseName, holderID string, ttl time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	return s.acquireFn(s.calls, leaseName, holderID, ttl)
}

func (s *scriptableManager) Release(_ context.Context, _ string, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.releaseHits++
	s.held = false
	return nil
}

func (s *scriptableManager) IsHeld(_ context.Context, _ string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.held, nil
}

func TestWithLease_FencesImmediatelyWhenStolen(t *testing.T) {
	mgr := &scriptableManager{
		acquireFn: func(call int, _, _ string, _ time.Duration) (bool, error) {
			if call == 1 {
				return true, nil
			}
			return false, nil // stolen on first renew
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var cancelledByLease atomic.Bool
	acquired, err := WithLease(ctx, mgr, Config{
		LeaseName:     "sweeper:fiscal",
		HolderID:      "pod-1",
		TTL:           90 * time.Millisecond,
		RenewInterval: 15 * time.Millisecond,
	}, func(leaseCtx context.Context) error {
		<-leaseCtx.Done()
		cancelledByLease.Store(true)
		return leaseCtx.Err()
	})
	if !acquired {
		t.Fatal("expected initial acquisition to succeed")
	}
	if !errors.Is(err, ErrLeaseLost) {
		t.Fatalf("expected ErrLeaseLost, got %v", err)
	}
	if !cancelledByLease.Load() {
		t.Fatal("expected fn context to cancel when lease was lost")
	}
	if mgr.releaseHits != 0 {
		t.Fatalf("expected stolen lease not to call Release, got %d calls", mgr.releaseHits)
	}
}

func TestWithLease_FencesOnPersistentRenewErrorsBeforeExpiry(t *testing.T) {
	errSpannerDown := errors.New("spanner unavailable")
	mgr := &scriptableManager{
		acquireFn: func(call int, _, _ string, _ time.Duration) (bool, error) {
			if call == 1 {
				return true, nil
			}
			return false, errSpannerDown
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	acquired, err := WithLease(ctx, mgr, Config{
		LeaseName:     "sweeper:credit",
		HolderID:      "pod-1",
		TTL:           90 * time.Millisecond,
		RenewInterval: 15 * time.Millisecond,
	}, func(leaseCtx context.Context) error {
		<-leaseCtx.Done()
		return nil
	})
	if !acquired {
		t.Fatal("expected initial acquisition to succeed")
	}
	if !errors.Is(err, ErrLeaseLost) {
		t.Fatalf("expected ErrLeaseLost when renew errors reach expiry window, got %v", err)
	}
}

func TestWithLease_ShortLivedTaskReleasesLease(t *testing.T) {
	ctx := context.Background()
	mgr := NewMemoryManager()

	acquired, err := WithLease(ctx, mgr, Config{
		LeaseName: "backfill:scheduled_reservations",
		HolderID:  "pod-1",
		TTL:       30 * time.Second,
	}, func(_ context.Context) error {
		return nil
	})
	if err != nil || !acquired {
		t.Fatalf("expected clean one-shot completion, got acquired=%v err=%v", acquired, err)
	}
	if held, err := mgr.IsHeld(ctx, "backfill:scheduled_reservations"); err != nil || held {
		t.Fatalf("expected lease released after task exit, got held=%v err=%v", held, err)
	}
}

func TestRunWithLease_MutualExclusionAndFailover(t *testing.T) {
	mgr := NewMemoryManager()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var active atomic.Int32
	var maxActive atomic.Int32
	var podARuns atomic.Int32
	var podBRuns atomic.Int32

	podACtx, cancelPodA := context.WithCancel(ctx)
	podAStarted := make(chan struct{}, 1)
	podADone := make(chan error, 1)
	go func() {
		podADone <- RunWithLease(podACtx, mgr, Config{
			LeaseName:     "sweeper:negotiation",
			HolderID:      "pod-a",
			TTL:           80 * time.Millisecond,
			RenewInterval: 20 * time.Millisecond,
			RetryInterval: 15 * time.Millisecond,
		}, func(leaseCtx context.Context) error {
			cur := active.Add(1)
			for {
				prev := maxActive.Load()
				if cur <= prev || maxActive.CompareAndSwap(prev, cur) {
					break
				}
			}
			podARuns.Add(1)
			select {
			case podAStarted <- struct{}{}:
			default:
			}
			<-leaseCtx.Done()
			active.Add(-1)
			return nil
		})
	}()

	select {
	case <-podAStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for pod-a to acquire lease")
	}

	podBStarted := make(chan struct{}, 1)
	podBDone := make(chan error, 1)
	go func() {
		podBDone <- RunWithLease(ctx, mgr, Config{
			LeaseName:     "sweeper:negotiation",
			HolderID:      "pod-b",
			TTL:           80 * time.Millisecond,
			RenewInterval: 20 * time.Millisecond,
			RetryInterval: 15 * time.Millisecond,
		}, func(leaseCtx context.Context) error {
			cur := active.Add(1)
			for {
				prev := maxActive.Load()
				if cur <= prev || maxActive.CompareAndSwap(prev, cur) {
					break
				}
			}
			podBRuns.Add(1)
			select {
			case podBStarted <- struct{}{}:
			default:
			}
			<-leaseCtx.Done()
			active.Add(-1)
			return nil
		})
	}()

	// Let standby pod-b poll at least twice while pod-a holds the lease.
	time.Sleep(50 * time.Millisecond)
	if podBRuns.Load() != 0 {
		t.Fatalf("pod-b must not run while pod-a holds lease, runs=%d", podBRuns.Load())
	}

	// Simulate pod-a shutdown -> releases lease -> pod-b takes over.
	cancelPodA()
	if err := <-podADone; err != nil {
		t.Fatalf("pod-a RunWithLease unexpected err: %v", err)
	}

	select {
	case <-podBStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for pod-b failover after pod-a stopped")
	}

	cancel()
	if err := <-podBDone; err != nil {
		t.Fatalf("pod-b RunWithLease unexpected err: %v", err)
	}

	if maxActive.Load() != 1 {
		t.Fatalf("mutual exclusion violated: max concurrent holders = %d", maxActive.Load())
	}
	if podARuns.Load() == 0 || podBRuns.Load() == 0 {
		t.Fatalf("expected both pods to lead in sequence: podA=%d podB=%d", podARuns.Load(), podBRuns.Load())
	}
}

func TestStartLeasedLoop_VoluntaryExitStopsLoopAndReleasesLease(t *testing.T) {
	mgr := NewMemoryManager()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	var runs atomic.Int32
	StartLeasedLoop(ctx, &wg, mgr, Config{
		LeaseName:     "worker:disabled-feature",
		HolderID:      "pod-a",
		TTL:           100 * time.Millisecond,
		RenewInterval: 25 * time.Millisecond,
		RetryInterval: 10 * time.Millisecond,
	}, func(_ context.Context) {
		runs.Add(1)
	})

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("StartLeasedLoop did not exit after voluntary worker return")
	}

	if runs.Load() != 1 {
		t.Fatalf("expected exactly 1 voluntary exit run, got %d", runs.Load())
	}
	if held, err := mgr.IsHeld(ctx, "worker:disabled-feature"); err != nil || held {
		t.Fatalf("expected lease released on voluntary exit, held=%v err=%v", held, err)
	}
}
