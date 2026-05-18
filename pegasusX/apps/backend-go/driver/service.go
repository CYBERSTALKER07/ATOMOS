// Package driver owns driver-role handlers and local scaffold state.
package driver

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// Service keeps additive in-memory driver state for scaffold routes.
type Service struct {
	mu                 sync.RWMutex
	availability       map[string]bool
	history            map[string][]HistoryRow
	earningsMinor      map[string]int64
	pendingCollections map[string][]PendingCollection
	manifestHashes     map[string][]string
	now                func() time.Time
}

// ServiceConfig is the constructor input.
type ServiceConfig struct {
	Now func() time.Time
}

// HistoryRow represents one completed or in-progress delivery history row.
type HistoryRow struct {
	OrderID     string `json:"order_id"`
	Status      string `json:"status"`
	TotalMinor  int64  `json:"total_minor"`
	Currency    string `json:"currency"`
	CompletedAt string `json:"completed_at,omitempty"`
}

// PendingCollection is one outstanding cash collection task.
type PendingCollection struct {
	OrderID     string `json:"order_id"`
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
	DueAt       string `json:"due_at"`
}

// NewService constructs the driver service.
func NewService(c ServiceConfig) *Service {
	if c.Now == nil {
		c.Now = func() time.Time { return time.Now().UTC() }
	}
	return &Service{
		availability:       make(map[string]bool),
		history:            make(map[string][]HistoryRow),
		earningsMinor:      make(map[string]int64),
		pendingCollections: make(map[string][]PendingCollection),
		manifestHashes:     make(map[string][]string),
		now:                c.Now,
	}
}

type availabilityPatchRequest struct {
	OnShift bool `json:"on_shift"`
}

// HandleProfile serves GET /v1/driver/profile.
func (s *Service) HandleProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	driverID := driverIDFromRequest(r)
	if driverID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	claims, _ := auth.FromContext(r.Context())
	s.mu.RLock()
	onShift := s.availability[driverID]
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"driver_id":      driverID,
		"home_node_type": claims.HomeNodeType,
		"home_node_id":   claims.HomeNodeID,
		"on_shift":       onShift,
		"updated_at":     s.now().Format(time.RFC3339Nano),
	})
}

// HandleHistory serves GET /v1/driver/history.
func (s *Service) HandleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	driverID := driverIDFromRequest(r)
	if driverID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	s.mu.RLock()
	rows := append([]HistoryRow(nil), s.history[driverID]...)
	s.mu.RUnlock()
	sort.Slice(rows, func(i, j int) bool { return rows[i].CompletedAt > rows[j].CompletedAt })
	writeJSON(w, http.StatusOK, map[string]any{"rows": rows})
}

// HandleEarnings serves GET /v1/driver/earnings.
func (s *Service) HandleEarnings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	driverID := driverIDFromRequest(r)
	if driverID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	s.mu.RLock()
	total := s.earningsMinor[driverID]
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"driver_id":   driverID,
		"currency":    "UZS",
		"today_minor": total,
		"week_minor":  total,
		"month_minor": total,
	})
}

// HandleAvailability serves GET/PATCH /v1/driver/availability.
func (s *Service) HandleAvailability(w http.ResponseWriter, r *http.Request) {
	driverID := driverIDFromRequest(r)
	if driverID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.mu.RLock()
		onShift := s.availability[driverID]
		s.mu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"driver_id": driverID, "on_shift": onShift})
	case http.MethodPatch:
		var req availabilityPatchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		defer r.Body.Close()
		s.mu.Lock()
		s.availability[driverID] = req.OnShift
		s.mu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{"driver_id": driverID, "on_shift": req.OnShift})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

// HandlePendingCollections serves GET /v1/driver/pending-collections.
func (s *Service) HandlePendingCollections(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	driverID := driverIDFromRequest(r)
	if driverID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	s.mu.RLock()
	pending := append([]PendingCollection(nil), s.pendingCollections[driverID]...)
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{"pending": pending})
}

// HandleManifestGate serves GET /v1/driver/manifest-gate.
func (s *Service) HandleManifestGate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	driverID := driverIDFromRequest(r)
	if driverID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"driver_id": driverID,
		"allowed":   true,
		"reason":    "ok",
	})
}

// HandleManifest serves GET /v1/driver/manifest and legacy GET /v1/fleet/manifest.
func (s *Service) HandleManifest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	driverID := driverIDFromRequest(r)
	if driverID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	s.mu.Lock()
	hashes := s.manifestHashes[driverID]
	if len(hashes) == 0 {
		hashes = []string{"sha256:demo-token-hash"}
		s.manifestHashes[driverID] = hashes
	}
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"driver_id": driverID,
		"date":      strings.TrimSpace(r.URL.Query().Get("date")),
		"hashes":    hashes,
	})
}

func driverIDFromRequest(r *http.Request) string {
	if claims, ok := auth.FromContext(r.Context()); ok {
		if strings.TrimSpace(claims.Subject) != "" {
			return strings.TrimSpace(claims.Subject)
		}
	}
	return strings.TrimSpace(r.URL.Query().Get("driver_id"))
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}
