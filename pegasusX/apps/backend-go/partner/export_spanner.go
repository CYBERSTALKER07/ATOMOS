package partner

import (
	"context"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// SpannerExportRepository persists PartnerExportJobs.
type SpannerExportRepository struct {
	client *spanner.Client
}

func NewSpannerExportRepository(client *spanner.Client) *SpannerExportRepository {
	return &SpannerExportRepository{client: client}
}

func (r *SpannerExportRepository) InsertJob(ctx context.Context, j ExportJob) error {
	cols := map[string]any{
		"JobId":      j.JobID,
		"TenantType": j.TenantType,
		"TenantId":   j.TenantID,
		"Resource":   j.Resource,
		"Format":     j.Format,
		"Status":     j.Status,
		"RowCount":   j.RowCount,
		"CreatedAt":  spanner.CommitTimestamp,
	}
	if j.FromDate != nil {
		cols["FromDate"] = civil.DateOf(*j.FromDate)
	}
	if j.ToDate != nil {
		cols["ToDate"] = civil.DateOf(*j.ToDate)
	}
	if j.ObjectPath != "" {
		cols["ObjectPath"] = j.ObjectPath
	}
	if j.Error != "" {
		cols["Error"] = j.Error
	}
	if j.SftpStatus != "" {
		cols["SftpStatus"] = j.SftpStatus
	}
	_, err := r.client.Apply(ctx, []*spanner.Mutation{spanner.InsertMap("PartnerExportJobs", cols)})
	return err
}

func (r *SpannerExportRepository) GetJob(ctx context.Context, jobID string) (ExportJob, bool, error) {
	row, err := r.client.Single().ReadRow(ctx, "PartnerExportJobs", spanner.Key{jobID},
		[]string{"JobId", "TenantType", "TenantId", "Resource", "Format", "Status",
			"FromDate", "ToDate", "ObjectPath", "RowCount", "Error", "SftpStatus", "CreatedAt", "FinishedAt"})
	if err != nil {
		if isSpannerNotFound(err) {
			return ExportJob{}, false, nil
		}
		return ExportJob{}, false, err
	}
	j, err := scanExportJob(row)
	if err != nil {
		return ExportJob{}, false, err
	}
	return j, true, nil
}

func (r *SpannerExportRepository) ListJobs(ctx context.Context, tenantType, tenantID string, limit int) ([]ExportJob, error) {
	if limit <= 0 {
		limit = 50
	}
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT JobId, TenantType, TenantId, Resource, Format, Status,
			FromDate, ToDate, ObjectPath, RowCount, Error, SftpStatus, CreatedAt, FinishedAt
			FROM PartnerExportJobs
			WHERE TenantType = @tt AND TenantId = @tid
			ORDER BY CreatedAt DESC LIMIT @lim`,
		Params: map[string]any{"tt": tenantType, "tid": tenantID, "lim": int64(limit)},
	})
	defer iter.Stop()
	return collectExportJobs(iter)
}

func (r *SpannerExportRepository) ListPending(ctx context.Context, limit int) ([]ExportJob, error) {
	if limit <= 0 {
		limit = 20
	}
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT JobId, TenantType, TenantId, Resource, Format, Status,
			FromDate, ToDate, ObjectPath, RowCount, Error, SftpStatus, CreatedAt, FinishedAt
			FROM PartnerExportJobs
			WHERE Status = @st
			ORDER BY CreatedAt ASC LIMIT @lim`,
		Params: map[string]any{"st": ExportStatusPending, "lim": int64(limit)},
	})
	defer iter.Stop()
	return collectExportJobs(iter)
}

func (r *SpannerExportRepository) UpdateJob(ctx context.Context, j ExportJob) error {
	cols := map[string]any{
		"JobId":      j.JobID,
		"TenantType": j.TenantType,
		"TenantId":   j.TenantID,
		"Resource":   j.Resource,
		"Format":     j.Format,
		"Status":     j.Status,
		"RowCount":   j.RowCount,
		"ObjectPath": nullableStr(j.ObjectPath),
		"Error":      nullableStr(j.Error),
		"SftpStatus": nullableStr(j.SftpStatus),
		"CreatedAt":  j.CreatedAt,
	}
	if j.FromDate != nil {
		cols["FromDate"] = civil.DateOf(*j.FromDate)
	}
	if j.ToDate != nil {
		cols["ToDate"] = civil.DateOf(*j.ToDate)
	}
	if j.FinishedAt != nil {
		cols["FinishedAt"] = *j.FinishedAt
	}
	_, err := r.client.Apply(ctx, []*spanner.Mutation{spanner.InsertOrUpdateMap("PartnerExportJobs", cols)})
	return err
}

func collectExportJobs(iter *spanner.RowIterator) ([]ExportJob, error) {
	var out []ExportJob
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		j, err := scanExportJob(row)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, nil
}

func scanExportJob(row *spanner.Row) (ExportJob, error) {
	var j ExportJob
	var from, to spanner.NullDate
	var path, errMsg, sftp spanner.NullString
	var finished spanner.NullTime
	var created time.Time
	if err := row.Columns(
		&j.JobID, &j.TenantType, &j.TenantID, &j.Resource, &j.Format, &j.Status,
		&from, &to, &path, &j.RowCount, &errMsg, &sftp, &created, &finished,
	); err != nil {
		return ExportJob{}, err
	}
	j.CreatedAt = created.UTC()
	if from.Valid {
		t := from.Date.In(time.UTC)
		j.FromDate = &t
	}
	if to.Valid {
		t := to.Date.In(time.UTC)
		j.ToDate = &t
	}
	if path.Valid {
		j.ObjectPath = path.StringVal
	}
	if errMsg.Valid {
		j.Error = errMsg.StringVal
	}
	if sftp.Valid {
		j.SftpStatus = sftp.StringVal
	}
	if finished.Valid {
		t := finished.Time.UTC()
		j.FinishedAt = &t
	}
	return j, nil
}

// SpannerSftpConfigRepository persists PartnerSftpConfigs.
type SpannerSftpConfigRepository struct {
	client *spanner.Client
}

func NewSpannerSftpConfigRepository(client *spanner.Client) *SpannerSftpConfigRepository {
	return &SpannerSftpConfigRepository{client: client}
}

func (r *SpannerSftpConfigRepository) Upsert(ctx context.Context, c SftpConfig) error {
	normalizeSftpDirs(&c)
	_, err := r.client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("PartnerSftpConfigs", map[string]any{
			"TenantType":  c.TenantType,
			"TenantId":    c.TenantID,
			"Host":        c.Host,
			"Port":        c.Port,
			"Username":    c.Username,
			"SecretRef":   c.SecretRef,
			"RemoteDir":   c.RemoteDir,
			"IsActive":    c.IsActive,
			"InboundDir":  c.InboundDir,
			"OutboundDir": c.OutboundDir,
			"ArchiveDir":  c.ArchiveDir,
			"EdiEnabled":  c.EdiEnabled,
			"UpdatedAt":   spanner.CommitTimestamp,
		}),
	})
	return err
}

func (r *SpannerSftpConfigRepository) Get(ctx context.Context, tenantType, tenantID string) (SftpConfig, bool, error) {
	row, err := r.client.Single().ReadRow(ctx, "PartnerSftpConfigs", spanner.Key{tenantType, tenantID},
		[]string{"TenantType", "TenantId", "Host", "Port", "Username", "SecretRef", "RemoteDir", "IsActive",
			"InboundDir", "OutboundDir", "ArchiveDir", "EdiEnabled", "UpdatedAt"})
	if err != nil {
		if isSpannerNotFound(err) {
			return SftpConfig{}, false, nil
		}
		return SftpConfig{}, false, err
	}
	c, err := scanSftpConfig(row)
	if err != nil {
		return SftpConfig{}, false, err
	}
	return c, true, nil
}

func nullableStr(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func isSpannerNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "not found") || strings.Contains(msg, "NotFound") || strings.Contains(msg, "code = NotFound")
}
