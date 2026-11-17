package bqexport

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/bigquery/v2"
	"google.golang.org/api/option"
)

const defaultMaxBatchRows = 500

// BigQueryConfig configures the Cloud BigQuery streaming JSON insert adapter.
type BigQueryConfig struct {
	ProjectID       string
	DatasetPrefix   string
	CredentialsFile string
	Endpoint        string // Non-production / test endpoint override only
	MaxBatchRows    int
	HTTPClient      *http.Client
}

// BigQuerySink streams canonical analytical rows via Cloud BigQuery tabledata.insertAll.
type BigQuerySink struct {
	svc           *bigquery.Service
	projectID     string
	datasetPrefix string
	maxBatchRows  int
	endpoint      string
}

// NewBigQuerySink constructs a fail-closed BigQuery streaming sink.
// In production (`auth.IsProduction() == true`), non-empty ProjectID and valid GCP
// credentials (explicit file or ADC) are mandatory, and custom Endpoint overrides are forbidden.
func NewBigQuerySink(ctx context.Context, cfg BigQueryConfig) (*BigQuerySink, error) {
	projectID := strings.TrimSpace(cfg.ProjectID)
	if projectID == "" {
		return nil, fmt.Errorf("%w: BIGQUERY_PROJECT_ID is empty", ErrBigQueryConfigRequired)
	}
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if auth.IsProduction() && endpoint != "" {
		return nil, fmt.Errorf("%w: custom BIGQUERY_ENDPOINT override is forbidden in production", ErrBigQueryConfigRequired)
	}

	opts := make([]option.ClientOption, 0, 4)
	if endpoint != "" {
		opts = append(opts, option.WithEndpoint(endpoint), option.WithoutAuthentication())
		if cfg.HTTPClient != nil {
			opts = append(opts, option.WithHTTPClient(cfg.HTTPClient))
		}
	} else {
		credsFile := strings.TrimSpace(cfg.CredentialsFile)
		if credsFile != "" {
			opts = append(opts, option.WithCredentialsFile(credsFile))
		} else {
			creds, err := google.FindDefaultCredentials(ctx, bigquery.BigqueryScope)
			if err != nil {
				return nil, fmt.Errorf("%w: GCP credentials unavailable: %v", ErrBigQueryConfigRequired, err)
			}
			opts = append(opts, option.WithCredentials(creds))
		}
		if cfg.HTTPClient != nil {
			opts = append(opts, option.WithHTTPClient(cfg.HTTPClient))
		}
	}

	svc, err := bigquery.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("%w: init bigquery service: %v", ErrBigQueryConfigRequired, err)
	}

	maxBatch := cfg.MaxBatchRows
	if maxBatch <= 0 {
		maxBatch = defaultMaxBatchRows
	}

	return &BigQuerySink{
		svc:           svc,
		projectID:     projectID,
		datasetPrefix: sanitizeDatasetPrefix(cfg.DatasetPrefix),
		maxBatchRows:  maxBatch,
		endpoint:      endpoint,
	}, nil
}

// WriteBatch groups rows by destination table and streams them idempotently via InsertAll.
func (s *BigQuerySink) WriteBatch(ctx context.Context, rows []Row) error {
	if s == nil || s.svc == nil {
		return ErrBigQueryConfigRequired
	}
	if len(rows) == 0 {
		return nil
	}

	type tableKey struct {
		dataset string
		table   string
	}
	grouped := make(map[tableKey][]Row)
	keys := make([]tableKey, 0, 4)

	for _, r := range rows {
		ds, tbl, err := splitDatasetTable(r.Table)
		if err != nil {
			return err
		}
		fullDS := s.resolveDataset(ds)
		k := tableKey{dataset: fullDS, table: tbl}
		if _, exists := grouped[k]; !exists {
			keys = append(keys, k)
		}
		grouped[k] = append(grouped[k], r)
	}

	sort.Slice(keys, func(i, j int) bool {
		if keys[i].dataset != keys[j].dataset {
			return keys[i].dataset < keys[j].dataset
		}
		return keys[i].table < keys[j].table
	})

	for _, k := range keys {
		tableRows := grouped[k]
		for start := 0; start < len(tableRows); start += s.maxBatchRows {
			end := start + s.maxBatchRows
			if end > len(tableRows) {
				end = len(tableRows)
			}
			if err := s.insertChunk(ctx, k.dataset, k.table, tableRows[start:end]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *BigQuerySink) insertChunk(ctx context.Context, datasetID, tableID string, chunk []Row) error {
	reqRows := make([]*bigquery.TableDataInsertAllRequestRows, 0, len(chunk))
	for _, r := range chunk {
		jsonMap := make(map[string]bigquery.JsonValue, len(r.Data))
		for k, v := range r.Data {
			jsonMap[k] = v
		}
		reqRows = append(reqRows, &bigquery.TableDataInsertAllRequestRows{
			InsertId: strings.TrimSpace(r.InsertID),
			Json:     jsonMap,
		})
	}

	req := &bigquery.TableDataInsertAllRequest{
		Kind:                "bigquery#tableDataInsertAllRequest",
		Rows:                reqRows,
		IgnoreUnknownValues: false,
		SkipInvalidRows:     false,
	}

	resp, err := s.svc.Tabledata.InsertAll(s.projectID, datasetID, tableID, req).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("bigquery insertAll %s.%s: %w", datasetID, tableID, err)
	}
	if resp != nil && len(resp.InsertErrors) > 0 {
		first := resp.InsertErrors[0]
		reason := ""
		if len(first.Errors) > 0 && first.Errors[0] != nil {
			reason = first.Errors[0].Reason + ": " + first.Errors[0].Message
		}
		return fmt.Errorf("%w: %s.%s row[%d] (%d error(s)): %s",
			ErrInsertRowsRejected, datasetID, tableID, first.Index, len(resp.InsertErrors), reason)
	}
	return nil
}

// Health verifies connectivity to the cell's raw BigQuery dataset.
func (s *BigQuerySink) Health(ctx context.Context) (HealthStatus, error) {
	status := HealthStatus{
		Adapter:   "bigquery",
		ProjectID: s.projectID,
		CheckedAt: time.Now().UTC(),
	}
	if s == nil || s.svc == nil {
		status.Healthy = false
		status.Detail = "bigquery service uninitialized"
		return status, ErrBigQueryConfigRequired
	}
	rawDataset := s.resolveDataset("raw")
	if _, err := s.svc.Datasets.Get(s.projectID, rawDataset).Context(ctx).Do(); err != nil {
		status.Healthy = false
		status.Detail = err.Error()
		return status, err
	}
	status.Healthy = true
	status.Detail = "dataset " + rawDataset + " reachable"
	return status, nil
}

func (s *BigQuerySink) resolveDataset(dataset string) string {
	if s.datasetPrefix == "" {
		return dataset
	}
	return s.datasetPrefix + "_" + dataset
}

func sanitizeDatasetPrefix(prefix string) string {
	trimmed := strings.TrimSpace(prefix)
	trimmed = strings.Trim(trimmed, "_-")
	if trimmed == "" {
		return ""
	}
	return strings.ReplaceAll(trimmed, "-", "_")
}

func splitDatasetTable(full string) (string, string, error) {
	parts := strings.Split(strings.TrimSpace(full), ".")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("%w: %q", ErrInvalidRowTable, full)
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}
