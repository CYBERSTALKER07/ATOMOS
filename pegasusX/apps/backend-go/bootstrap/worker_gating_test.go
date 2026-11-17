package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cloud.google.com/go/spanner"

	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/kafkautil"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/workerlease"
)

func setupStubAppConfig(t *testing.T, runMode string) *App {
	t.Helper()
	stubRuntimeConstructors(t)

	newRedisRuntimeAdapter = func(_ cache.RedisConfig) (redisRuntimeAdapter, error) {
		return &fakeRedisAdapter{}, nil
	}
	newKafkaRuntimePublisher = func(_ string, _ outbox.KafkaPublisherConfig) (kafkaRuntimePublisher, error) {
		return &fakeKafkaPublisher{}, nil
	}
	newKafkaRuntimeDLQWriter = func(_ string, _ string, _ kafkautil.ClientAuth) (kafkaRuntimeDLQWriter, error) {
		return &fakeKafkaDLQWriter{}, nil
	}
	newSpannerRuntimeClient = func(_ context.Context, _ string) (*spanner.Client, error) {
		return nil, errors.New("skip spanner in worker gating test")
	}

	cfg := testConfig()
	cfg.RunMode = runMode
	cfg.RequireInfraAdapters = true

	app, err := NewApp(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewApp(runMode=%q) failed: %v", runMode, err)
	}
	t.Cleanup(app.Close)
	return app
}

func TestNewApp_APIModeDoesNotRunSweepers(t *testing.T) {
	ctx := context.Background()
	app := setupStubAppConfig(t, RunModeAPI)

	if app.LeaseManager == nil {
		t.Fatal("expected LeaseManager to be initialized")
	}
	if app.SMSTransport == nil || app.SMSTransport.Breaker != app.OutboundCircuits.Notification {
		t.Fatal("expected SMSTransport circuit breaker wired to OutboundCircuits.Notification")
	}
	// Give any erroneously launched goroutines a moment to acquire leases.
	time.Sleep(30 * time.Millisecond)

	leases := []string{
		"sweeper:order_preorder",
		"sweeper:order_fiscal",
		"sweeper:order_stale_reservation",
		"sweeper:credit_overdue",
		"sweeper:order_negotiation",
		"sweeper:order_deferred_payment",
		"sweeper:global_products_match",
		"sweeper:inventory_holds",
	}
	for _, name := range leases {
		held, err := app.LeaseManager.IsHeld(ctx, name)
		if err != nil {
			t.Fatalf("IsHeld(%s) error: %v", name, err)
		}
		if held {
			t.Fatalf("API-only mode must not acquire sweeper lease %s", name)
		}
	}
}

func TestNewApp_WorkerModeAcquiresAndReleasesSweeperLeases(t *testing.T) {
	ctx := context.Background()
	app := setupStubAppConfig(t, RunModeWorker)

	expected := []string{
		"sweeper:order_preorder",
		"sweeper:order_fiscal",
		"sweeper:order_stale_reservation",
		"sweeper:credit_overdue",
	}

	deadline := time.Now().Add(2 * time.Second)
	for _, name := range expected {
		for {
			held, err := app.LeaseManager.IsHeld(ctx, name)
			if err != nil {
				t.Fatalf("IsHeld(%s) error: %v", name, err)
			}
			if held {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("timed out waiting for worker mode to acquire lease %s", name)
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	app.Close()

	for _, name := range expected {
		held, err := app.LeaseManager.IsHeld(ctx, name)
		if err != nil {
			t.Fatalf("IsHeld(%s) after Close error: %v", name, err)
		}
		if held {
			t.Fatalf("expected sweeper lease %s to be released on App.Close()", name)
		}
	}
}

func TestStartSpawningSweeper_SingleLeaderWithFailover(t *testing.T) {
	mgr := workerlease.NewMemoryManager()
	log := slog.Default()

	var (
		wgA      sync.WaitGroup
		wgB      sync.WaitGroup
		active   atomic.Int32
		maxAct   atomic.Int32
		startsA  atomic.Int32
		startsB  atomic.Int32
		startedA = make(chan struct{}, 1)
		startedB = make(chan struct{}, 1)
	)

	ctxA, cancelA := context.WithCancel(context.Background())
	defer cancelA()
	ctxB, cancelB := context.WithCancel(context.Background())
	defer cancelB()

	makeStarter := func(starts *atomic.Int32, signal chan struct{}) func(context.Context, *sync.WaitGroup) {
		return func(leaseCtx context.Context, localWg *sync.WaitGroup) {
			starts.Add(1)
			localWg.Add(1)
			cur := active.Add(1)
			for {
				prev := maxAct.Load()
				if cur <= prev || maxAct.CompareAndSwap(prev, cur) {
					break
				}
			}
			select {
			case signal <- struct{}{}:
			default:
			}
			go func() {
				defer localWg.Done()
				defer active.Add(-1)
				<-leaseCtx.Done()
			}()
		}
	}

	startSpawningSweeper(ctxA, &wgA, mgr, "sweeper:test_exclusive", "pod-a", log, makeStarter(&startsA, startedA))
	select {
	case <-startedA:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for pod-a sweeper to start")
	}

	startSpawningSweeper(ctxB, &wgB, mgr, "sweeper:test_exclusive", "pod-b", log, makeStarter(&startsB, startedB))
	// Give pod-b time to confirm it stays standby while pod-a leads.
	time.Sleep(50 * time.Millisecond)
	if startsB.Load() != 0 {
		t.Fatalf("standby pod-b started sweeper while pod-a held lease")
	}

	cancelA()
	wgA.Wait()

	// Note: startSpawningSweeper retries on workerlease.DefaultRetryInterval (5s),
	// so verify directly via WithLease or short retry that pod-b can immediately acquire after pod-a Close.
	acquired, err := workerlease.WithLease(ctxB, mgr, workerlease.Config{
		LeaseName: "sweeper:test_exclusive",
		HolderID:  "pod-b-probe",
		TTL:       30 * time.Second,
	}, func(_ context.Context) error {
		return nil
	})
	if err != nil || !acquired {
		t.Fatalf("expected pod-b-probe to acquire lease immediately after pod-a exit, acquired=%v err=%v", acquired, err)
	}

	cancelB()
	wgB.Wait()

	if maxAct.Load() != 1 {
		t.Fatalf("expected strict mutual exclusion (maxActive=1), got %d", maxAct.Load())
	}
}
