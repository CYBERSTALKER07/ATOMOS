package ontology

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

// SpannerRepository persists OntologyObjects and OntologyGraphEdges to Cloud Spanner
// with monotonic Version checks, <=900-mutation chunking, and atomic OutboxEvents buffering.
type SpannerRepository struct {
	client *spanner.Client
	now    func() time.Time
}

// NewSpannerRepository constructs a Cloud Spanner ontology repository.
func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	if client == nil {
		return nil
	}
	return &SpannerRepository{
		client: client,
		now:    func() time.Time { return time.Now().UTC() },
	}
}

// Client exposes the underlying Spanner client.
func (r *SpannerRepository) Client() *spanner.Client {
	if r == nil {
		return nil
	}
	return r.client
}

// UpsertBatch persists deduplicated OntologyObjects and OntologyGraphEdges in <=900-mutation
// Spanner transactions, skipping stale objects whose incoming Version <= existing Version.
func (r *SpannerRepository) UpsertBatch(ctx context.Context, objs []*OntologyObject, edges []*OntologyGraphEdge, opts UpsertOptions) (UpsertStats, error) {
	if r == nil || r.client == nil {
		return UpsertStats{}, ErrRepoUnavailable
	}
	if err := ctx.Err(); err != nil {
		return UpsertStats{}, err
	}

	dedupObjs := DeduplicateObjects(objs)
	dedupEdges := DeduplicateEdges(edges)

	for _, o := range dedupObjs {
		if err := ValidateObject(o); err != nil {
			return UpsertStats{}, err
		}
	}
	for _, e := range dedupEdges {
		if err := ValidateEdge(e); err != nil {
			return UpsertStats{}, err
		}
	}

	var stats UpsertStats
	objPerBatch := MaxChunkMutations
	if opts.EmitOutbox {
		// Each changed object writes 1 OntologyObjects row + 1 OutboxEvents row = 2 mutations.
		objPerBatch = MaxChunkMutations / 2
	}
	if objPerBatch < 1 {
		objPerBatch = 1
	}

	for start := 0; start < len(dedupObjs); start += objPerBatch {
		end := start + objPerBatch
		if end > len(dedupObjs) {
			end = len(dedupObjs)
		}
		chunk := dedupObjs[start:end]
		upserted, skipped, err := r.upsertObjectsChunk(ctx, chunk, opts)
		if err != nil {
			return stats, err
		}
		stats.ObjectsUpserted += upserted
		stats.ObjectsSkipped += skipped
		stats.BatchesCommitted++
	}

	for start := 0; start < len(dedupEdges); start += MaxChunkMutations {
		end := start + MaxChunkMutations
		if end > len(dedupEdges) {
			end = len(dedupEdges)
		}
		chunk := dedupEdges[start:end]
		upserted, err := r.upsertEdgesChunk(ctx, chunk)
		if err != nil {
			return stats, err
		}
		stats.EdgesUpserted += upserted
		stats.BatchesCommitted++
	}

	return stats, nil
}

type existingObjectMeta struct {
	SupplierID       string
	Version          int64
	PropertiesJSON   string
	SecurityMarkings []string
	ValidFrom        time.Time
}

func (r *SpannerRepository) upsertObjectsChunk(ctx context.Context, chunk []*OntologyObject, opts UpsertOptions) (int, int, error) {
	if len(chunk) == 0 {
		return 0, 0, nil
	}
	var upsertedCount, skippedCount int
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		upsertedCount = 0
		skippedCount = 0

		keys := make([]spanner.Key, 0, len(chunk))
		for _, o := range chunk {
			keys = append(keys, spanner.Key{NormalizeCellID(o.CellID), o.ObjectType, o.ObjectID})
		}

		existingByKey := make(map[objKey]existingObjectMeta, len(chunk))
		iter := txn.Read(ctx, "OntologyObjects", spanner.KeySetFromKeys(keys...), []string{
			"CellId", "ObjectType", "ObjectId", "SupplierId", "Version", "PropertiesJson", "SecurityMarkings", "ValidFrom",
		})
		defer iter.Stop()
		for {
			row, readErr := iter.Next()
			if errors.Is(readErr, iterator.Done) {
				break
			}
			if readErr != nil {
				return fmt.Errorf("read existing OntologyObjects chunk: %w", readErr)
			}
			var cell, oType, oID, supID, propsJSON string
			var ver int64
			var markings []string
			var validFrom time.Time
			if err := row.Columns(&cell, &oType, &oID, &supID, &ver, &propsJSON, &markings, &validFrom); err != nil {
				return fmt.Errorf("scan existing OntologyObjects row: %w", err)
			}
			existingByKey[objKey{cellID: cell, objectType: oType, objectID: oID}] = existingObjectMeta{
				SupplierID:       supID,
				Version:          ver,
				PropertiesJSON:   propsJSON,
				SecurityMarkings: NormalizeMarkings(markings),
				ValidFrom:        validFrom,
			}
		}

		muts := make([]*spanner.Mutation, 0, len(chunk)*2)
		buf := outbox.NewSpannerTxnBuffer(txn)
		now := r.now()

		for _, incoming := range chunk {
			cellID := NormalizeCellID(incoming.CellID)
			k := objKey{cellID: cellID, objectType: incoming.ObjectType, objectID: incoming.ObjectID}
			existing, exists := existingByKey[k]

			if exists && incoming.Version > 0 && existing.Version >= incoming.Version {
				skippedCount++
				continue
			}

			supplierID := strings.TrimSpace(incoming.SupplierID)
			if supplierID == "" && exists {
				supplierID = existing.SupplierID
			}
			if supplierID == "" {
				supplierID = outbox.PlatformSupplierID
			}

			markings := incoming.SecurityMarkings
			if exists {
				markings = MergeMarkings(existing.SecurityMarkings, markings)
			} else {
				markings = NormalizeMarkings(markings)
			}
			if markings == nil {
				markings = []string{}
			}

			propsJSON := incoming.PropertiesJSON
			if strings.TrimSpace(propsJSON) == "" {
				propsJSON = "{}"
			}

			if exists && opts.SkipNoOpIfUnchanged && incoming.Version <= 0 &&
				existing.PropertiesJSON == propsJSON &&
				slices.Equal(existing.SecurityMarkings, markings) &&
				existing.SupplierID == supplierID {
				skippedCount++
				continue
			}

			nextVer := incoming.Version
			if nextVer <= 0 {
				if exists {
					nextVer = existing.Version + 1
				} else {
					nextVer = 1
				}
			}

			var validFrom any = spanner.CommitTimestamp
			if exists && !existing.ValidFrom.IsZero() {
				validFrom = existing.ValidFrom
			}

			muts = append(muts, spanner.InsertOrUpdateMap("OntologyObjects", map[string]any{
				"CellId":           cellID,
				"ObjectType":       incoming.ObjectType,
				"ObjectId":         incoming.ObjectID,
				"SupplierId":       supplierID,
				"Version":          nextVer,
				"PropertiesJson":   propsJSON,
				"SecurityMarkings": markings,
				"ValidFrom":        validFrom,
				"UpdatedAt":        spanner.CommitTimestamp,
			}))
			upsertedCount++

			if opts.EmitOutbox {
				changeKind := opts.ChangeKind
				if changeKind == "" {
					if exists {
						changeKind = "UPDATED"
					} else {
						changeKind = "CREATED"
					}
				}
				ev := events.OntologyObjectChangedEvent{
					BaseEvent: events.BaseEvent{
						Type:      events.EventOntologyObjectChanged,
						Version:   nextVer,
						Timestamp: now.Format(time.RFC3339Nano),
					},
					CellID:           cellID,
					ObjectType:       incoming.ObjectType,
					ObjectID:         incoming.ObjectID,
					SupplierID:       supplierID,
					Version:          nextVer,
					ChangeKind:       changeKind,
					SourceEventType:  opts.SourceEventType,
					SourceEventID:    opts.SourceEventID,
					SecurityMarkings: slices.Clone(markings),
					UpdatedAt:        now.Format(time.RFC3339Nano),
				}
				aggID := fmt.Sprintf("%s/%s/%s", cellID, incoming.ObjectType, incoming.ObjectID)
				if len(aggID) > 64 {
					aggID = incoming.ObjectID
				}
				if emitErr := outbox.EmitJSON(ctx, buf, events.AggregateOntologyObject, aggID, events.TopicIntel, ev); emitErr != nil {
					return emitErr
				}
			}
		}

		if len(muts) == 0 {
			return nil
		}
		if err := txn.BufferWrite(muts); err != nil {
			return fmt.Errorf("buffer OntologyObjects chunk: %w", err)
		}
		if err := buf.Flush(ctx); err != nil {
			return fmt.Errorf("flush ontology outbox events: %w", err)
		}
		return nil
	})
	if err != nil {
		return 0, 0, err
	}
	return upsertedCount, skippedCount, nil
}

func (r *SpannerRepository) upsertEdgesChunk(ctx context.Context, chunk []*OntologyGraphEdge) (int, error) {
	if len(chunk) == 0 {
		return 0, nil
	}
	var upserted int
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		upserted = 0
		keys := make([]spanner.Key, 0, len(chunk))
		for _, e := range chunk {
			keys = append(keys, spanner.Key{
				NormalizeCellID(e.CellID),
				e.SourceObjectType,
				e.SourceObjectID,
				e.LinkType,
				e.TargetObjectType,
				e.TargetObjectID,
			})
		}

		existingValidFrom := make(map[edgeKey]time.Time, len(chunk))
		iter := txn.Read(ctx, "OntologyGraphEdges", spanner.KeySetFromKeys(keys...), []string{
			"CellId", "SourceObjectType", "SourceObjectId", "LinkType", "TargetObjectType", "TargetObjectId", "ValidFrom",
		})
		defer iter.Stop()
		for {
			row, readErr := iter.Next()
			if errors.Is(readErr, iterator.Done) {
				break
			}
			if readErr != nil {
				return fmt.Errorf("read existing OntologyGraphEdges chunk: %w", readErr)
			}
			var cell, st, sid, lt, dt, did string
			var vf time.Time
			if err := row.Columns(&cell, &st, &sid, &lt, &dt, &did, &vf); err != nil {
				return fmt.Errorf("scan existing OntologyGraphEdges row: %w", err)
			}
			existingValidFrom[edgeKey{
				cellID: cell, srcType: st, srcID: sid, linkType: lt, dstType: dt, dstID: did,
			}] = vf
		}

		muts := make([]*spanner.Mutation, 0, len(chunk))
		for _, e := range chunk {
			cellID := NormalizeCellID(e.CellID)
			k := edgeKey{
				cellID:   cellID,
				srcType:  e.SourceObjectType,
				srcID:    e.SourceObjectID,
				linkType: e.LinkType,
				dstType:  e.TargetObjectType,
				dstID:    e.TargetObjectID,
			}
			var validFrom any = spanner.CommitTimestamp
			if vf, ok := existingValidFrom[k]; ok && !vf.IsZero() {
				validFrom = vf
			}
			var validTo spanner.NullTime
			if e.ValidTo != nil && !e.ValidTo.IsZero() {
				validTo = spanner.NullTime{Time: e.ValidTo.UTC(), Valid: true}
			}
			var meta spanner.NullString
			if strings.TrimSpace(e.MetadataJSON) != "" {
				meta = spanner.NullString{StringVal: e.MetadataJSON, Valid: true}
			}
			supplierID := strings.TrimSpace(e.SupplierID)
			if supplierID == "" {
				supplierID = outbox.PlatformSupplierID
			}
			muts = append(muts, spanner.InsertOrUpdateMap("OntologyGraphEdges", map[string]any{
				"CellId":            cellID,
				"SourceObjectType":  e.SourceObjectType,
				"SourceObjectId":    e.SourceObjectID,
				"LinkType":          e.LinkType,
				"TargetObjectType":  e.TargetObjectType,
				"TargetObjectId":    e.TargetObjectID,
				"SupplierId":        supplierID,
				"CreatedByActionId": e.CreatedByActionID,
				"MetadataJson":      meta,
				"ValidFrom":         validFrom,
				"ValidTo":           validTo,
			}))
			upserted++
		}
		if len(muts) > MaxChunkMutations {
			return fmt.Errorf("ontology: chunk mutations %d exceed ceiling %d", len(muts), MaxChunkMutations)
		}
		return txn.BufferWrite(muts)
	})
	if err != nil {
		return 0, err
	}
	return upserted, nil
}

// GetObject reads a single OntologyObject via StrongRead.
func (r *SpannerRepository) GetObject(ctx context.Context, cellID, objectType, objectID string) (*OntologyObject, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	stmt := spanner.Statement{
		SQL: `SELECT CellId, ObjectType, ObjectId, SupplierId, Version, PropertiesJson, SecurityMarkings, ValidFrom, UpdatedAt
		      FROM OntologyObjects
		      WHERE CellId = @cellID AND ObjectType = @objectType AND ObjectId = @objectID`,
		Params: map[string]any{
			"cellID":     NormalizeCellID(cellID),
			"objectType": strings.TrimSpace(objectType),
			"objectID":   strings.TrimSpace(objectID),
		},
	}
	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return nil, ErrObjectNotFound
		}
		return nil, err
	}
	return scanOntologyObjectRow(row)
}

// ListObjects queries OntologyObjects with tenant/type/marking/search filters and pagination.
func (r *SpannerRepository) ListObjects(ctx context.Context, filter ListObjectsFilter) ([]*OntologyObject, int, error) {
	if r == nil || r.client == nil {
		return nil, 0, ErrRepoUnavailable
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	where := []string{"CellId = @cellID"}
	params := map[string]any{
		"cellID": NormalizeCellID(filter.CellID),
		"limit":  int64(limit),
		"offset": int64(offset),
	}
	if sid := strings.TrimSpace(filter.SupplierID); sid != "" {
		where = append(where, "SupplierId = @supplierID")
		params["supplierID"] = sid
	}
	if ot := strings.TrimSpace(filter.ObjectType); ot != "" {
		where = append(where, "ObjectType = @objectType")
		params["objectType"] = ot
	}
	if mk := strings.TrimSpace(filter.Marking); mk != "" {
		where = append(where, "@marking IN UNNEST(SecurityMarkings)")
		params["marking"] = mk
	}
	if q := strings.ToLower(strings.TrimSpace(filter.Search)); q != "" {
		where = append(where, "(LOWER(ObjectId) LIKE @q OR LOWER(PropertiesJson) LIKE @q)")
		params["q"] = "%" + q + "%"
	}
	whereClause := strings.Join(where, " AND ")

	ro := r.client.ReadOnlyTransaction()
	defer ro.Close()

	countIter := ro.Query(ctx, spanner.Statement{
		SQL:    "SELECT COUNT(*) FROM OntologyObjects WHERE " + whereClause,
		Params: params,
	})
	defer countIter.Stop()

	var total int64
	if row, err := countIter.Next(); err == nil {
		_ = row.Column(0, &total)
	} else if !errors.Is(err, iterator.Done) {
		return nil, 0, err
	}

	listIter := ro.Query(ctx, spanner.Statement{
		SQL: fmt.Sprintf(`SELECT CellId, ObjectType, ObjectId, SupplierId, Version, PropertiesJson, SecurityMarkings, ValidFrom, UpdatedAt
		                 FROM OntologyObjects
		                 WHERE %s
		                 ORDER BY ObjectType ASC, ObjectId ASC
		                 LIMIT @limit OFFSET @offset`, whereClause),
		Params: params,
	})
	defer listIter.Stop()

	var objs []*OntologyObject
	for {
		row, err := listIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, 0, err
		}
		o, scanErr := scanOntologyObjectRow(row)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		objs = append(objs, o)
	}
	if objs == nil {
		objs = []*OntologyObject{}
	}
	return objs, int(total), nil
}

// GetNeighbors executes a bounded multi-hop BFS across OntologyGraphEdges (depth 1..4).
func (r *SpannerRepository) GetNeighbors(ctx context.Context, cellID, supplierID, objectType, objectID string, depth int, linkTypes []string) (*OntologyObject, []NeighborNode, []*OntologyGraphEdge, error) {
	if r == nil || r.client == nil {
		return nil, nil, nil, ErrRepoUnavailable
	}
	cellID = NormalizeCellID(cellID)
	supID := strings.TrimSpace(supplierID)
	objectType = strings.TrimSpace(objectType)
	objectID = strings.TrimSpace(objectID)
	if depth <= 0 {
		depth = 2
	}
	if depth > 4 {
		depth = 4
	}

	ro := r.client.ReadOnlyTransaction()
	defer ro.Close()

	rootObj, err := readObjectWithTx(ctx, ro, cellID, objectType, objectID)
	if err != nil {
		return nil, nil, nil, err
	}
	if supID != "" && rootObj.SupplierID != supID {
		return nil, nil, nil, ErrObjectNotFound
	}

	rootKey := objKey{cellID: cellID, objectType: objectType, objectID: objectID}
	visited := map[objKey]bool{rootKey: true}
	frontier := []objKey{rootKey}

	type discoveredEdge struct {
		edge      *OntologyGraphEdge
		targetKey objKey
		direction string
		parentKey objKey
		hop       int
	}
	edgeSeen := make(map[edgeKey]bool)
	var allEdges []*OntologyGraphEdge
	var pending []discoveredEdge

	for hop := 1; hop <= depth && len(frontier) > 0; hop++ {
		byType := make(map[string][]string)
		for _, fk := range frontier {
			byType[fk.objectType] = append(byType[fk.objectType], fk.objectID)
		}

		var nextFrontier []objKey
		for fType, fIDs := range byType {
			outEdges, err := queryFrontierEdges(ctx, ro, cellID, supID, fType, fIDs, linkTypes, true)
			if err != nil {
				return nil, nil, nil, err
			}
			inEdges, err := queryFrontierEdges(ctx, ro, cellID, supID, fType, fIDs, linkTypes, false)
			if err != nil {
				return nil, nil, nil, err
			}

			for _, e := range outEdges {
				ek := edgeKey{
					cellID: e.CellID, srcType: e.SourceObjectType, srcID: e.SourceObjectID,
					linkType: e.LinkType, dstType: e.TargetObjectType, dstID: e.TargetObjectID,
				}
				if !edgeSeen[ek] {
					edgeSeen[ek] = true
					allEdges = append(allEdges, e)
				}
				tk := objKey{cellID: cellID, objectType: e.TargetObjectType, objectID: e.TargetObjectID}
				if !visited[tk] {
					visited[tk] = true
					pk := objKey{cellID: cellID, objectType: e.SourceObjectType, objectID: e.SourceObjectID}
					pending = append(pending, discoveredEdge{edge: e, targetKey: tk, direction: "OUTBOUND", parentKey: pk, hop: hop})
					nextFrontier = append(nextFrontier, tk)
				}
			}

			for _, e := range inEdges {
				ek := edgeKey{
					cellID: e.CellID, srcType: e.SourceObjectType, srcID: e.SourceObjectID,
					linkType: e.LinkType, dstType: e.TargetObjectType, dstID: e.TargetObjectID,
				}
				if !edgeSeen[ek] {
					edgeSeen[ek] = true
					allEdges = append(allEdges, e)
				}
				tk := objKey{cellID: cellID, objectType: e.SourceObjectType, objectID: e.SourceObjectID}
				if !visited[tk] {
					visited[tk] = true
					pk := objKey{cellID: cellID, objectType: e.TargetObjectType, objectID: e.TargetObjectID}
					pending = append(pending, discoveredEdge{edge: e, targetKey: tk, direction: "INBOUND", parentKey: pk, hop: hop})
					nextFrontier = append(nextFrontier, tk)
				}
			}
		}
		frontier = nextFrontier
	}

	if len(pending) == 0 {
		return rootObj, []NeighborNode{}, allEdges, nil
	}

	spannerKeys := make([]spanner.Key, 0, len(pending))
	for _, p := range pending {
		spannerKeys = append(spannerKeys, spanner.Key{p.targetKey.cellID, p.targetKey.objectType, p.targetKey.objectID})
	}
	objByKey := make(map[objKey]*OntologyObject, len(spannerKeys))
	objIter := ro.Read(ctx, "OntologyObjects", spanner.KeySetFromKeys(spannerKeys...), []string{
		"CellId", "ObjectType", "ObjectId", "SupplierId", "Version", "PropertiesJson", "SecurityMarkings", "ValidFrom", "UpdatedAt",
	})
	defer objIter.Stop()
	for {
		row, readErr := objIter.Next()
		if errors.Is(readErr, iterator.Done) {
			break
		}
		if readErr != nil {
			return nil, nil, nil, readErr
		}
		o, scanErr := scanOntologyObjectRow(row)
		if scanErr != nil {
			return nil, nil, nil, scanErr
		}
		if supID != "" && o.SupplierID != supID {
			continue
		}
		objByKey[objKey{cellID: o.CellID, objectType: o.ObjectType, objectID: o.ObjectID}] = o
	}

	nodes := make([]NeighborNode, 0, len(pending))
	for _, p := range pending {
		if o, ok := objByKey[p.targetKey]; ok {
			nodes = append(nodes, NeighborNode{
				Object:      o,
				Hop:         p.hop,
				ViaLinkType: p.edge.LinkType,
				Direction:   p.direction,
				ParentType:  p.parentKey.objectType,
				ParentID:    p.parentKey.objectID,
			})
		}
	}
	return rootObj, nodes, allEdges, nil
}

// GetGraphForSupplier returns up to 2,000 objects and 5,000 active edges for a supplier.
func (r *SpannerRepository) GetGraphForSupplier(ctx context.Context, cellID, supplierID string) ([]*OntologyObject, []*OntologyGraphEdge, error) {
	if r == nil || r.client == nil {
		return nil, nil, ErrRepoUnavailable
	}
	cellID = NormalizeCellID(cellID)
	supID := strings.TrimSpace(supplierID)

	ro := r.client.ReadOnlyTransaction()
	defer ro.Close()

	objStmt := spanner.Statement{
		SQL: `SELECT CellId, ObjectType, ObjectId, SupplierId, Version, PropertiesJson, SecurityMarkings, ValidFrom, UpdatedAt
		      FROM OntologyObjects
		      WHERE CellId = @cellID AND SupplierId = @supplierID
		      ORDER BY ObjectType ASC, ObjectId ASC
		      LIMIT 2000`,
		Params: map[string]any{"cellID": cellID, "supplierID": supID},
	}
	objIter := ro.Query(ctx, objStmt)
	defer objIter.Stop()

	var objs []*OntologyObject
	for {
		row, err := objIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		o, scanErr := scanOntologyObjectRow(row)
		if scanErr != nil {
			return nil, nil, scanErr
		}
		objs = append(objs, o)
	}

	edgeStmt := spanner.Statement{
		SQL: `SELECT CellId, SourceObjectType, SourceObjectId, LinkType, TargetObjectType, TargetObjectId, SupplierId, CreatedByActionId, MetadataJson, ValidFrom, ValidTo
		      FROM OntologyGraphEdges
		      WHERE CellId = @cellID AND SupplierId = @supplierID AND ValidTo IS NULL
		      LIMIT 5000`,
		Params: map[string]any{"cellID": cellID, "supplierID": supID},
	}
	edgeIter := ro.Query(ctx, edgeStmt)
	defer edgeIter.Stop()

	var edges []*OntologyGraphEdge
	for {
		row, err := edgeIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		e, scanErr := scanOntologyEdgeRow(row)
		if scanErr != nil {
			return nil, nil, scanErr
		}
		edges = append(edges, e)
	}

	return objs, edges, nil
}

// ListActiveSupplierIDs lists all SupplierId rows for nightly ontology rebuild sweeps.
func (r *SpannerRepository) ListActiveSupplierIDs(ctx context.Context) ([]string, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{
		SQL: `SELECT SupplierId FROM Suppliers ORDER BY SupplierId ASC LIMIT 500`,
	})
	defer iter.Stop()

	var out []string
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var sid string
		if err := row.Column(0, &sid); err == nil && strings.TrimSpace(sid) != "" {
			out = append(out, strings.TrimSpace(sid))
		}
	}
	return out, nil
}

// LoadDomainSnapshot reads live Spanner domain tables for (cellID, supplierID) and materializes
// canonical v1 OntologyObjects and OntologyGraphEdges without floats, using ReadOnlyTransaction.
func (r *SpannerRepository) LoadDomainSnapshot(ctx context.Context, cellID, supplierID string) ([]*OntologyObject, []*OntologyGraphEdge, []string, error) {
	if r == nil || r.client == nil {
		return nil, nil, nil, ErrRepoUnavailable
	}
	cellID = NormalizeCellID(cellID)
	sid := strings.TrimSpace(supplierID)
	if sid == "" {
		return nil, nil, nil, fmt.Errorf("%w: supplier_id required", ErrInvalidIdentity)
	}

	ro := r.client.ReadOnlyTransaction()
	defer ro.Close()

	var objs []*OntologyObject
	var edges []*OntologyGraphEdge
	tables := make([]string, 0, 16)

	// 1. Suppliers & SupplierProfiles -> Supplier + LegalEntity
	if err := loadSuppliersSnapshot(ctx, ro, cellID, sid, &objs); err != nil {
		return nil, nil, nil, err
	}
	tables = append(tables, "Suppliers", "SupplierProfiles")

	// 2. Factories -> Factory
	if err := loadFactoriesSnapshot(ctx, ro, cellID, sid, &objs); err != nil {
		return nil, nil, nil, err
	}
	tables = append(tables, "Factories")

	// 3. Warehouses -> Warehouse (+ supplies edge from PrimaryFactoryId / Supplier)
	if err := loadWarehousesSnapshot(ctx, ro, cellID, sid, &objs, &edges); err != nil {
		return nil, nil, nil, err
	}
	tables = append(tables, "Warehouses")

	// 4. Products -> SKU (+ supplies edge from Supplier)
	if err := loadProductsSnapshot(ctx, ro, cellID, sid, &objs, &edges); err != nil {
		return nil, nil, nil, err
	}
	tables = append(tables, "Products")

	// 5. StockLots -> Lot (+ stocks edges Warehouse -> Lot, Lot -> SKU)
	if err := loadStockLotsSnapshot(ctx, ro, cellID, sid, &objs, &edges); err != nil {
		return nil, nil, nil, err
	}
	tables = append(tables, "StockLots")

	// 6. Retailers (scoped to supplier via Orders/SupplierRetailerLinks/RetailerCreditProfiles) -> Shop
	if err := loadRetailersSnapshot(ctx, ro, cellID, sid, &objs); err != nil {
		return nil, nil, nil, err
	}
	tables = append(tables, "Retailers")

	// 7. Vehicles -> Vehicle
	if err := loadVehiclesSnapshot(ctx, ro, cellID, sid, &objs); err != nil {
		return nil, nil, nil, err
	}
	tables = append(tables, "Vehicles")

	// 8. Drivers -> Driver
	if err := loadDriversSnapshot(ctx, ro, cellID, sid, &objs, &edges); err != nil {
		return nil, nil, nil, err
	}
	tables = append(tables, "Drivers")

	// 9. Orders + Orders.LineItemsJson -> Order & OrderLine (+ ordered edges)
	if err := loadOrdersSnapshot(ctx, ro, cellID, sid, &objs, &edges); err != nil {
		return nil, nil, nil, err
	}
	tables = append(tables, "Orders")

	// 10. SupplierTruckManifests + ManifestOrders -> Manifest (+ shipped_on edges)
	if err := loadManifestsSnapshot(ctx, ro, cellID, sid, &objs, &edges); err != nil {
		return nil, nil, nil, err
	}
	tables = append(tables, "SupplierTruckManifests", "ManifestOrders")

	// 11. ArInvoices -> NasiyaInvoice (+ owes edges)
	if err := loadArInvoicesSnapshot(ctx, ro, cellID, sid, &objs, &edges); err != nil {
		return nil, nil, nil, err
	}
	tables = append(tables, "ArInvoices")

	// 12. RetailerCreditProfiles -> CreditLine (+ owes / supplies edges)
	if err := loadCreditLinesSnapshot(ctx, ro, cellID, sid, &objs, &edges); err != nil {
		return nil, nil, nil, err
	}
	tables = append(tables, "RetailerCreditProfiles")

	// 13. PaymentSessions -> Payment (+ ordered / owes edges)
	if err := loadPaymentsSnapshot(ctx, ro, cellID, sid, &objs, &edges); err != nil {
		return nil, nil, nil, err
	}
	tables = append(tables, "PaymentSessions")

	// 14. OrderFiscalReceipts -> FiscalDoc (+ ordered edges)
	if err := loadFiscalDocsSnapshot(ctx, ro, cellID, sid, &objs, &edges); err != nil {
		return nil, nil, nil, err
	}
	tables = append(tables, "OrderFiscalReceipts")

	// 15. NegotiationProposals + ReplenishmentInsights -> Proposal
	if err := loadProposalsSnapshot(ctx, ro, cellID, sid, &objs, &edges); err != nil {
		return nil, nil, nil, err
	}
	tables = append(tables, "NegotiationProposals", "ReplenishmentInsights")

	return objs, edges, tables, nil
}

func readObjectWithTx(ctx context.Context, ro *spanner.ReadOnlyTransaction, cellID, objectType, objectID string) (*OntologyObject, error) {
	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT CellId, ObjectType, ObjectId, SupplierId, Version, PropertiesJson, SecurityMarkings, ValidFrom, UpdatedAt
		      FROM OntologyObjects
		      WHERE CellId = @cellID AND ObjectType = @objectType AND ObjectId = @objectID`,
		Params: map[string]any{
			"cellID":     cellID,
			"objectType": objectType,
			"objectID":   objectID,
		},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return nil, ErrObjectNotFound
		}
		return nil, err
	}
	return scanOntologyObjectRow(row)
}

func queryFrontierEdges(ctx context.Context, ro *spanner.ReadOnlyTransaction, cellID, supID, nodeType string, nodeIDs, linkTypes []string, outbound bool) ([]*OntologyGraphEdge, error) {
	if len(nodeIDs) == 0 {
		return nil, nil
	}
	params := map[string]any{
		"cellID":   cellID,
		"nodeType": nodeType,
		"nodeIDs":  nodeIDs,
	}
	var sql string
	if outbound {
		sql = `SELECT CellId, SourceObjectType, SourceObjectId, LinkType, TargetObjectType, TargetObjectId, SupplierId, CreatedByActionId, MetadataJson, ValidFrom, ValidTo
		       FROM OntologyGraphEdges
		       WHERE CellId = @cellID AND SourceObjectType = @nodeType AND SourceObjectId IN UNNEST(@nodeIDs) AND ValidTo IS NULL`
	} else {
		sql = `SELECT CellId, SourceObjectType, SourceObjectId, LinkType, TargetObjectType, TargetObjectId, SupplierId, CreatedByActionId, MetadataJson, ValidFrom, ValidTo
		       FROM OntologyGraphEdges@{FORCE_INDEX=Idx_OntologyGraphEdges_Reverse}
		       WHERE CellId = @cellID AND TargetObjectType = @nodeType AND TargetObjectId IN UNNEST(@nodeIDs) AND ValidTo IS NULL`
	}
	if supID != "" {
		sql += " AND SupplierId = @supplierID"
		params["supplierID"] = supID
	}
	if len(linkTypes) > 0 {
		sql += " AND LinkType IN UNNEST(@linkTypes)"
		params["linkTypes"] = linkTypes
	}

	iter := ro.Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	var out []*OntologyGraphEdge
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		e, scanErr := scanOntologyEdgeRow(row)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, e)
	}
	return out, nil
}

func scanOntologyObjectRow(row *spanner.Row) (*OntologyObject, error) {
	var cell, oType, oID, supID, propsJSON string
	var ver int64
	var markings []string
	var validFrom, updatedAt time.Time
	if err := row.Columns(&cell, &oType, &oID, &supID, &ver, &propsJSON, &markings, &validFrom, &updatedAt); err != nil {
		return nil, err
	}
	props, err := DecodeJSONMapNoFloats([]byte(propsJSON))
	if err != nil {
		props = map[string]any{}
	}
	return &OntologyObject{
		CellID:           cell,
		ObjectType:       oType,
		ObjectID:         oID,
		SupplierID:       supID,
		Version:          ver,
		Properties:       props,
		PropertiesJSON:   propsJSON,
		SecurityMarkings: NormalizeMarkings(markings),
		ValidFrom:        validFrom,
		UpdatedAt:        updatedAt,
	}, nil
}

func scanOntologyEdgeRow(row *spanner.Row) (*OntologyGraphEdge, error) {
	var cell, st, sid, lt, dt, did, supID, actionID string
	var metaNS spanner.NullString
	var validFrom time.Time
	var validTo spanner.NullTime
	if err := row.Columns(&cell, &st, &sid, &lt, &dt, &did, &supID, &actionID, &metaNS, &validFrom, &validTo); err != nil {
		return nil, err
	}
	edge := &OntologyGraphEdge{
		CellID:            cell,
		SourceObjectType:  st,
		SourceObjectID:    sid,
		LinkType:          lt,
		TargetObjectType:  dt,
		TargetObjectID:    did,
		SupplierID:        supID,
		CreatedByActionID: actionID,
		ValidFrom:         validFrom,
	}
	if metaNS.Valid && strings.TrimSpace(metaNS.StringVal) != "" {
		edge.MetadataJSON = metaNS.StringVal
		_ = json.Unmarshal([]byte(metaNS.StringVal), &edge.Metadata)
	}
	if validTo.Valid {
		t := validTo.Time
		edge.ValidTo = &t
	}
	return edge, nil
}

// ── Domain snapshot loaders (zero floats, ReadOnlyTransaction only) ─────────

func loadSuppliersSnapshot(ctx context.Context, ro *spanner.ReadOnlyTransaction, cellID, sid string, objs *[]*OntologyObject) error {
	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT s.SupplierId, s.Name, s.CountryCode, s.Currency, s.IsConfigured,
		             COALESCE(s.MarketCode, ''), COALESCE(s.HomeCell, ''),
		             COALESCE(p.TaxId, ''), COALESCE(p.CompanyRegNumber, ''),
		             COALESCE(p.ContactName, ''), COALESCE(p.Email, ''), COALESCE(p.Phone, ''),
		             COALESCE(p.BankName, ''), COALESCE(p.AccountHolder, ''), COALESCE(p.AccountNumber, ''),
		             COALESCE(p.SwiftBic, ''), COALESCE(p.IBAN, ''), COALESCE(p.Gln, '')
		      FROM Suppliers s
		      LEFT JOIN SupplierProfiles p ON s.SupplierId = p.SupplierId
		      WHERE s.SupplierId = @sid`,
		Params: map[string]any{"sid": sid},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		var id, name, cc, cur, market, homeCell, taxID, regNo, contact, email, phone, bank, holder, acctNo, swift, iban, gln string
		var configured bool
		if err := row.Columns(&id, &name, &cc, &cur, &configured, &market, &homeCell, &taxID, &regNo, &contact, &email, &phone, &bank, &holder, &acctNo, &swift, &iban, &gln); err != nil {
			return err
		}
		*objs = append(*objs, &OntologyObject{
			CellID:     cellID,
			ObjectType: TypeSupplier,
			ObjectID:   id,
			SupplierID: id,
			Properties: map[string]any{
				"supplier_id":   id,
				"name":          name,
				"country_code":  cc,
				"currency":      cur,
				"is_configured": configured,
				"market_code":   market,
				"home_cell":     homeCell,
				"gln":           gln,
			},
		})
		if taxID != "" || regNo != "" || bank != "" || iban != "" {
			*objs = append(*objs, &OntologyObject{
				CellID:           cellID,
				ObjectType:       TypeLegalEntity,
				ObjectID:         "le:" + id,
				SupplierID:       id,
				SecurityMarkings: []string{MarkingFinanceOnly, MarkingPII},
				Properties: map[string]any{
					"supplier_id":        id,
					"legal_name":         name,
					"tax_id":             taxID,
					"company_reg_number": regNo,
					"contact_name":       contact,
					"email":              email,
					"phone":              phone,
					"bank_name":          bank,
					"account_holder":     holder,
					"account_number":     acctNo,
					"swift_bic":          swift,
					"iban":               iban,
				},
			})
		}
	}
	return nil
}

func loadFactoriesSnapshot(ctx context.Context, ro *spanner.ReadOnlyTransaction, cellID, sid string, objs *[]*OntologyObject) error {
	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT FactoryId, SupplierId, Name, COALESCE(H3Cell, ''), COALESCE(Address, ''), IsActive, DailyOutputCapacity, COALESCE(CountryCode, '')
		      FROM Factories WHERE SupplierId = @sid`,
		Params: map[string]any{"sid": sid},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		var fid, supID, name, h3, addr, cc string
		var active bool
		var capDaily int64
		if err := row.Columns(&fid, &supID, &name, &h3, &addr, &active, &capDaily, &cc); err != nil {
			return err
		}
		*objs = append(*objs, &OntologyObject{
			CellID:     cellID,
			ObjectType: TypeFactory,
			ObjectID:   fid,
			SupplierID: supID,
			Properties: map[string]any{
				"factory_id":            fid,
				"supplier_id":           supID,
				"name":                  name,
				"h3_cell":               h3,
				"address":               addr,
				"is_active":             active,
				"daily_output_capacity": capDaily,
				"country_code":          cc,
			},
		})
	}
	return nil
}

func loadWarehousesSnapshot(ctx context.Context, ro *spanner.ReadOnlyTransaction, cellID, sid string, objs *[]*OntologyObject, edges *[]*OntologyGraphEdge) error {
	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT WarehouseId, SupplierId, Name, COALESCE(Address, ''), COALESCE(PrimaryFactoryId, ''),
		             COALESCE(SecondaryFactoryId, ''), TransferMode, IsActive, IsOnShift, COALESCE(H3Cell, ''), COALESCE(CountryCode, ''), COALESCE(Gln, '')
		      FROM Warehouses WHERE SupplierId = @sid`,
		Params: map[string]any{"sid": sid},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		var wid, supID, name, addr, f1, f2, mode, h3, cc, gln string
		var active, onShift bool
		if err := row.Columns(&wid, &supID, &name, &addr, &f1, &f2, &mode, &active, &onShift, &h3, &cc, &gln); err != nil {
			return err
		}
		*objs = append(*objs, &OntologyObject{
			CellID:     cellID,
			ObjectType: TypeWarehouse,
			ObjectID:   wid,
			SupplierID: supID,
			Properties: map[string]any{
				"warehouse_id":         wid,
				"supplier_id":          supID,
				"name":                 name,
				"address":              addr,
				"primary_factory_id":   f1,
				"secondary_factory_id": f2,
				"transfer_mode":        mode,
				"is_active":            active,
				"is_on_shift":          onShift,
				"h3_cell":              h3,
				"country_code":         cc,
				"gln":                  gln,
			},
		})
		*edges = append(*edges, &OntologyGraphEdge{
			CellID: cellID, SourceObjectType: TypeSupplier, SourceObjectID: supID,
			LinkType: LinkSupplies, TargetObjectType: TypeWarehouse, TargetObjectID: wid,
			SupplierID: supID, CreatedByActionID: "rebuild-from-spanner",
		})
		if strings.TrimSpace(f1) != "" {
			*edges = append(*edges, &OntologyGraphEdge{
				CellID: cellID, SourceObjectType: TypeFactory, SourceObjectID: f1,
				LinkType: LinkSupplies, TargetObjectType: TypeWarehouse, TargetObjectID: wid,
				SupplierID: supID, CreatedByActionID: "rebuild-from-spanner",
			})
		}
	}
	return nil
}

func loadProductsSnapshot(ctx context.Context, ro *spanner.ReadOnlyTransaction, cellID, sid string, objs *[]*OntologyObject, edges *[]*OntologyGraphEdge) error {
	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT ProductId, SupplierId, CategoryId, Name, PriceMinor, Currency, StockQuantity,
		             Unit, SaleUnit, COALESCE(Barcode, ''), HandlingClass, RequiresColdChain, IsHazardous, IsPerishable, IsActive, Version
		      FROM Products WHERE SupplierId = @sid`,
		Params: map[string]any{"sid": sid},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		var pid, supID, catID, name, cur, unit, saleUnit, barcode, handling string
		var priceMinor, stockQty, ver int64
		var cold, haz, perish, active bool
		if err := row.Columns(&pid, &supID, &catID, &name, &priceMinor, &cur, &stockQty, &unit, &saleUnit, &barcode, &handling, &cold, &haz, &perish, &active, &ver); err != nil {
			return err
		}
		*objs = append(*objs, &OntologyObject{
			CellID:     cellID,
			ObjectType: TypeSKU,
			ObjectID:   pid,
			SupplierID: supID,
			Version:    ver,
			Properties: map[string]any{
				"product_id":          pid,
				"supplier_id":         supID,
				"category_id":         catID,
				"name":                name,
				"price_minor":         priceMinor,
				"currency":            cur,
				"stock_quantity":      stockQty,
				"unit":                unit,
				"sale_unit":           saleUnit,
				"barcode":             barcode,
				"handling_class":      handling,
				"requires_cold_chain": cold,
				"is_hazardous":        haz,
				"is_perishable":       perish,
				"is_active":           active,
			},
		})
		*edges = append(*edges, &OntologyGraphEdge{
			CellID: cellID, SourceObjectType: TypeSupplier, SourceObjectID: supID,
			LinkType: LinkSupplies, TargetObjectType: TypeSKU, TargetObjectID: pid,
			SupplierID: supID, CreatedByActionID: "rebuild-from-spanner",
		})
	}
	return nil
}

func loadStockLotsSnapshot(ctx context.Context, ro *spanner.ReadOnlyTransaction, cellID, sid string, objs *[]*OntologyObject, edges *[]*OntologyGraphEdge) error {
	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT LotId, SupplierId, WarehouseId, ProductId, LocationId, COALESCE(LotCode, ''),
		             ExpiryDate, ManufacturedDate, QuantityOnHand, QuantityReserved, Status
		      FROM StockLots WHERE SupplierId = @sid`,
		Params: map[string]any{"sid": sid},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		var lotID, supID, whID, prodID, locID, lotCode, status string
		var expDate, mfgDate spanner.NullDate
		var qtyHand, qtyRes int64
		if err := row.Columns(&lotID, &supID, &whID, &prodID, &locID, &lotCode, &expDate, &mfgDate, &qtyHand, &qtyRes, &status); err != nil {
			return err
		}
		props := map[string]any{
			"lot_id":            lotID,
			"supplier_id":       supID,
			"warehouse_id":      whID,
			"product_id":        prodID,
			"location_id":       locID,
			"lot_code":          lotCode,
			"quantity_on_hand":  qtyHand,
			"quantity_reserved": qtyRes,
			"status":            status,
		}
		if expDate.Valid {
			props["expiry_date"] = expDate.Date.String()
		}
		if mfgDate.Valid {
			props["manufactured_date"] = mfgDate.Date.String()
		}
		var markings []string
		upper := strings.ToUpper(status)
		if strings.Contains(upper, "RECALL") {
			markings = append(markings, MarkingLotRecallQuarantine)
		} else if strings.Contains(upper, "QUARANTINE") {
			markings = append(markings, MarkingDiscrepantQuarantine)
		}
		*objs = append(*objs, &OntologyObject{
			CellID:           cellID,
			ObjectType:       TypeLot,
			ObjectID:         lotID,
			SupplierID:       supID,
			SecurityMarkings: markings,
			Properties:       props,
		})
		*edges = append(*edges,
			&OntologyGraphEdge{
				CellID: cellID, SourceObjectType: TypeWarehouse, SourceObjectID: whID,
				LinkType: LinkStocks, TargetObjectType: TypeLot, TargetObjectID: lotID,
				SupplierID: supID, CreatedByActionID: "rebuild-from-spanner",
			},
			&OntologyGraphEdge{
				CellID: cellID, SourceObjectType: TypeLot, SourceObjectID: lotID,
				LinkType: LinkStocks, TargetObjectType: TypeSKU, TargetObjectID: prodID,
				SupplierID: supID, CreatedByActionID: "rebuild-from-spanner",
			},
		)
	}
	return nil
}

func loadRetailersSnapshot(ctx context.Context, ro *spanner.ReadOnlyTransaction, cellID, sid string, objs *[]*OntologyObject) error {
	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT r.RetailerId, COALESCE(r.Name, r.RetailerId), r.Phone, COALESCE(r.Email, ''), r.CountryCode,
		             COALESCE(r.H3Cell, ''), COALESCE(r.DeliveryAddress, ''), COALESCE(r.Gln, '')
		      FROM Retailers r
		      WHERE r.RetailerId IN (
		        SELECT RetailerId FROM Orders WHERE SupplierId = @sid
		        UNION DISTINCT
		        SELECT RetailerId FROM SupplierRetailerLinks WHERE SupplierId = @sid
		        UNION DISTINCT
		        SELECT RetailerId FROM RetailerCreditProfiles WHERE SupplierId = @sid
		      )`,
		Params: map[string]any{"sid": sid},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		var rid, name, phone, email, cc, h3, addr, gln string
		if err := row.Columns(&rid, &name, &phone, &email, &cc, &h3, &addr, &gln); err != nil {
			return err
		}
		*objs = append(*objs, &OntologyObject{
			CellID:           cellID,
			ObjectType:       TypeShop,
			ObjectID:         rid,
			SupplierID:       sid,
			SecurityMarkings: []string{MarkingPII},
			Properties: map[string]any{
				"retailer_id":      rid,
				"supplier_id":      sid,
				"name":             name,
				"phone":            phone,
				"email":            email,
				"country_code":     cc,
				"h3_cell":          h3,
				"delivery_address": addr,
				"gln":              gln,
			},
		})
	}
	return nil
}

func loadVehiclesSnapshot(ctx context.Context, ro *spanner.ReadOnlyTransaction, cellID, sid string, objs *[]*OntologyObject) error {
	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT VehicleId, COALESCE(Label, LicensePlate), LicensePlate, SupplierId, HomeNodeType, HomeNodeId,
		             VehicleClass, IsActive, COALESCE(UnavailableReason, '')
		      FROM Vehicles WHERE SupplierId = @sid`,
		Params: map[string]any{"sid": sid},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		var vid, label, plate, supID, hnt, hnid, vClass, unavail string
		var active bool
		if err := row.Columns(&vid, &label, &plate, &supID, &hnt, &hnid, &vClass, &active, &unavail); err != nil {
			return err
		}
		*objs = append(*objs, &OntologyObject{
			CellID:     cellID,
			ObjectType: TypeVehicle,
			ObjectID:   vid,
			SupplierID: supID,
			Properties: map[string]any{
				"vehicle_id":         vid,
				"label":              label,
				"license_plate":      plate,
				"supplier_id":        supID,
				"home_node_type":     hnt,
				"home_node_id":       hnid,
				"vehicle_class":      vClass,
				"is_active":          active,
				"unavailable_reason": unavail,
			},
		})
	}
	return nil
}

func loadDriversSnapshot(ctx context.Context, ro *spanner.ReadOnlyTransaction, cellID, sid string, objs *[]*OntologyObject, edges *[]*OntologyGraphEdge) error {
	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT DriverId, Name, Phone, SupplierId, HomeNodeType, HomeNodeId, COALESCE(VehicleId, ''), IsActive, OnShift, COALESCE(UnavailableReason, '')
		      FROM Drivers WHERE SupplierId = @sid`,
		Params: map[string]any{"sid": sid},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		var did, name, phone, supID, hnt, hnid, vid, unavail string
		var active, onShift bool
		if err := row.Columns(&did, &name, &phone, &supID, &hnt, &hnid, &vid, &active, &onShift, &unavail); err != nil {
			return err
		}
		*objs = append(*objs, &OntologyObject{
			CellID:           cellID,
			ObjectType:       TypeDriver,
			ObjectID:         did,
			SupplierID:       supID,
			SecurityMarkings: []string{MarkingPII},
			Properties: map[string]any{
				"driver_id":          did,
				"name":               name,
				"phone":              phone,
				"supplier_id":        supID,
				"home_node_type":     hnt,
				"home_node_id":       hnid,
				"vehicle_id":         vid,
				"is_active":          active,
				"on_shift":           onShift,
				"unavailable_reason": unavail,
			},
		})
		if vid != "" {
			*edges = append(*edges, &OntologyGraphEdge{
				CellID: cellID, SourceObjectType: TypeDriver, SourceObjectID: did,
				LinkType: LinkShippedOn, TargetObjectType: TypeVehicle, TargetObjectID: vid,
				SupplierID: supID, CreatedByActionID: "rebuild-from-spanner",
			})
		}
	}
	return nil
}

type orderLineSnapshotDTO struct {
	SKU           string `json:"sku"`
	Name          string `json:"name"`
	Quantity      int64  `json:"quantity"`
	UnitPrice     int64  `json:"unit_price_minor"`
	DeliveredQty  int64  `json:"delivered_qty,omitempty"`
	RemainingQty  int64  `json:"remaining_qty,omitempty"`
	OffloadStatus string `json:"offload_status,omitempty"`
	SubstituteSKU string `json:"substitute_sku,omitempty"`
	OriginalSKU   string `json:"original_sku,omitempty"`
	PriceGapMinor int64  `json:"price_gap_minor,omitempty"`
}

func loadOrdersSnapshot(ctx context.Context, ro *spanner.ReadOnlyTransaction, cellID, sid string, objs *[]*OntologyObject, edges *[]*OntologyGraphEdge) error {
	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT OrderId, SupplierId, RetailerId, COALESCE(WarehouseId, ''), COALESCE(DriverId, ''),
		             COALESCE(VehicleId, ''), COALESCE(ManifestId, ''), Status, OrderSource, ConfirmationStatus,
		             LineItemsJson, TotalMinor, OriginalTotalMinor, Currency, DeliveryFeeMinor,
		             COALESCE(FiscalStatus, ''), Version
		      FROM Orders WHERE SupplierId = @sid`,
		Params: map[string]any{"sid": sid},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		var oid, supID, rid, wid, did, vid, mid, status, source, confStatus, cur, fiscalStatus string
		var lineItemsRaw []byte
		var totalMinor, origTotalMinor, feeMinor, ver int64
		if err := row.Columns(&oid, &supID, &rid, &wid, &did, &vid, &mid, &status, &source, &confStatus, &lineItemsRaw, &totalMinor, &origTotalMinor, &cur, &feeMinor, &fiscalStatus, &ver); err != nil {
			return err
		}
		*objs = append(*objs, &OntologyObject{
			CellID:           cellID,
			ObjectType:       TypeOrder,
			ObjectID:         oid,
			SupplierID:       supID,
			Version:          ver,
			SecurityMarkings: []string{MarkingFinanceOnly},
			Properties: map[string]any{
				"order_id":             oid,
				"supplier_id":          supID,
				"retailer_id":          rid,
				"warehouse_id":         wid,
				"driver_id":            did,
				"vehicle_id":           vid,
				"manifest_id":          mid,
				"status":               status,
				"order_source":         source,
				"confirmation_status":  confStatus,
				"total_minor":          totalMinor,
				"original_total_minor": origTotalMinor,
				"delivery_fee_minor":   feeMinor,
				"currency":             cur,
				"fiscal_status":        fiscalStatus,
			},
		})
		if rid != "" {
			*edges = append(*edges, &OntologyGraphEdge{
				CellID: cellID, SourceObjectType: TypeShop, SourceObjectID: rid,
				LinkType: LinkOrdered, TargetObjectType: TypeOrder, TargetObjectID: oid,
				SupplierID: supID, CreatedByActionID: "rebuild-from-spanner",
			})
		}
		if wid != "" {
			*edges = append(*edges, &OntologyGraphEdge{
				CellID: cellID, SourceObjectType: TypeWarehouse, SourceObjectID: wid,
				LinkType: LinkSupplies, TargetObjectType: TypeOrder, TargetObjectID: oid,
				SupplierID: supID, CreatedByActionID: "rebuild-from-spanner",
			})
		}
		if mid != "" {
			*edges = append(*edges, &OntologyGraphEdge{
				CellID: cellID, SourceObjectType: TypeOrder, SourceObjectID: oid,
				LinkType: LinkShippedOn, TargetObjectType: TypeManifest, TargetObjectID: mid,
				SupplierID: supID, CreatedByActionID: "rebuild-from-spanner",
			})
		}
		if len(lineItemsRaw) > 0 {
			var lines []orderLineSnapshotDTO
			if err := json.Unmarshal(lineItemsRaw, &lines); err == nil {
				for idx, li := range lines {
					if strings.TrimSpace(li.SKU) == "" {
						continue
					}
					lineID := fmt.Sprintf("%s:%s:%d", oid, li.SKU, idx)
					if len(lineID) > 64 {
						lineID = fmt.Sprintf("%s:%d", oid, idx)
					}
					*objs = append(*objs, &OntologyObject{
						CellID:           cellID,
						ObjectType:       TypeOrderLine,
						ObjectID:         lineID,
						SupplierID:       supID,
						Version:          ver,
						SecurityMarkings: []string{MarkingFinanceOnly},
						Properties: map[string]any{
							"order_id":         oid,
							"sku":              li.SKU,
							"name":             li.Name,
							"quantity":         li.Quantity,
							"unit_price_minor": li.UnitPrice,
							"delivered_qty":    li.DeliveredQty,
							"remaining_qty":    li.RemainingQty,
							"offload_status":   li.OffloadStatus,
							"substitute_sku":   li.SubstituteSKU,
							"original_sku":     li.OriginalSKU,
							"price_gap_minor":  li.PriceGapMinor,
							"currency":         cur,
						},
					})
					*edges = append(*edges,
						&OntologyGraphEdge{
							CellID: cellID, SourceObjectType: TypeOrder, SourceObjectID: oid,
							LinkType: LinkOrdered, TargetObjectType: TypeOrderLine, TargetObjectID: lineID,
							SupplierID: supID, CreatedByActionID: "rebuild-from-spanner",
						},
						&OntologyGraphEdge{
							CellID: cellID, SourceObjectType: TypeOrderLine, SourceObjectID: lineID,
							LinkType: LinkOrdered, TargetObjectType: TypeSKU, TargetObjectID: li.SKU,
							SupplierID: supID, CreatedByActionID: "rebuild-from-spanner",
						},
					)
					if li.SubstituteSKU != "" && li.OriginalSKU != "" && li.SubstituteSKU != li.OriginalSKU {
						*edges = append(*edges, &OntologyGraphEdge{
							CellID: cellID, SourceObjectType: TypeSKU, SourceObjectID: li.SubstituteSKU,
							LinkType: LinkSubstitutes, TargetObjectType: TypeSKU, TargetObjectID: li.OriginalSKU,
							SupplierID: supID, CreatedByActionID: "rebuild-from-spanner",
						})
					}
				}
			}
		}
	}
	return nil
}

func loadManifestsSnapshot(ctx context.Context, ro *spanner.ReadOnlyTransaction, cellID, sid string, objs *[]*OntologyObject, edges *[]*OntologyGraphEdge) error {
	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT ManifestId, SupplierId, COALESCE(WarehouseId, ''), COALESCE(RouteId, ''), TruckId, DriverId, State, StopCount, ReplanCount
		      FROM SupplierTruckManifests WHERE SupplierId = @sid`,
		Params: map[string]any{"sid": sid},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		var mid, supID, wid, routeID, truckID, driverID, state string
		var stopCount, replanCount int64
		if err := row.Columns(&mid, &supID, &wid, &routeID, &truckID, &driverID, &state, &stopCount, &replanCount); err != nil {
			return err
		}
		*objs = append(*objs, &OntologyObject{
			CellID:     cellID,
			ObjectType: TypeManifest,
			ObjectID:   mid,
			SupplierID: supID,
			Properties: map[string]any{
				"manifest_id":  mid,
				"supplier_id":  supID,
				"warehouse_id": wid,
				"route_id":     routeID,
				"vehicle_id":   truckID,
				"driver_id":    driverID,
				"state":        state,
				"stop_count":   stopCount,
				"replan_count": replanCount,
			},
		})
		if truckID != "" {
			*edges = append(*edges, &OntologyGraphEdge{
				CellID: cellID, SourceObjectType: TypeManifest, SourceObjectID: mid,
				LinkType: LinkShippedOn, TargetObjectType: TypeVehicle, TargetObjectID: truckID,
				SupplierID: supID, CreatedByActionID: "rebuild-from-spanner",
			})
		}
		if driverID != "" {
			*edges = append(*edges, &OntologyGraphEdge{
				CellID: cellID, SourceObjectType: TypeManifest, SourceObjectID: mid,
				LinkType: LinkShippedOn, TargetObjectType: TypeDriver, TargetObjectID: driverID,
				SupplierID: supID, CreatedByActionID: "rebuild-from-spanner",
			})
		}
	}

	moIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT mo.ManifestId, mo.OrderId, mo.SequenceIndex, mo.State
		      FROM ManifestOrders mo
		      JOIN SupplierTruckManifests m ON mo.ManifestId = m.ManifestId
		      WHERE m.SupplierId = @sid`,
		Params: map[string]any{"sid": sid},
	})
	defer moIter.Stop()
	for {
		row, err := moIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		var mid, oid, state string
		var seqIdx int64
		if err := row.Columns(&mid, &oid, &seqIdx, &state); err != nil {
			return err
		}
		*edges = append(*edges, &OntologyGraphEdge{
			CellID: cellID, SourceObjectType: TypeOrder, SourceObjectID: oid,
			LinkType: LinkShippedOn, TargetObjectType: TypeManifest, TargetObjectID: mid,
			SupplierID: sid, CreatedByActionID: "rebuild-from-spanner",
		})
	}
	return nil
}

func loadArInvoicesSnapshot(ctx context.Context, ro *spanner.ReadOnlyTransaction, cellID, sid string, objs *[]*OntologyObject, edges *[]*OntologyGraphEdge) error {
	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT InvoiceId, SupplierId, RetailerId, OrderId, Status, PrincipalMinor, BalanceMinor, Currency,
		             TermsDays, GracePeriodDays, COALESCE(AgingBucket, ''), DunningStep, Version
		      FROM ArInvoices WHERE SupplierId = @sid`,
		Params: map[string]any{"sid": sid},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		var invID, supID, rid, oid, status, cur, bucket string
		var principal, balance, terms, grace, dunning, ver int64
		if err := row.Columns(&invID, &supID, &rid, &oid, &status, &principal, &balance, &cur, &terms, &grace, &bucket, &dunning, &ver); err != nil {
			return err
		}
		*objs = append(*objs, &OntologyObject{
			CellID:           cellID,
			ObjectType:       TypeNasiyaInvoice,
			ObjectID:         invID,
			SupplierID:       supID,
			Version:          ver,
			SecurityMarkings: []string{MarkingFinanceOnly},
			Properties: map[string]any{
				"invoice_id":        invID,
				"supplier_id":       supID,
				"retailer_id":       rid,
				"order_id":          oid,
				"status":            status,
				"principal_minor":   principal,
				"balance_minor":     balance,
				"currency":          cur,
				"terms_days":        terms,
				"grace_period_days": grace,
				"aging_bucket":      bucket,
				"dunning_step":      dunning,
			},
		})
		if rid != "" {
			*edges = append(*edges, &OntologyGraphEdge{
				CellID: cellID, SourceObjectType: TypeShop, SourceObjectID: rid,
				LinkType: LinkOwes, TargetObjectType: TypeNasiyaInvoice, TargetObjectID: invID,
				SupplierID: supID, CreatedByActionID: "rebuild-from-spanner",
			})
		}
		if oid != "" {
			*edges = append(*edges, &OntologyGraphEdge{
				CellID: cellID, SourceObjectType: TypeOrder, SourceObjectID: oid,
				LinkType: LinkOwes, TargetObjectType: TypeNasiyaInvoice, TargetObjectID: invID,
				SupplierID: supID, CreatedByActionID: "rebuild-from-spanner",
			})
		}
	}
	return nil
}

func loadCreditLinesSnapshot(ctx context.Context, ro *spanner.ReadOnlyTransaction, cellID, sid string, objs *[]*OntologyObject, edges *[]*OntologyGraphEdge) error {
	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT RetailerId, SupplierId, CreditLimitMinor, CurrentBalanceMinor, ReservedMinor,
		             AvailableCreditMinor, RiskScore, DelinquencyCount, Status, Version
		      FROM RetailerCreditProfiles WHERE SupplierId = @sid`,
		Params: map[string]any{"sid": sid},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		var rid, supID, status string
		var limitM, balM, resM, availM, risk, delinq, ver int64
		if err := row.Columns(&rid, &supID, &limitM, &balM, &resM, &availM, &risk, &delinq, &status, &ver); err != nil {
			return err
		}
		clID := rid + ":" + supID
		if len(clID) > 64 {
			clID = rid
		}
		*objs = append(*objs, &OntologyObject{
			CellID:           cellID,
			ObjectType:       TypeCreditLine,
			ObjectID:         clID,
			SupplierID:       supID,
			Version:          ver,
			SecurityMarkings: []string{MarkingFinanceOnly},
			Properties: map[string]any{
				"credit_line_id":         clID,
				"retailer_id":            rid,
				"supplier_id":            supID,
				"credit_limit_minor":     limitM,
				"current_balance_minor":  balM,
				"reserved_minor":         resM,
				"available_credit_minor": availM,
				"risk_score":             risk,
				"delinquency_count":      delinq,
				"status":                 status,
			},
		})
		*edges = append(*edges, &OntologyGraphEdge{
			CellID: cellID, SourceObjectType: TypeShop, SourceObjectID: rid,
			LinkType: LinkOwes, TargetObjectType: TypeCreditLine, TargetObjectID: clID,
			SupplierID: supID, CreatedByActionID: "rebuild-from-spanner",
		})
	}
	return nil
}

func loadPaymentsSnapshot(ctx context.Context, ro *spanner.ReadOnlyTransaction, cellID, sid string, objs *[]*OntologyObject, edges *[]*OntologyGraphEdge) error {
	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT SessionId, OrderId, SupplierId, RetailerId, Gateway, Currency, AmountMinor, Mode, Status
		      FROM PaymentSessions WHERE SupplierId = @sid`,
		Params: map[string]any{"sid": sid},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		var sessID, oid, supID, rid, gw, cur, mode, status string
		var amtMinor int64
		if err := row.Columns(&sessID, &oid, &supID, &rid, &gw, &cur, &amtMinor, &mode, &status); err != nil {
			return err
		}
		*objs = append(*objs, &OntologyObject{
			CellID:           cellID,
			ObjectType:       TypePayment,
			ObjectID:         sessID,
			SupplierID:       supID,
			SecurityMarkings: []string{MarkingFinanceOnly},
			Properties: map[string]any{
				"payment_id":   sessID,
				"order_id":     oid,
				"supplier_id":  supID,
				"retailer_id":  rid,
				"gateway":      gw,
				"currency":     cur,
				"amount_minor": amtMinor,
				"mode":         mode,
				"status":       status,
			},
		})
		if oid != "" {
			*edges = append(*edges, &OntologyGraphEdge{
				CellID: cellID, SourceObjectType: TypePayment, SourceObjectID: sessID,
				LinkType: LinkOrdered, TargetObjectType: TypeOrder, TargetObjectID: oid,
				SupplierID: supID, CreatedByActionID: "rebuild-from-spanner",
			})
		}
	}
	return nil
}

func loadFiscalDocsSnapshot(ctx context.Context, ro *spanner.ReadOnlyTransaction, cellID, sid string, objs *[]*OntologyObject, edges *[]*OntologyGraphEdge) error {
	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT OrderId, AttemptId, SupplierId, COALESCE(RetailerId, ''), Provider, Status,
		             COALESCE(FiscalReceiptId, ''), COALESCE(FiscalQR, ''), AmountMinor, Currency, COALESCE(PaymentMethod, '')
		      FROM OrderFiscalReceipts WHERE SupplierId = @sid`,
		Params: map[string]any{"sid": sid},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		var oid, attID, supID, rid, prov, status, recID, qr, cur, pm string
		var amtMinor int64
		if err := row.Columns(&oid, &attID, &supID, &rid, &prov, &status, &recID, &qr, &amtMinor, &cur, &pm); err != nil {
			return err
		}
		docID := recID
		if docID == "" {
			docID = oid + ":" + attID
		}
		if len(docID) > 64 {
			docID = attID
		}
		*objs = append(*objs, &OntologyObject{
			CellID:           cellID,
			ObjectType:       TypeFiscalDoc,
			ObjectID:         docID,
			SupplierID:       supID,
			SecurityMarkings: []string{MarkingFinanceOnly},
			Properties: map[string]any{
				"fiscal_doc_id":     docID,
				"order_id":          oid,
				"attempt_id":        attID,
				"supplier_id":       supID,
				"retailer_id":       rid,
				"provider":          prov,
				"status":            status,
				"fiscal_receipt_id": recID,
				"fiscal_qr":         qr,
				"amount_minor":      amtMinor,
				"currency":          cur,
				"payment_method":    pm,
			},
		})
		*edges = append(*edges, &OntologyGraphEdge{
			CellID: cellID, SourceObjectType: TypeFiscalDoc, SourceObjectID: docID,
			LinkType: LinkOrdered, TargetObjectType: TypeOrder, TargetObjectID: oid,
			SupplierID: supID, CreatedByActionID: "rebuild-from-spanner",
		})
	}
	return nil
}

func loadProposalsSnapshot(ctx context.Context, ro *spanner.ReadOnlyTransaction, cellID, sid string, objs *[]*OntologyObject, edges *[]*OntologyGraphEdge) error {
	negIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT p.ProposalId, p.OrderId, p.DriverId, p.Status, COALESCE(p.Resolution, ''), o.SupplierId
		      FROM NegotiationProposals p
		      JOIN Orders o ON p.OrderId = o.OrderId
		      WHERE o.SupplierId = @sid`,
		Params: map[string]any{"sid": sid},
	})
	defer negIter.Stop()
	for {
		row, err := negIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		var pid, oid, did, status, res, supID string
		if err := row.Columns(&pid, &oid, &did, &status, &res, &supID); err != nil {
			return err
		}
		*objs = append(*objs, &OntologyObject{
			CellID:     cellID,
			ObjectType: TypeProposal,
			ObjectID:   pid,
			SupplierID: supID,
			Properties: map[string]any{
				"proposal_id":   pid,
				"proposal_kind": "NEGOTIATION",
				"order_id":      oid,
				"driver_id":     did,
				"status":        status,
				"resolution":    res,
			},
		})
		*edges = append(*edges, &OntologyGraphEdge{
			CellID: cellID, SourceObjectType: TypeProposal, SourceObjectID: pid,
			LinkType: LinkOrdered, TargetObjectType: TypeOrder, TargetObjectID: oid,
			SupplierID: supID, CreatedByActionID: "rebuild-from-spanner",
		})
	}

	insIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT InsightId, WarehouseId, ProductId, SupplierId, CurrentStock, SuggestedQuantity, UrgencyLevel, ReasonCode, Status
		      FROM ReplenishmentInsights WHERE SupplierId = @sid`,
		Params: map[string]any{"sid": sid},
	})
	defer insIter.Stop()
	for {
		row, err := insIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		var iid, wid, prodID, supID, urgency, reason, status string
		var curStock, sugQty int64
		if err := row.Columns(&iid, &wid, &prodID, &supID, &curStock, &sugQty, &urgency, &reason, &status); err != nil {
			return err
		}
		*objs = append(*objs, &OntologyObject{
			CellID:     cellID,
			ObjectType: TypeProposal,
			ObjectID:   iid,
			SupplierID: supID,
			Properties: map[string]any{
				"proposal_id":        iid,
				"proposal_kind":      "REPLENISHMENT",
				"warehouse_id":       wid,
				"product_id":         prodID,
				"current_stock":      curStock,
				"suggested_quantity": sugQty,
				"urgency_level":      urgency,
				"reason_code":        reason,
				"status":             status,
			},
		})
		*edges = append(*edges, &OntologyGraphEdge{
			CellID: cellID, SourceObjectType: TypeProposal, SourceObjectID: iid,
			LinkType: LinkStocks, TargetObjectType: TypeSKU, TargetObjectID: prodID,
			SupplierID: supID, CreatedByActionID: "rebuild-from-spanner",
		})
	}
	return nil
}

// sortNeighborsDeterministic ensures neighbor nodes return in consistent order (hop, type, id).
func sortNeighborsDeterministic(nodes []NeighborNode) {
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Hop != nodes[j].Hop {
			return nodes[i].Hop < nodes[j].Hop
		}
		if nodes[i].Object.ObjectType != nodes[j].Object.ObjectType {
			return nodes[i].Object.ObjectType < nodes[j].Object.ObjectType
		}
		return nodes[i].Object.ObjectID < nodes[j].Object.ObjectID
	})
}
