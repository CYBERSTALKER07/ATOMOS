package predictivepush

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
)

// Run is the main entry point for the Predictive Push agent.
// It is expected to be called by a cron job (e.g., daily at 1 AM).
func Run(ctx context.Context, client *spanner.Client) error {
	slog.Info("starting predictive push agent analysis")

	analyzer := NewAnalyzer(client)
	locator := NewLocator(client)
	allocator := NewAllocator(client, locator)
	auditor := NewAuditor(client)

	// We predict for 24-48 hours ahead to give payloaders time.
	// For this example, let's target tomorrow.
	targetDate := time.Now().AddDate(0, 0, 1)

	// 1. Analyze historical patterns to predict tomorrow's orders
	events, err := analyzer.Analyze(ctx, targetDate)
	if err != nil {
		slog.Error("failed to analyze historical orders", "err", err)
		return err
	}

	if len(events) == 0 {
		slog.Info("no predictive push events identified for target date", "date", targetDate.Format("2006-01-02"))
		return nil
	}

	slog.Info("identified predictive push events", "count", len(events), "date", targetDate.Format("2006-01-02"))

	shadow := strings.EqualFold(strings.TrimSpace(os.Getenv("PLANNING_BRAIN_SHADOW")), "true")
	if shadow {
		slog.Info("planning brain shadow mode: baseline projection only")
		return allocator.writeDemandBaselines(ctx, events)
	}

	// 2. Proactively allocate stock (generate ReplenishmentInsights)
	if err := allocator.Allocate(ctx, events); err != nil {
		slog.Error("failed to allocate proactive stock", "err", err)
		return err
	}

	// 3. Log predictions for auditing and model tuning
	if err := auditor.LogPredictions(ctx, events); err != nil {
		slog.Error("failed to log AI predictions", "err", err)
		return err
	}

	slog.Info("successfully completed predictive push cycle")
	return nil
}
