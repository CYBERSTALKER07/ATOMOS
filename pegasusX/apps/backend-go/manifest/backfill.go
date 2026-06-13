package manifest

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

var defaultBackfillStates = []string{"SEALED", "DISPATCHED", "LOADING", "DRAFT"}

// BackfillOptions controls the route-geometry backfill sweep.
type BackfillOptions struct {
	Limit  int
	DryRun bool
	States []string
}

// BackfillResult summarizes one backfill run.
type BackfillResult struct {
	Candidates int  `json:"candidates"`
	Updated    int  `json:"updated"`
	Failed     int  `json:"failed"`
	DryRun     bool `json:"dry_run"`
}

// BackfillRouteGeometry fills EncodedRoutePolyline for manifests that are missing it.
func (s *Store) BackfillRouteGeometry(ctx context.Context, opts BackfillOptions) (BackfillResult, error) {
	if s == nil || s.client == nil {
		return BackfillResult{}, fmt.Errorf("manifest store: nil client")
	}
	opts = normalizeBackfillOptions(opts)

	manifestIDs, err := s.listManifestIDsMissingGeometry(ctx, opts.States, opts.Limit)
	if err != nil {
		return BackfillResult{}, err
	}

	result := BackfillResult{
		Candidates: len(manifestIDs),
		DryRun:     opts.DryRun,
	}
	if opts.DryRun || len(manifestIDs) == 0 {
		return result, nil
	}

	for _, manifestID := range manifestIDs {
		if err := s.PersistRouteGeometryForManifest(ctx, manifestID, "backfill"); err != nil {
			result.Failed++
			slog.WarnContext(ctx, "route geometry backfill failed",
				"manifest_id", manifestID,
				"err", err,
			)
			continue
		}
		result.Updated++
	}
	return result, nil
}

// PersistDispatchPreviewGeometries best-effort persists route overlays after dispatch commit.
func (s *Store) PersistDispatchPreviewGeometries(ctx context.Context, manifestIDs []string) {
	if s == nil {
		return
	}
	for _, manifestID := range manifestIDs {
		manifestID = strings.TrimSpace(manifestID)
		if manifestID == "" {
			continue
		}
		if err := s.PersistRouteGeometryForManifest(ctx, manifestID, "dispatch_preview"); err != nil {
			slog.WarnContext(ctx, "dispatch preview route geometry failed",
				"manifest_id", manifestID,
				"err", err,
			)
		}
	}
}

func normalizeBackfillOptions(opts BackfillOptions) BackfillOptions {
	if opts.Limit <= 0 {
		opts.Limit = 100
	}
	if len(opts.States) == 0 {
		opts.States = append([]string(nil), defaultBackfillStates...)
	}
	return opts
}

func (s *Store) listManifestIDsMissingGeometry(ctx context.Context, states []string, limit int) ([]string, error) {
	stmt := spanner.Statement{
		SQL: `SELECT ManifestId
		      FROM SupplierTruckManifests
		      WHERE State IN UNNEST(@states)
		        AND (EncodedRoutePolyline IS NULL OR EncodedRoutePolyline = '')
		      ORDER BY UpdatedAt DESC
		      LIMIT @limit`,
		Params: map[string]any{
			"states": states,
			"limit":  int64(limit),
		},
	}
	iter := s.client.Single().WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).Query(ctx, stmt)
	defer iter.Stop()

	manifestIDs := make([]string, 0, limit)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list manifests missing geometry: %w", err)
		}
		var manifestID string
		if err := row.Columns(&manifestID); err != nil {
			return nil, err
		}
		manifestIDs = append(manifestIDs, manifestID)
	}
	return manifestIDs, nil
}
