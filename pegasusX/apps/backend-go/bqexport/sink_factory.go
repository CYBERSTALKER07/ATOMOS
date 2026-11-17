package bqexport

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// NewSinkFromEnv constructs the configured analytical Sink from environment variables.
//
// Fail-closed contract:
//   - If BIGQUERY_PROJECT_ID is set: builds the real Cloud BigQuery adapter and fails closed if
//     GCP credentials cannot be initialized.
//   - Else if PEGASUSX_ENV=production: fails closed with ErrBigQueryConfigRequired (never falls back to JSONL).
//   - Else if BIGQUERY_LOCAL_SPOOL_DIR is set: builds JSONLSpoolSink.
//   - Else if BIGQUERY_SINK_ENABLED=true: builds JSONLSpoolSink in os.TempDir()/pegasusx-bq-spool.
//   - Else (unconfigured local/dev): returns (nil, nil).
func NewSinkFromEnv(ctx context.Context) (Sink, error) {
	projectID := strings.TrimSpace(os.Getenv("BIGQUERY_PROJECT_ID"))
	datasetPrefix := strings.TrimSpace(os.Getenv("BIGQUERY_DATASET_PREFIX"))
	credsFile := strings.TrimSpace(os.Getenv("BIGQUERY_CREDENTIALS_FILE"))
	if credsFile == "" {
		credsFile = strings.TrimSpace(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))
	}
	endpoint := strings.TrimSpace(os.Getenv("BIGQUERY_ENDPOINT"))

	if projectID != "" {
		return NewBigQuerySink(ctx, BigQueryConfig{
			ProjectID:       projectID,
			DatasetPrefix:   datasetPrefix,
			CredentialsFile: credsFile,
			Endpoint:        endpoint,
		})
	}

	if auth.IsProduction() {
		return nil, fmt.Errorf("%w: BIGQUERY_PROJECT_ID is required when PEGASUSX_ENV=production", ErrBigQueryConfigRequired)
	}

	if spoolDir := strings.TrimSpace(os.Getenv("BIGQUERY_LOCAL_SPOOL_DIR")); spoolDir != "" {
		return NewJSONLSpoolSink(spoolDir)
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("BIGQUERY_SINK_ENABLED")), "true") {
		return NewJSONLSpoolSink(filepath.Join(os.TempDir(), "pegasusx-bq-spool"))
	}
	return nil, nil
}
