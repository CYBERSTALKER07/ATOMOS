package partner

import (
	"context"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// SpannerEdiDocumentRepository persists PartnerEdiDocuments.
type SpannerEdiDocumentRepository struct {
	client *spanner.Client
}

func NewSpannerEdiDocumentRepository(client *spanner.Client) *SpannerEdiDocumentRepository {
	return &SpannerEdiDocumentRepository{client: client}
}

func (r *SpannerEdiDocumentRepository) Insert(ctx context.Context, d EdiDocument) error {
	_, err := r.client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("PartnerEdiDocuments", map[string]any{
			"DocumentId":    d.DocumentID,
			"TenantType":    d.TenantType,
			"TenantId":      d.TenantID,
			"Direction":     d.Direction,
			"DocType":       d.DocType,
			"ExternalDocId": d.ExternalDocID,
			"OrderId":       nullableStr(d.OrderID),
			"Status":        d.Status,
			"ObjectPath":    nullableStr(d.ObjectPath),
			"RemoteName":    nullableStr(d.RemoteName),
			"Error":         nullableStr(d.Error),
			"PayloadHash":   nullableStr(d.PayloadHash),
			"CreatedAt":     spanner.CommitTimestamp,
			"FinishedAt":    nullableTime(d.FinishedAt),
		}),
	})
	return err
}

func (r *SpannerEdiDocumentRepository) Get(ctx context.Context, documentID string) (EdiDocument, bool, error) {
	row, err := r.client.Single().ReadRow(ctx, "PartnerEdiDocuments", spanner.Key{documentID}, ediDocCols)
	if err != nil {
		if isSpannerNotFound(err) {
			return EdiDocument{}, false, nil
		}
		return EdiDocument{}, false, err
	}
	return scanEdiDoc(row)
}

func (r *SpannerEdiDocumentRepository) GetByExternal(ctx context.Context, tenantType, tenantID, direction, docType, externalDocID string) (EdiDocument, bool, error) {
	stmt := spanner.Statement{
		SQL: `SELECT DocumentId, TenantType, TenantId, Direction, DocType, ExternalDocId, OrderId, Status,
			ObjectPath, RemoteName, Error, PayloadHash, CreatedAt, FinishedAt
			FROM PartnerEdiDocuments
			WHERE TenantType=@tt AND TenantId=@tid AND Direction=@dir AND DocType=@dt AND ExternalDocId=@ext
			LIMIT 1`,
		Params: map[string]any{
			"tt": tenantType, "tid": tenantID, "dir": direction, "dt": docType, "ext": externalDocID,
		},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return EdiDocument{}, false, nil
	}
	if err != nil {
		return EdiDocument{}, false, err
	}
	return scanEdiDoc(row)
}

func (r *SpannerEdiDocumentRepository) Update(ctx context.Context, d EdiDocument) error {
	_, err := r.client.Apply(ctx, []*spanner.Mutation{
		spanner.UpdateMap("PartnerEdiDocuments", map[string]any{
			"DocumentId":    d.DocumentID,
			"TenantType":    d.TenantType,
			"TenantId":      d.TenantID,
			"Direction":     d.Direction,
			"DocType":       d.DocType,
			"ExternalDocId": d.ExternalDocID,
			"OrderId":       nullableStr(d.OrderID),
			"Status":        d.Status,
			"ObjectPath":    nullableStr(d.ObjectPath),
			"RemoteName":    nullableStr(d.RemoteName),
			"Error":         nullableStr(d.Error),
			"PayloadHash":   nullableStr(d.PayloadHash),
			"FinishedAt":    nullableTime(d.FinishedAt),
		}),
	})
	return err
}

func (r *SpannerEdiDocumentRepository) ListByTenant(ctx context.Context, tenantType, tenantID string, limit int) ([]EdiDocument, error) {
	if limit <= 0 {
		limit = 50
	}
	stmt := spanner.Statement{
		SQL: `SELECT DocumentId, TenantType, TenantId, Direction, DocType, ExternalDocId, OrderId, Status,
			ObjectPath, RemoteName, Error, PayloadHash, CreatedAt, FinishedAt
			FROM PartnerEdiDocuments
			WHERE TenantType=@tt AND TenantId=@tid
			ORDER BY CreatedAt DESC LIMIT @lim`,
		Params: map[string]any{"tt": tenantType, "tid": tenantID, "lim": int64(limit)},
	}
	return queryEdiDocs(ctx, r.client, stmt)
}

func (r *SpannerEdiDocumentRepository) ListPendingOutbound(ctx context.Context, limit int) ([]EdiDocument, error) {
	if limit <= 0 {
		limit = 20
	}
	stmt := spanner.Statement{
		SQL: `SELECT DocumentId, TenantType, TenantId, Direction, DocType, ExternalDocId, OrderId, Status,
			ObjectPath, RemoteName, Error, PayloadHash, CreatedAt, FinishedAt
			FROM PartnerEdiDocuments
			WHERE Status=@st AND Direction=@dir
			ORDER BY CreatedAt LIMIT @lim`,
		Params: map[string]any{"st": EdiStatusReceived, "dir": EdiDirectionOut, "lim": int64(limit)},
	}
	return queryEdiDocs(ctx, r.client, stmt)
}

func (r *SpannerSftpConfigRepository) ListEdiEnabled(ctx context.Context, limit int) ([]SftpConfig, error) {
	if limit <= 0 {
		limit = 50
	}
	stmt := spanner.Statement{
		SQL: `SELECT TenantType, TenantId, Host, Port, Username, SecretRef, RemoteDir, IsActive,
			InboundDir, OutboundDir, ArchiveDir, EdiEnabled, HostKeySha256, UpdatedAt
			FROM PartnerSftpConfigs
			WHERE IsActive=TRUE AND EdiEnabled=TRUE
			LIMIT @lim`,
		Params: map[string]any{"lim": int64(limit)},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	out := make([]SftpConfig, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		c, err := scanSftpConfig(row)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

var ediDocCols = []string{
	"DocumentId", "TenantType", "TenantId", "Direction", "DocType", "ExternalDocId", "OrderId", "Status",
	"ObjectPath", "RemoteName", "Error", "PayloadHash", "CreatedAt", "FinishedAt",
}

func scanEdiDoc(row *spanner.Row) (EdiDocument, bool, error) {
	var d EdiDocument
	var orderID, objectPath, remoteName, errMsg, hash spanner.NullString
	var created time.Time
	var finished spanner.NullTime
	if err := row.Columns(
		&d.DocumentID, &d.TenantType, &d.TenantID, &d.Direction, &d.DocType, &d.ExternalDocID,
		&orderID, &d.Status, &objectPath, &remoteName, &errMsg, &hash, &created, &finished,
	); err != nil {
		return EdiDocument{}, false, err
	}
	d.OrderID = orderID.StringVal
	d.ObjectPath = objectPath.StringVal
	d.RemoteName = remoteName.StringVal
	d.Error = errMsg.StringVal
	d.PayloadHash = hash.StringVal
	d.CreatedAt = created.UTC()
	if finished.Valid {
		t := finished.Time.UTC()
		d.FinishedAt = &t
	}
	return d, true, nil
}

func queryEdiDocs(ctx context.Context, client *spanner.Client, stmt spanner.Statement) ([]EdiDocument, error) {
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	out := make([]EdiDocument, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		d, _, err := scanEdiDoc(row)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

func nullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}

func scanSftpConfig(row *spanner.Row) (SftpConfig, error) {
	var c SftpConfig
	var inbound, outbound, archive, hostKey spanner.NullString
	var ediEnabled spanner.NullBool
	var updated time.Time
	if err := row.Columns(
		&c.TenantType, &c.TenantID, &c.Host, &c.Port, &c.Username, &c.SecretRef, &c.RemoteDir, &c.IsActive,
		&inbound, &outbound, &archive, &ediEnabled, &hostKey, &updated,
	); err != nil {
		return SftpConfig{}, err
	}
	c.InboundDir = inbound.StringVal
	c.OutboundDir = outbound.StringVal
	c.ArchiveDir = archive.StringVal
	c.EdiEnabled = ediEnabled.Valid && ediEnabled.Bool
	c.HostKeySHA256 = hostKey.StringVal
	c.UpdatedAt = updated.UTC()
	normalizeSftpDirs(&c)
	return c, nil
}

func normalizeSftpDirs(c *SftpConfig) {
	if c == nil {
		return
	}
	if strings.TrimSpace(c.InboundDir) == "" {
		c.InboundDir = "inbound"
	}
	if strings.TrimSpace(c.OutboundDir) == "" {
		c.OutboundDir = "outbound"
	}
	if strings.TrimSpace(c.ArchiveDir) == "" {
		c.ArchiveDir = "archive"
	}
}
