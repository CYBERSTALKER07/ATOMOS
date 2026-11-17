package supplier

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/entityresolution"
)

// SupplierRetailerLink re-exports the tenant-scoped SupplierRetailerLinks model.
type SupplierRetailerLink = entityresolution.SupplierRetailerLink

// NormalizeTIN deterministically normalizes tax identifiers using entityresolution rules.
func NormalizeTIN(raw string) string {
	return entityresolution.NormalizeTIN(raw)
}

// NormalizeUZPhone deterministically normalizes Uzbekistan (+998...) phone numbers.
func NormalizeUZPhone(raw string) string {
	return entityresolution.NormalizeUZPhone(raw)
}

// UpsertSupplierRetailerLink writes a tenant-scoped SupplierRetailerLinks row and clusters same-supplier TIN/phone duplicates.
func UpsertSupplierRetailerLink(ctx context.Context, client *spanner.Client, link SupplierRetailerLink, now time.Time) (SupplierRetailerLink, []SupplierRetailerLink, error) {
	return entityresolution.UpsertSupplierRetailerLink(ctx, client, link, now)
}

// ResolveSupplierRetailerLinkDedup resolves a DUPLICATE_CANDIDATE row inside SupplierId ("merge_into" or "mark_clean").
func ResolveSupplierRetailerLinkDedup(ctx context.Context, client *spanner.Client, supplierID, retailerID, action, mergedIntoID string, now time.Time) (SupplierRetailerLink, []SupplierRetailerLink, error) {
	return entityresolution.NewSpannerRetailerLinkRepository(client).Resolve(ctx, supplierID, retailerID, action, mergedIntoID, now)
}

type retailerDedupCluster struct {
	DedupClusterID string                 `json:"dedup_cluster_id"`
	Members        []SupplierRetailerLink `json:"members"`
}

func (s *Service) serviceNow() time.Time {
	if s != nil && s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}

func parseLimitParam(raw string, fallback int) int {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback
	}
	v, err := strconv.Atoi(trimmed)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}

// HandleRetailerLinks serves GET and POST /v1/supplier/retailer-links strictly scoped to the caller's SupplierId.
func (s *Service) HandleRetailerLinks(w http.ResponseWriter, r *http.Request) {
	sid := strings.TrimSpace(s.scopedSupplierID(r))
	if sid == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "supplier_id_required"})
		return
	}
	repo := s.RetailerLinks()
	if repo == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "retailer_links_unavailable"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		statusFilter := strings.TrimSpace(r.URL.Query().Get("status"))
		limit := parseLimitParam(r.URL.Query().Get("limit"), 100)
		links, err := repo.List(r.Context(), sid, statusFilter, limit)
		if err != nil {
			s.warnLog(r.Context(), "list supplier retailer links failed", "supplier_id", sid, "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "retailer_links_list_failed"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"supplier_id": sid,
			"links":       links,
		})

	case http.MethodPost:
		var body struct {
			RetailerID      string `json:"retailer_id"`
			TaxIDTIN        string `json:"tax_id_tin"`
			TIN             string `json:"tin"`
			Phone           string `json:"phone"`
			NormalizedPhone string `json:"normalized_phone"`
			LegalName       string `json:"legal_name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json_body"})
			return
		}
		rid := strings.TrimSpace(body.RetailerID)
		if rid == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "retailer_id_required"})
			return
		}
		tin := strings.TrimSpace(body.TaxIDTIN)
		if tin == "" {
			tin = strings.TrimSpace(body.TIN)
		}
		phone := strings.TrimSpace(body.NormalizedPhone)
		if phone == "" {
			phone = strings.TrimSpace(body.Phone)
		}
		saved, peers, err := repo.Upsert(r.Context(), SupplierRetailerLink{
			SupplierID:      sid,
			RetailerID:      rid,
			TaxIDTIN:        tin,
			NormalizedPhone: phone,
			LegalName:       strings.TrimSpace(body.LegalName),
		}, s.serviceNow())
		if err != nil {
			writeRetailerLinkError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"supplier_id":     sid,
			"link":            saved,
			"duplicate_peers": peers,
		})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

// HandleRetailerLinksDedup serves GET /v1/supplier/retailer-links/dedup strictly scoped to SupplierId.
func (s *Service) HandleRetailerLinksDedup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	sid := strings.TrimSpace(s.scopedSupplierID(r))
	if sid == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "supplier_id_required"})
		return
	}
	repo := s.RetailerLinks()
	if repo == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "retailer_links_unavailable"})
		return
	}
	limit := parseLimitParam(r.URL.Query().Get("limit"), 100)
	candidates, err := repo.List(r.Context(), sid, entityresolution.DedupStatusDuplicateCandidate, limit)
	if err != nil {
		s.warnLog(r.Context(), "list retailer dedup candidates failed", "supplier_id", sid, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "retailer_links_dedup_failed"})
		return
	}

	clusterMap := make(map[string][]SupplierRetailerLink)
	for _, item := range candidates {
		cid := strings.TrimSpace(item.DedupClusterID)
		if cid == "" {
			cid = "unclustered:" + item.RetailerID
		}
		clusterMap[cid] = append(clusterMap[cid], item)
	}
	clusters := make([]retailerDedupCluster, 0, len(clusterMap))
	for cid, members := range clusterMap {
		sort.Slice(members, func(i, j int) bool { return members[i].RetailerID < members[j].RetailerID })
		clusters = append(clusters, retailerDedupCluster{
			DedupClusterID: cid,
			Members:        members,
		})
	}
	sort.Slice(clusters, func(i, j int) bool { return clusters[i].DedupClusterID < clusters[j].DedupClusterID })

	writeJSON(w, http.StatusOK, map[string]any{
		"supplier_id": sid,
		"candidates":  candidates,
		"clusters":    clusters,
	})
}

// HandleResolveRetailerLink serves POST /v1/supplier/retailer-links/{retailerID}/resolve.
func (s *Service) HandleResolveRetailerLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	sid := strings.TrimSpace(s.scopedSupplierID(r))
	if sid == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "supplier_id_required"})
		return
	}
	rid := strings.TrimSpace(chi.URLParam(r, "retailerID"))
	if rid == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "retailer_id_required"})
		return
	}
	repo := s.RetailerLinks()
	if repo == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "retailer_links_unavailable"})
		return
	}
	var body struct {
		Action           string `json:"action"`
		MergedIntoID     string `json:"merged_into_id"`
		TargetRetailerID string `json:"target_retailer_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json_body"})
		return
	}
	targetID := strings.TrimSpace(body.MergedIntoID)
	if targetID == "" {
		targetID = strings.TrimSpace(body.TargetRetailerID)
	}
	action := strings.TrimSpace(body.Action)
	if action == "" && targetID != "" {
		action = "merge_into"
	}
	updated, peers, err := repo.Resolve(r.Context(), sid, rid, action, targetID, s.serviceNow())
	if err != nil {
		writeRetailerLinkError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":        "resolved",
		"supplier_id":   sid,
		"link":          updated,
		"updated_peers": peers,
	})
}

func (s *Service) warnLog(ctx context.Context, msg string, args ...any) {
	if s != nil && s.log != nil {
		s.log.WarnContext(ctx, msg, args...)
	}
}

func writeRetailerLinkError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, entityresolution.ErrRetailerLinkNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
	case errors.Is(err, entityresolution.ErrInvalidRetailerLinkInput),
		errors.Is(err, entityresolution.ErrRetailerMergeTargetNotFound):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "retailer_link_operation_failed"})
	}
}
