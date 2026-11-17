package ontology

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

// MemoryRepository is a thread-safe in-memory Repository for unit tests and ALLOW_MEMORY_FALLBACK.
type MemoryRepository struct {
	mu      sync.RWMutex
	now     func() time.Time
	objects map[objKey]*OntologyObject
	edges   map[edgeKey]*OntologyGraphEdge
	outbox  []OutboxBufferRecord
}

// NewMemoryRepository constructs an in-memory ontology store.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		now:     func() time.Time { return time.Now().UTC() },
		objects: make(map[objKey]*OntologyObject),
		edges:   make(map[edgeKey]*OntologyGraphEdge),
	}
}

// OutboxRecords returns a snapshot of emitted outbox events (useful in unit tests).
func (r *MemoryRepository) OutboxRecords() []OutboxBufferRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return slices.Clone(r.outbox)
}

// UpsertBatch persists objects and edges monotonically inside <=900-mutation chunks.
func (r *MemoryRepository) UpsertBatch(ctx context.Context, objs []*OntologyObject, edges []*OntologyGraphEdge, opts UpsertOptions) (UpsertStats, error) {
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

	r.mu.Lock()
	defer r.mu.Unlock()

	var stats UpsertStats
	now := r.now()

	objPerBatch := MaxChunkMutations
	if opts.EmitOutbox {
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
		stats.BatchesCommitted++
		for _, incoming := range dedupObjs[start:end] {
			k := objKey{
				cellID:     NormalizeCellID(incoming.CellID),
				objectType: incoming.ObjectType,
				objectID:   incoming.ObjectID,
			}
			existing, exists := r.objects[k]
			if exists && incoming.Version > 0 && existing.Version >= incoming.Version {
				stats.ObjectsSkipped++
				continue
			}

			cp := incoming.Clone()
			cp.CellID = k.cellID
			if cp.SupplierID == "" && exists {
				cp.SupplierID = existing.SupplierID
			}
			if exists {
				cp.ValidFrom = existing.ValidFrom
				cp.SecurityMarkings = MergeMarkings(existing.SecurityMarkings, cp.SecurityMarkings)
				if opts.SkipNoOpIfUnchanged && incoming.Version <= 0 &&
					existing.PropertiesJSON == cp.PropertiesJSON &&
					slices.Equal(existing.SecurityMarkings, cp.SecurityMarkings) {
					stats.ObjectsSkipped++
					continue
				}
				if cp.Version <= 0 {
					cp.Version = existing.Version + 1
				}
			} else {
				cp.ValidFrom = now
				if cp.Version <= 0 {
					cp.Version = 1
				}
			}
			cp.UpdatedAt = now
			if err := cp.PrepareForSpanner(); err != nil {
				return stats, err
			}
			r.objects[k] = cp
			stats.ObjectsUpserted++

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
						Version:   cp.Version,
						Timestamp: now.Format(time.RFC3339Nano),
					},
					CellID:           cp.CellID,
					ObjectType:       cp.ObjectType,
					ObjectID:         cp.ObjectID,
					SupplierID:       cp.SupplierID,
					Version:          cp.Version,
					ChangeKind:       changeKind,
					SourceEventType:  opts.SourceEventType,
					SourceEventID:    opts.SourceEventID,
					SecurityMarkings: slices.Clone(cp.SecurityMarkings),
					UpdatedAt:        now.Format(time.RFC3339Nano),
				}
				raw, _ := json.Marshal(ev)
				r.outbox = append(r.outbox, OutboxBufferRecord{
					Topic:       events.TopicIntel,
					AggregateID: fmt.Sprintf("%s/%s/%s", cp.CellID, cp.ObjectType, cp.ObjectID),
					EventType:   events.EventOntologyObjectChanged,
					PayloadJSON: string(raw),
					CreatedAt:   now,
				})
			}
		}
	}

	for start := 0; start < len(dedupEdges); start += MaxChunkMutations {
		end := start + MaxChunkMutations
		if end > len(dedupEdges) {
			end = len(dedupEdges)
		}
		stats.BatchesCommitted++
		for _, edge := range dedupEdges[start:end] {
			k := edgeKey{
				cellID:   NormalizeCellID(edge.CellID),
				srcType:  edge.SourceObjectType,
				srcID:    edge.SourceObjectID,
				linkType: edge.LinkType,
				dstType:  edge.TargetObjectType,
				dstID:    edge.TargetObjectID,
			}
			cp := &OntologyGraphEdge{
				CellID:            k.cellID,
				SourceObjectType:  edge.SourceObjectType,
				SourceObjectID:    edge.SourceObjectID,
				LinkType:          edge.LinkType,
				TargetObjectType:  edge.TargetObjectType,
				TargetObjectID:    edge.TargetObjectID,
				SupplierID:        edge.SupplierID,
				CreatedByActionID: edge.CreatedByActionID,
				Metadata:          cloneStringMap(edge.Metadata),
				MetadataJSON:      edge.MetadataJSON,
				ValidFrom:         now,
				ValidTo:           edge.ValidTo,
			}
			if existing, ok := r.edges[k]; ok {
				cp.ValidFrom = existing.ValidFrom
			}
			_ = cp.PrepareForSpanner()
			r.edges[k] = cp
			stats.EdgesUpserted++
		}
	}

	return stats, nil
}

// GetObject retrieves a single OntologyObject by primary key.
func (r *MemoryRepository) GetObject(ctx context.Context, cellID, objectType, objectID string) (*OntologyObject, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	obj, ok := r.objects[objKey{
		cellID:     NormalizeCellID(cellID),
		objectType: strings.TrimSpace(objectType),
		objectID:   strings.TrimSpace(objectID),
	}]
	if !ok {
		return nil, ErrObjectNotFound
	}
	return obj.Clone(), nil
}

// ListObjects filters and paginates OntologyObjects deterministically.
func (r *MemoryRepository) ListObjects(ctx context.Context, filter ListObjectsFilter) ([]*OntologyObject, int, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	cellID := NormalizeCellID(filter.CellID)
	supID := strings.TrimSpace(filter.SupplierID)
	objType := strings.TrimSpace(filter.ObjectType)
	marking := strings.TrimSpace(filter.Marking)
	search := strings.ToLower(strings.TrimSpace(filter.Search))

	r.mu.RLock()
	matched := make([]*OntologyObject, 0)
	for _, o := range r.objects {
		if o.CellID != cellID {
			continue
		}
		if supID != "" && o.SupplierID != supID {
			continue
		}
		if objType != "" && !strings.EqualFold(o.ObjectType, objType) {
			continue
		}
		if marking != "" && !slices.Contains(o.SecurityMarkings, marking) {
			continue
		}
		if search != "" {
			haystack := strings.ToLower(o.ObjectID + " " + o.ObjectType + " " + o.PropertiesJSON)
			if !strings.Contains(haystack, search) {
				continue
			}
		}
		matched = append(matched, o.Clone())
	}
	r.mu.RUnlock()

	sort.Slice(matched, func(i, j int) bool {
		if matched[i].ObjectType != matched[j].ObjectType {
			return matched[i].ObjectType < matched[j].ObjectType
		}
		return matched[i].ObjectID < matched[j].ObjectID
	})

	total := len(matched)
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
	if offset >= total {
		return []*OntologyObject{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return matched[offset:end], total, nil
}

// GetNeighbors performs bounded BFS up to depth (1..4) across outbound and inbound edges.
func (r *MemoryRepository) GetNeighbors(ctx context.Context, cellID, supplierID, objectType, objectID string, depth int, linkTypes []string) (*OntologyObject, []NeighborNode, []*OntologyGraphEdge, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, nil, err
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
	linkFilter := make(map[string]bool, len(linkTypes))
	for _, lt := range linkTypes {
		lt = strings.TrimSpace(lt)
		if lt != "" {
			linkFilter[lt] = true
		}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	rootKey := objKey{cellID: cellID, objectType: objectType, objectID: objectID}
	rootObj, ok := r.objects[rootKey]
	if !ok {
		return nil, nil, nil, ErrObjectNotFound
	}
	if supID != "" && rootObj.SupplierID != supID {
		return nil, nil, nil, ErrObjectNotFound
	}

	type queueItem struct {
		key objKey
		hop int
	}

	visited := map[objKey]bool{rootKey: true}
	queue := []queueItem{{key: rootKey, hop: 0}}
	edgeSeen := make(map[edgeKey]bool)
	var nodes []NeighborNode
	var edges []*OntologyGraphEdge

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		if curr.hop >= depth {
			continue
		}

		for ek, e := range r.edges {
			if ek.cellID != cellID || e.ValidTo != nil {
				continue
			}
			if supID != "" && e.SupplierID != "" && e.SupplierID != supID {
				continue
			}
			if len(linkFilter) > 0 && !linkFilter[e.LinkType] {
				continue
			}

			var targetKey objKey
			var dir string
			if e.SourceObjectType == curr.key.objectType && e.SourceObjectID == curr.key.objectID {
				targetKey = objKey{cellID: cellID, objectType: e.TargetObjectType, objectID: e.TargetObjectID}
				dir = "OUTBOUND"
			} else if e.TargetObjectType == curr.key.objectType && e.TargetObjectID == curr.key.objectID {
				targetKey = objKey{cellID: cellID, objectType: e.SourceObjectType, objectID: e.SourceObjectID}
				dir = "INBOUND"
			} else {
				continue
			}

			if !edgeSeen[ek] {
				edgeSeen[ek] = true
				edges = append(edges, cloneEdge(e))
			}

			if !visited[targetKey] {
				visited[targetKey] = true
				if targetObj, exists := r.objects[targetKey]; exists {
					if supID == "" || targetObj.SupplierID == supID {
						nodes = append(nodes, NeighborNode{
							Object:      targetObj.Clone(),
							Hop:         curr.hop + 1,
							ViaLinkType: e.LinkType,
							Direction:   dir,
							ParentType:  curr.key.objectType,
							ParentID:    curr.key.objectID,
						})
					}
				}
				queue = append(queue, queueItem{key: targetKey, hop: curr.hop + 1})
			}
		}
	}

	return rootObj.Clone(), nodes, edges, nil
}

// GetGraphForSupplier returns all objects and active edges scoped to (cellID, supplierID).
func (r *MemoryRepository) GetGraphForSupplier(ctx context.Context, cellID, supplierID string) ([]*OntologyObject, []*OntologyGraphEdge, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	cellID = NormalizeCellID(cellID)
	supID := strings.TrimSpace(supplierID)

	r.mu.RLock()
	defer r.mu.RUnlock()

	var objs []*OntologyObject
	for _, o := range r.objects {
		if o.CellID == cellID && (supID == "" || o.SupplierID == supID) {
			objs = append(objs, o.Clone())
		}
	}
	var edges []*OntologyGraphEdge
	for _, e := range r.edges {
		if e.CellID == cellID && e.ValidTo == nil && (supID == "" || e.SupplierID == supID) {
			edges = append(edges, cloneEdge(e))
		}
	}
	sort.Slice(objs, func(i, j int) bool {
		if objs[i].ObjectType != objs[j].ObjectType {
			return objs[i].ObjectType < objs[j].ObjectType
		}
		return objs[i].ObjectID < objs[j].ObjectID
	})
	return objs, edges, nil
}

func cloneStringMap(src map[string]string) map[string]string {
	if src == nil {
		return nil
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func cloneEdge(e *OntologyGraphEdge) *OntologyGraphEdge {
	if e == nil {
		return nil
	}
	var vt *time.Time
	if e.ValidTo != nil {
		t := *e.ValidTo
		vt = &t
	}
	return &OntologyGraphEdge{
		CellID:            e.CellID,
		SourceObjectType:  e.SourceObjectType,
		SourceObjectID:    e.SourceObjectID,
		LinkType:          e.LinkType,
		TargetObjectType:  e.TargetObjectType,
		TargetObjectID:    e.TargetObjectID,
		SupplierID:        e.SupplierID,
		CreatedByActionID: e.CreatedByActionID,
		Metadata:          cloneStringMap(e.Metadata),
		MetadataJSON:      e.MetadataJSON,
		ValidFrom:         e.ValidFrom,
		ValidTo:           vt,
	}
}
