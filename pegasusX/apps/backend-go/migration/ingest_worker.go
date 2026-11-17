package migration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/migration/parser"
)

// ArtifactBytesReader abstracts reading raw artifact bytes from GCS or local storage.
type ArtifactBytesReader func(ctx context.Context, gcsPath string) ([]byte, error)

// ErrArtifactDigestMismatch indicates downloaded artifact bytes do not match declared content_sha256.
var ErrArtifactDigestMismatch = errors.New("artifact_sha256_mismatch")

type loadedArtifactPayload struct {
	art      Artifact
	raw      []byte
	encoding string
	streams  []parser.ParsedStream
}

// ProcessBundleSortAndStage downloads uploaded artifacts for a bundle, sniffs & parses all formats,
// runs Universal Auto-Sorting + composite sheet splitting, persists child sessions, and streams
// every row into SupplierImportStagedRows in 5,000-row chunks with zero 10k row ceiling.
func ProcessBundleSortAndStage(
	ctx context.Context,
	repo Repository,
	reader ArtifactBytesReader,
	supplierID, projectID, bundleID string,
	overrides *BundleSortOverrideRequest,
	newID func() string,
) (SortDecision, error) {
	if repo == nil {
		return SortDecision{}, fmt.Errorf("migration repository is required")
	}
	if reader == nil {
		reader = ReadArtifactBytes
	}
	if newID == nil {
		newID = uuid.NewString
	}

	bundle, err := repo.GetBundle(ctx, supplierID, projectID, bundleID)
	if err != nil {
		return SortDecision{}, err
	}
	artifacts, err := repo.ListArtifactsByBundle(ctx, supplierID, projectID, bundleID)
	if err != nil {
		return SortDecision{}, err
	}
	if len(artifacts) == 0 {
		return SortDecision{}, fmt.Errorf("%w: bundle has no artifacts", ErrInvalidInput)
	}
	existingSessions, err := repo.ListSessionsByBundle(ctx, supplierID, projectID, bundleID)
	if err != nil {
		return SortDecision{}, err
	}
	// Duplicate un-overridden deliveries (for instance Kafka MIGRATION_BUNDLE_UPLOADED
	// racing with ?sync=true) must be idempotent once the bundle is sorted so they cannot
	// overwrite confirmed mappings, dry-run reports, or applied sessions.
	if overrides == nil && (bundle.Status == BundleStatusSorted || bundle.Status == BundleStatusNeedsReview) && len(existingSessions) > 0 {
		return SortDecision{
			BundleStatus: bundle.Status,
			Sessions:     existingSessions,
		}, nil
	}

	encodingOverrides := extractArtifactEncodingOverrides(artifacts, overrides)
	payloads := make([]loadedArtifactPayload, 0, len(artifacts))
	sortInputs := make([]ArtifactStreamsInput, 0, len(artifacts))

	for _, art := range artifacts {
		raw, err := reader(ctx, art.GCSPath)
		if err != nil {
			_ = repo.MarkBundleFailed(ctx, supplierID, projectID, bundleID, err.Error())
			return SortDecision{}, err
		}
		if expected := strings.ToLower(strings.TrimSpace(art.ContentSHA256)); expected != "" {
			sum := sha256.Sum256(raw)
			actual := hex.EncodeToString(sum[:])
			if actual != expected {
				mismatchErr := fmt.Errorf("%w: %s expected %s got %s", ErrArtifactDigestMismatch, art.FileName, expected, actual)
				_ = repo.MarkBundleFailed(ctx, supplierID, projectID, bundleID, mismatchErr.Error())
				return SortDecision{}, mismatchErr
			}
		} else {
			sum := sha256.Sum256(raw)
			art.ContentSHA256 = hex.EncodeToString(sum[:])
		}

		enc := strings.TrimSpace(art.Encoding)
		if ovEnc, ok := encodingOverrides[art.ArtifactID]; ok && ovEnc != "" {
			enc = ovEnc
		}

		streams, _, parseErr := parser.ParseArtifact(art.FileName, raw, enc, nil)
		if parseErr != nil {
			_ = repo.MarkBundleFailed(ctx, supplierID, projectID, bundleID, parseErr.Error())
			return SortDecision{}, parseErr
		}
		if len(streams) > 0 && art.Encoding == "" {
			art.Encoding = streams[0].Encoding
		}

		payloads = append(payloads, loadedArtifactPayload{
			art:      art,
			raw:      raw,
			encoding: enc,
			streams:  streams,
		})
		sortInputs = append(sortInputs, ArtifactStreamsInput{
			Artifact: art,
			Streams:  streams,
		})
	}

	decision, err := ClassifyAndSplitBundle(sortInputs, overrides, existingSessions, newID)
	if err != nil {
		return SortDecision{}, err
	}

	persistedSessions, err := repo.UpsertSortedBundleSessions(
		ctx,
		supplierID,
		projectID,
		bundleID,
		decision.BundleStatus,
		decision.SummaryJSON,
		decision.ArtifactStreams,
		decision.Sessions,
	)
	if err != nil {
		return SortDecision{}, err
	}
	decision.Sessions = persistedSessions

	// Group child sessions by (artifact_id, stream_key) so Pass 2 streams rows once per artifact.
	sessionsByArtStream := make(map[string][]Session)
	for _, s := range persistedSessions {
		k := s.ArtifactID + "::" + s.StreamKey
		sessionsByArtStream[k] = append(sessionsByArtStream[k], s)
	}

	for _, p := range payloads {
		hasTargetForArt := false
		for _, st := range p.streams {
			if len(sessionsByArtStream[p.art.ArtifactID+"::"+st.StreamName]) > 0 {
				hasTargetForArt = true
				break
			}
		}
		if !hasTargetForArt {
			continue
		}

		// Track per-session sequential row index (important when composite order streams filter open vs historical rows).
		nextRowIdx := make(map[string]int64)
		_, _, err := parser.ParseArtifact(p.art.FileName, p.raw, p.encoding, func(streamName string, headers []string, chunk []parser.ParsedRow) error {
			targets := sessionsByArtStream[p.art.ArtifactID+"::"+streamName]
			if len(targets) == 0 || len(chunk) == 0 {
				return nil
			}
			needsOrderStatusFilter := streamHasSplitOrderSiblings(targets)

			for _, sess := range targets {
				staged := make([]StagedRow, 0, len(chunk))
				for _, row := range chunk {
					if needsOrderStatusFilter && !SessionSplitRowMatches(sess.StreamSplitKey, headers, row.Fields) {
						continue
					}
					rawJSON, marshalErr := json.Marshal(row.Fields)
					if marshalErr != nil {
						return fmt.Errorf("marshal staged row %d: %w", row.RowIndex, marshalErr)
					}
					idx := nextRowIdx[sess.SessionID]
					nextRowIdx[sess.SessionID] = idx + 1
					staged = append(staged, StagedRow{
						SupplierID:     supplierID,
						SessionID:      sess.SessionID,
						RowIndex:       idx,
						RawData:        rawJSON,
						StreamSplitKey: sess.StreamSplitKey,
					})
				}
				if len(staged) > 0 {
					if _, saveErr := repo.SaveStagedRowsChunk(ctx, supplierID, sess.SessionID, staged); saveErr != nil {
						return saveErr
					}
				}
			}
			return nil
		})
		if err != nil {
			return SortDecision{}, err
		}
		// Keep reported session TotalRows synchronized after filtered split staging.
		for i := range decision.Sessions {
			s := &decision.Sessions[i]
			if count, ok := nextRowIdx[s.SessionID]; ok {
				s.TotalRows = count
			}
		}
	}

	return decision, nil
}

func extractArtifactEncodingOverrides(artifacts []Artifact, overrides *BundleSortOverrideRequest) map[string]string {
	out := make(map[string]string, len(artifacts))
	if overrides == nil {
		return out
	}
	for _, ov := range overrides.Overrides {
		if enc := strings.TrimSpace(ov.Encoding); enc != "" {
			out[ov.ArtifactID] = enc
		}
	}
	return out
}

func streamHasSplitOrderSiblings(sessions []Session) bool {
	hasOpenSplit := false
	hasHistSplit := false
	for _, s := range sessions {
		switch s.StreamSplitKey {
		case "orders_open":
			hasOpenSplit = true
		case "orders_historical":
			hasHistSplit = true
		}
	}
	return hasOpenSplit && hasHistSplit
}

// SessionSplitRowMatches decides whether a row belongs to a status-filtered child split session
// when a single sheet mixes open and historical orders.
func SessionSplitRowMatches(splitKey string, headers []string, fields map[string]any) bool {
	if splitKey != "orders_open" && splitKey != "orders_historical" {
		return true
	}
	statusCols := findStatusHeaderNames(headers)
	if len(statusCols) == 0 {
		// CommerceML uses "status" and "operation".
		statusCols = []string{"status", "operation"}
	}
	hasOpen, _ := inspectOrderStatuses([]parser.ParsedRow{{Fields: fields}}, statusCols)
	if splitKey == "orders_open" {
		return hasOpen
	}
	return !hasOpen
}
