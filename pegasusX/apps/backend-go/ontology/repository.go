package ontology

import (
	"context"
	"time"
)

// MaxChunkMutations caps mutations per Cloud Spanner ReadWriteTransaction cleanly
// below Spanner's 1,000-mutation application ceiling (UA-12).
const MaxChunkMutations = 900

// UpsertOptions controls monotonic version checks and outbox emission on batch upserts.
type UpsertOptions struct {
	// StrictMonotonic skips overwriting an existing object whenever existing.Version >= incoming.Version
	// (when incoming.Version > 0). When incoming.Version <= 0, existing objects bump Version+1 if changed
	// (or start at Version=1 if new).
	StrictMonotonic bool
	// EmitOutbox buffers events.EventOntologyObjectChanged -> events.TopicIntel atomically
	// inside the same persistence transaction for every created/updated object.
	EmitOutbox bool
	// SkipNoOpIfUnchanged avoids version churn during nightly RebuildFromSpanner when PropertiesJSON
	// and SecurityMarkings match the persisted row.
	SkipNoOpIfUnchanged bool
	ChangeKind          string
	SourceEventType     string
	SourceEventID       string
}

// Repository is the persistence seam for OntologyObjects and OntologyGraphEdges.
type Repository interface {
	UpsertBatch(ctx context.Context, objs []*OntologyObject, edges []*OntologyGraphEdge, opts UpsertOptions) (UpsertStats, error)
	GetObject(ctx context.Context, cellID, objectType, objectID string) (*OntologyObject, error)
	ListObjects(ctx context.Context, filter ListObjectsFilter) ([]*OntologyObject, int, error)
	GetNeighbors(ctx context.Context, cellID, supplierID, objectType, objectID string, depth int, linkTypes []string) (*OntologyObject, []NeighborNode, []*OntologyGraphEdge, error)
	GetGraphForSupplier(ctx context.Context, cellID, supplierID string) ([]*OntologyObject, []*OntologyGraphEdge, error)
}

// RebuildSource optionally supplies a live domain-table snapshot (implemented by SpannerRepository).
type RebuildSource interface {
	LoadDomainSnapshot(ctx context.Context, cellID, supplierID string) ([]*OntologyObject, []*OntologyGraphEdge, []string, error)
	ListActiveSupplierIDs(ctx context.Context) ([]string, error)
}

// OutboxBufferRecord captures an emitted ontology change event in memory/tests.
type OutboxBufferRecord struct {
	Topic       string
	AggregateID string
	EventType   string
	PayloadJSON string
	CreatedAt   time.Time
}

type objKey struct {
	cellID     string
	objectType string
	objectID   string
}

type edgeKey struct {
	cellID   string
	srcType  string
	srcID    string
	linkType string
	dstType  string
	dstID    string
}

// DeduplicateObjects deduplicates objects by primary key (CellId, ObjectType, ObjectId),
// preserving order and keeping the latest instance (or highest positive Version when monotonic).
func DeduplicateObjects(objs []*OntologyObject) []*OntologyObject {
	if len(objs) == 0 {
		return nil
	}
	seen := make(map[objKey]int, len(objs))
	out := make([]*OntologyObject, 0, len(objs))
	for _, o := range objs {
		if o == nil {
			continue
		}
		k := objKey{
			cellID:     NormalizeCellID(o.CellID),
			objectType: o.ObjectType,
			objectID:   o.ObjectID,
		}
		if idx, ok := seen[k]; ok {
			prev := out[idx]
			if prev.Version > 0 && o.Version > 0 && prev.Version > o.Version {
				continue
			}
			out[idx] = o
			continue
		}
		seen[k] = len(out)
		out = append(out, o)
	}
	return out
}

// DeduplicateEdges deduplicates edges by primary key
// (CellId, SourceObjectType, SourceObjectId, LinkType, TargetObjectType, TargetObjectId).
func DeduplicateEdges(edges []*OntologyGraphEdge) []*OntologyGraphEdge {
	if len(edges) == 0 {
		return nil
	}
	seen := make(map[edgeKey]int, len(edges))
	out := make([]*OntologyGraphEdge, 0, len(edges))
	for _, e := range edges {
		if e == nil {
			continue
		}
		k := edgeKey{
			cellID:   NormalizeCellID(e.CellID),
			srcType:  e.SourceObjectType,
			srcID:    e.SourceObjectID,
			linkType: e.LinkType,
			dstType:  e.TargetObjectType,
			dstID:    e.TargetObjectID,
		}
		if idx, ok := seen[k]; ok {
			out[idx] = e
			continue
		}
		seen[k] = len(out)
		out = append(out, e)
	}
	return out
}
