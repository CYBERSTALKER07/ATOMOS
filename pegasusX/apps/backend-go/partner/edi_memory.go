package partner

import (
	"context"
	"sync"
)

// MemoryEdiDocumentRepository for tests / local.
type MemoryEdiDocumentRepository struct {
	mu   sync.RWMutex
	docs map[string]EdiDocument
}

func NewMemoryEdiDocumentRepository() *MemoryEdiDocumentRepository {
	return &MemoryEdiDocumentRepository{docs: map[string]EdiDocument{}}
}

func ediIdemKey(tenantType, tenantID, direction, docType, external string) string {
	return tenantType + "|" + tenantID + "|" + direction + "|" + docType + "|" + external
}

func (r *MemoryEdiDocumentRepository) Insert(_ context.Context, d EdiDocument) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := ediIdemKey(d.TenantType, d.TenantID, d.Direction, d.DocType, d.ExternalDocID)
	for _, existing := range r.docs {
		if ediIdemKey(existing.TenantType, existing.TenantID, existing.Direction, existing.DocType, existing.ExternalDocID) == key {
			return errConflict("edi_document")
		}
	}
	r.docs[d.DocumentID] = d
	return nil
}

func (r *MemoryEdiDocumentRepository) Get(_ context.Context, documentID string) (EdiDocument, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.docs[documentID]
	return d, ok, nil
}

func (r *MemoryEdiDocumentRepository) GetByExternal(_ context.Context, tenantType, tenantID, direction, docType, externalDocID string) (EdiDocument, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := ediIdemKey(tenantType, tenantID, direction, docType, externalDocID)
	for _, d := range r.docs {
		if ediIdemKey(d.TenantType, d.TenantID, d.Direction, d.DocType, d.ExternalDocID) == key {
			return d, true, nil
		}
	}
	return EdiDocument{}, false, nil
}

func (r *MemoryEdiDocumentRepository) Update(_ context.Context, d EdiDocument) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.docs[d.DocumentID] = d
	return nil
}

func (r *MemoryEdiDocumentRepository) ListByTenant(_ context.Context, tenantType, tenantID string, limit int) ([]EdiDocument, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 {
		limit = 50
	}
	out := make([]EdiDocument, 0)
	for _, d := range r.docs {
		if d.TenantType == tenantType && d.TenantID == tenantID {
			out = append(out, d)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (r *MemoryEdiDocumentRepository) ListPendingOutbound(_ context.Context, limit int) ([]EdiDocument, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 {
		limit = 20
	}
	out := make([]EdiDocument, 0)
	for _, d := range r.docs {
		if d.Direction == EdiDirectionOut && d.Status == EdiStatusReceived {
			out = append(out, d)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (r *MemorySftpConfigRepository) ListEdiEnabled(_ context.Context, limit int) ([]SftpConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 {
		limit = 50
	}
	out := make([]SftpConfig, 0)
	for _, c := range r.cfgs {
		if c.IsActive && c.EdiEnabled {
			out = append(out, c)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

// errConflict is returned on unique-index collisions.
type conflictError struct{ what string }

func (e conflictError) Error() string { return "conflict:" + e.what }

func errConflict(what string) error { return conflictError{what: what} }
