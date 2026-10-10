package factory

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

type factoryExceptionBackend interface {
	List(ctx context.Context, supplierID, factoryID string) ([]ManifestException, error)
	Get(ctx context.Context, exceptionID, supplierID, factoryID string) (ManifestException, bool, error)
}

func (s *Service) exceptionBackend() factoryExceptionBackend {
	if s != nil && s.exceptionRepo != nil {
		return s.exceptionRepo
	}
	if s != nil && s.spannerClient != nil {
		return spannerExceptionRepo{client: s.spannerClient}
	}
	return nil
}

func writeFactoryExceptionList(w http.ResponseWriter, rows []ManifestException, escalatedOnly bool) {
	if rows == nil {
		rows = []ManifestException{}
	}
	if escalatedOnly {
		filtered := make([]ManifestException, 0, len(rows))
		for i := range rows {
			if rows[i].Escalated {
				filtered = append(filtered, rows[i])
			}
		}
		rows = filtered
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].CreatedAt > rows[j].CreatedAt })
	writeJSON(w, http.StatusOK, map[string]any{"exceptions": rows})
}

func (s *Service) lookupManifestException(ctx context.Context, exceptionID string) (row ManifestException, fromMemory bool, ok bool, err error) {
	backend := s.exceptionBackend()
	if backend != nil {
		row, ok, err = backend.Get(ctx, exceptionID, s.resolveSupplierScope(ctx), strings.TrimSpace(s.resolveFactoryNode(ctx)))
		if err != nil {
			return ManifestException{}, false, false, err
		}
		if ok {
			return row, false, true, nil
		}
		if !s.portalSeedEnabled() {
			return ManifestException{}, false, false, nil
		}
	}

	s.mu.Lock()
	s.ensureDemoDataLocked()
	for i := range s.manifestExceptions {
		if s.manifestExceptions[i].ExceptionID == exceptionID {
			row = s.manifestExceptions[i]
			s.mu.Unlock()
			return row, true, true, nil
		}
	}
	s.mu.Unlock()
	return ManifestException{}, false, false, nil
}

func removeMemoryExceptionLocked(rows []ManifestException, exceptionID string) []ManifestException {
	for i := range rows {
		if rows[i].ExceptionID == exceptionID {
			return append(rows[:i], rows[i+1:]...)
		}
	}
	return rows
}

type spannerExceptionRepo struct {
	client *spanner.Client
}

func (r spannerExceptionRepo) List(ctx context.Context, supplierID, factoryID string) ([]ManifestException, error) {
	if r.client == nil {
		return []ManifestException{}, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT e.ExceptionId, e.OrderId, COALESCE(e.ManifestId, ''), e.Reason,
		             COALESCE(e.Metadata, ''), e.AttemptCount, e.EscalatedAt, e.CreatedAt
		      FROM ManifestExceptions e
		      INNER JOIN FactoryTruckManifests m ON e.ManifestId = m.ManifestId
		      WHERE e.SupplierId = @supplierId
		        AND m.FactoryId = @factoryId
		        AND e.ResolvedAt IS NULL
		      ORDER BY e.CreatedAt DESC
		      LIMIT 200`,
		Params: map[string]any{
			"supplierId": supplierID,
			"factoryId":  factoryID,
		},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	out := make([]ManifestException, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		rec, err := scanFactoryExceptionRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, nil
}

func (r spannerExceptionRepo) Get(ctx context.Context, exceptionID, supplierID, factoryID string) (ManifestException, bool, error) {
	if r.client == nil {
		return ManifestException{}, false, nil
	}
	row, err := r.client.Single().ReadRow(ctx, "ManifestExceptions", spanner.Key{exceptionID},
		[]string{"ExceptionId", "OrderId", "ManifestId", "Reason", "Metadata", "AttemptCount", "EscalatedAt", "CreatedAt", "SupplierId"})
	if err != nil {
		if qcRowMissing(err) {
			return ManifestException{}, false, nil
		}
		return ManifestException{}, false, err
	}
	rec, storedSupplier, err := scanFactoryExceptionLookupRow(row)
	if err != nil {
		return ManifestException{}, false, err
	}
	if strings.TrimSpace(storedSupplier) != strings.TrimSpace(supplierID) {
		return ManifestException{}, false, nil
	}
	if strings.TrimSpace(rec.ManifestID) == "" {
		return ManifestException{}, false, nil
	}
	manifest, err := r.client.Single().ReadRow(ctx, "FactoryTruckManifests", spanner.Key{rec.ManifestID}, []string{"FactoryId"})
	if err != nil {
		if qcRowMissing(err) {
			return ManifestException{}, false, nil
		}
		return ManifestException{}, false, err
	}
	var owner string
	if err := manifest.Column(0, &owner); err != nil {
		return ManifestException{}, false, err
	}
	if strings.TrimSpace(owner) != strings.TrimSpace(factoryID) {
		return ManifestException{}, false, nil
	}
	return rec, true, nil
}

func scanFactoryExceptionRow(row *spanner.Row) (ManifestException, error) {
	var rec ManifestException
	var orderID, reason string
	var manifestID, metadata spanner.NullString
	var attemptCount int64
	var escalatedAt spanner.NullTime
	var createdAt time.Time
	if err := row.Columns(&rec.ExceptionID, &orderID, &manifestID, &reason, &metadata, &attemptCount, &escalatedAt, &createdAt); err != nil {
		return ManifestException{}, err
	}
	rec.TransferID = orderID
	rec.ManifestID = manifestID.StringVal
	rec.Reason = reason
	if metadata.Valid {
		rec.Metadata = metadata.StringVal
	}
	rec.AttemptCount = attemptCount
	rec.Escalated = escalatedAt.Valid
	rec.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
	return rec, nil
}

func scanFactoryExceptionLookupRow(row *spanner.Row) (ManifestException, string, error) {
	var rec ManifestException
	var orderID, reason, storedSupplier string
	var manifestID, metadata spanner.NullString
	var attemptCount int64
	var escalatedAt spanner.NullTime
	var createdAt time.Time
	if err := row.Columns(&rec.ExceptionID, &orderID, &manifestID, &reason, &metadata, &attemptCount, &escalatedAt, &createdAt, &storedSupplier); err != nil {
		return ManifestException{}, "", err
	}
	rec.TransferID = orderID
	rec.ManifestID = manifestID.StringVal
	rec.Reason = reason
	if metadata.Valid {
		rec.Metadata = metadata.StringVal
	}
	rec.AttemptCount = attemptCount
	rec.Escalated = escalatedAt.Valid
	rec.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
	return rec, storedSupplier, nil
}
