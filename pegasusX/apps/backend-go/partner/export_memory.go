package partner

import (
	"context"
	"sync"
	"time"
)

// MemoryExportRepository for tests / local.
type MemoryExportRepository struct {
	mu   sync.RWMutex
	jobs map[string]ExportJob
}

func NewMemoryExportRepository() *MemoryExportRepository {
	return &MemoryExportRepository{jobs: map[string]ExportJob{}}
}

func (r *MemoryExportRepository) InsertJob(_ context.Context, j ExportJob) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.jobs[j.JobID] = j
	return nil
}

func (r *MemoryExportRepository) GetJob(_ context.Context, jobID string) (ExportJob, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	j, ok := r.jobs[jobID]
	return j, ok, nil
}

func (r *MemoryExportRepository) ListJobs(_ context.Context, tenantType, tenantID string, limit int) ([]ExportJob, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 {
		limit = 50
	}
	out := make([]ExportJob, 0)
	for _, j := range r.jobs {
		if j.TenantType == tenantType && j.TenantID == tenantID {
			out = append(out, j)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (r *MemoryExportRepository) ListPending(_ context.Context, limit int) ([]ExportJob, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 {
		limit = 20
	}
	out := make([]ExportJob, 0)
	for _, j := range r.jobs {
		if j.Status == ExportStatusPending {
			out = append(out, j)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (r *MemoryExportRepository) UpdateJob(_ context.Context, j ExportJob) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.jobs[j.JobID] = j
	return nil
}

// MemorySftpConfigRepository for tests / local.
type MemorySftpConfigRepository struct {
	mu   sync.RWMutex
	cfgs map[string]SftpConfig
}

func NewMemorySftpConfigRepository() *MemorySftpConfigRepository {
	return &MemorySftpConfigRepository{cfgs: map[string]SftpConfig{}}
}

func sftpKey(tenantType, tenantID string) string {
	return tenantType + "|" + tenantID
}

func (r *MemorySftpConfigRepository) Upsert(_ context.Context, c SftpConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	normalizeSftpDirs(&c)
	c.UpdatedAt = time.Now().UTC()
	r.cfgs[sftpKey(c.TenantType, c.TenantID)] = c
	return nil
}

func (r *MemorySftpConfigRepository) Get(_ context.Context, tenantType, tenantID string) (SftpConfig, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.cfgs[sftpKey(tenantType, tenantID)]
	if ok {
		normalizeSftpDirs(&c)
	}
	return c, ok, nil
}
