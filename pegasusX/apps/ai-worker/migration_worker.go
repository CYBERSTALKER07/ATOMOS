package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"

	"github.com/pegasusx/pegasusx/apps/backend-go/migration"
	"github.com/pegasusx/pegasusx/apps/backend-go/migration/parser"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplier"
)

type bundleUploadedEnvelope struct {
	Type       string `json:"type"`
	SupplierID string `json:"supplier_id"`
	ProjectID  string `json:"project_id"`
	BundleID   string `json:"bundle_id"`
	HomeCell   string `json:"home_cell,omitempty"`
}

type migrationBundleProcessor struct {
	repo       migration.Repository
	readObject migration.ArtifactBytesReader
	mapper     *migration.AIMapper
	logger     *slog.Logger
}

func newMigrationBundleProcessor(
	spannerClient *spanner.Client,
	opener *supplier.ImportObjectOpener,
	logger *slog.Logger,
) *migrationBundleProcessor {
	if spannerClient == nil {
		return nil
	}
	if logger == nil {
		logger = slog.Default()
	}
	var reader migration.ArtifactBytesReader
	if opener != nil {
		reader = func(ctx context.Context, gcsPath string) ([]byte, error) {
			return opener.ReadObject(ctx, gcsPath)
		}
	} else {
		reader = migration.ReadArtifactBytes
	}
	return &migrationBundleProcessor{
		repo:       migration.NewSpannerRepository(spannerClient),
		readObject: reader,
		mapper:     migration.NewAIMapper(migration.GeminiMapperConfigFromEnv()),
		logger:     logger,
	}
}

func detectEnvelopeType(raw []byte) string {
	var env struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return ""
	}
	return strings.TrimSpace(env.Type)
}

func (p *migrationBundleProcessor) ProcessEvent(ctx context.Context, payload []byte) (terminalErr error, transientErr error) {
	if p == nil || p.repo == nil {
		return nil, errors.New("migration worker unavailable")
	}
	var env bundleUploadedEnvelope
	if err := json.Unmarshal(payload, &env); err != nil {
		return fmt.Errorf("malformed migration bundle event: %w", err), nil
	}
	if strings.TrimSpace(env.SupplierID) == "" || strings.TrimSpace(env.ProjectID) == "" || strings.TrimSpace(env.BundleID) == "" {
		return errors.New("missing supplier_id/project_id/bundle_id in MIGRATION_BUNDLE_UPLOADED event"), nil
	}

	if bundle, bErr := p.repo.GetBundle(ctx, env.SupplierID, env.ProjectID, env.BundleID); bErr == nil {
		if bundle.Status == migration.BundleStatusSorted || bundle.Status == migration.BundleStatusNeedsReview {
			if existing, sErr := p.repo.ListSessionsByBundle(ctx, env.SupplierID, env.ProjectID, env.BundleID); sErr == nil && len(existing) > 0 {
				p.logger.Info("bundle already sorted synchronously; skipping duplicate MIGRATION_BUNDLE_UPLOADED event",
					"supplier_id", env.SupplierID,
					"project_id", env.ProjectID,
					"bundle_id", env.BundleID,
					"bundle_status", bundle.Status,
					"sessions", len(existing),
				)
				return nil, nil
			}
		}
	}

	decision, err := migration.ProcessBundleSortAndStage(
		ctx,
		p.repo,
		p.readObject,
		env.SupplierID,
		env.ProjectID,
		env.BundleID,
		nil,
		uuid.NewString,
	)
	if err != nil {
		if isTerminalBundleArtifactError(err) {
			return err, nil
		}
		return nil, err
	}

	homeCell := strings.TrimSpace(env.HomeCell)
	if homeCell == "" {
		homeCell = "cell-uz"
	}
	docs, mapErr := migration.ProposeMappingsForBundle(
		ctx,
		p.repo,
		p.mapper,
		env.SupplierID,
		homeCell,
		env.ProjectID,
		env.BundleID,
	)
	if mapErr != nil {
		return nil, mapErr
	}

	p.logger.Info("migration bundle sorted and mapped",
		"supplier_id", env.SupplierID,
		"project_id", env.ProjectID,
		"bundle_id", env.BundleID,
		"bundle_status", decision.BundleStatus,
		"sessions", len(decision.Sessions),
		"mapped_docs", len(docs),
	)
	return nil, nil
}

func isTerminalBundleArtifactError(err error) bool {
	return errors.Is(err, parser.ErrPasswordProtectedWorkbook) ||
		errors.Is(err, parser.ErrFileFormatMismatch) ||
		errors.Is(err, parser.ErrMalformedArtifact) ||
		errors.Is(err, migration.ErrArtifactDigestMismatch) ||
		errors.Is(err, migration.ErrInvalidInput) ||
		errors.Is(err, migration.ErrNotFound)
}
