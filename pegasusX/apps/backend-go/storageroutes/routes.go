package storageroutes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/web"
	"github.com/pegasusx/pegasusx/apps/backend-go/storage"
)

// VaultStore abstracts dossier persistence operations needed by storage routes.
type VaultStore interface {
	CreateDossier(ctx context.Context, targetID, targetType string) (*storage.EvidenceDossier, error)
	GetDossier(ctx context.Context, dossierID string) (*storage.EvidenceDossier, []*storage.EvidenceItem, error)
	AddItem(ctx context.Context, dossierID string, req *storage.EvidenceItem) (*storage.EvidenceItem, error)
	SealDossier(ctx context.Context, dossierID string) error
}

type AddItemRequest struct {
	ItemType  string  `json:"item_type"`
	FileHash  string  `json:"file_hash"`
	MimeType  string  `json:"mime_type"`
	SizeBytes int64   `json:"size_bytes"`
	Extension string  `json:"extension"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type CreateDossierRequest struct {
	TargetID   string `json:"target_id"`
	TargetType string `json:"target_type"`
}

func Mount(r chi.Router, vault *storage.Vault) {
	MountWithStore(r, vault)
}

// MountWithStore attaches dossier and evidence-upload routes using any VaultStore implementation.
func MountWithStore(r chi.Router, vault VaultStore) {
	r.Route("/dossiers", func(r chi.Router) {
		r.Use(auth.RequireRole(
			auth.RoleAdmin,
			auth.RoleWarehouseAdmin,
			auth.RoleWarehouse,
			auth.RoleDriver,
			auth.RolePlatformAdmin,
			auth.RoleRetailer,
		))

		r.Post("/", func(w http.ResponseWriter, req *http.Request) {
			if _, ok := auth.FromContext(req.Context()); !ok {
				web.JSONError(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			req.Body = http.MaxBytesReader(w, req.Body, 1<<20)
			var body CreateDossierRequest
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				web.JSONError(w, "invalid request", http.StatusBadRequest)
				return
			}
			if body.TargetID == "" || body.TargetType == "" {
				web.JSONError(w, "target_id and target_type are required", http.StatusBadRequest)
				return
			}

			dossier, err := vault.CreateDossier(req.Context(), body.TargetID, body.TargetType)
			if err != nil {
				web.JSONError(w, "failed to create dossier", http.StatusInternalServerError)
				return
			}
			web.JSONResponse(w, http.StatusCreated, dossier)
		})

		r.Get("/{dossier_id}", func(w http.ResponseWriter, req *http.Request) {
			if _, ok := auth.FromContext(req.Context()); !ok {
				web.JSONError(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			id := strings.TrimSpace(chi.URLParam(req, "dossier_id"))
			if !isSafePathSegment(id) {
				web.JSONError(w, "invalid dossier_id", http.StatusBadRequest)
				return
			}
			dossier, items, err := vault.GetDossier(req.Context(), id)
			if err != nil {
				if errors.Is(err, storage.ErrDossierNotFound) {
					web.JSONError(w, "dossier not found", http.StatusNotFound)
					return
				}
				web.JSONError(w, "failed to fetch dossier", http.StatusInternalServerError)
				return
			}
			web.JSONResponse(w, http.StatusOK, map[string]any{
				"dossier": dossier,
				"items":   items,
			})
		})

		r.Post("/{dossier_id}/items", func(w http.ResponseWriter, req *http.Request) {
			// Authenticate actor claims and resolve tenant scope BEFORE generating an upload ticket.
			claims, ok := auth.FromContext(req.Context())
			if !ok {
				web.JSONError(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			actorScope, uploaderID := resolveDossierActor(claims)
			if actorScope == "" || uploaderID == "" {
				web.JSONError(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			id := strings.TrimSpace(chi.URLParam(req, "dossier_id"))
			if !isSafePathSegment(id) {
				web.JSONError(w, "invalid dossier_id", http.StatusBadRequest)
				return
			}

			req.Body = http.MaxBytesReader(w, req.Body, 1<<20)
			var body AddItemRequest
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				web.JSONError(w, "invalid request", http.StatusBadRequest)
				return
			}
			if strings.TrimSpace(body.FileHash) == "" || strings.TrimSpace(body.ItemType) == "" {
				web.JSONError(w, "file_hash and item_type are required", http.StatusBadRequest)
				return
			}

			ext, expectedMIME, err := storage.NormalizeUploadExtension(body.Extension)
			if err != nil {
				web.JSONError(w, "unsupported extension", http.StatusBadRequest)
				return
			}
			mimeType := strings.TrimSpace(body.MimeType)
			if mimeType == "" {
				mimeType = expectedMIME
			} else if !strings.EqualFold(mimeType, expectedMIME) {
				web.JSONError(w, "mime_type does not match extension", http.StatusBadRequest)
				return
			}
			if err := storage.ValidateUploadSize(body.SizeBytes); err != nil {
				web.JSONError(w, "size_bytes must be between 1 and 10485760 bytes", http.StatusBadRequest)
				return
			}

			objectPrefix := fmt.Sprintf("claims/%s/%s", actorScope, id)
			uploadURL, publicURL, err := storage.GenerateUploadTicketFor(objectPrefix, ext)
			if err != nil {
				if errors.Is(err, storage.ErrInvalidUploadExtension) ||
					errors.Is(err, storage.ErrInvalidUploadPrefix) ||
					errors.Is(err, storage.ErrInvalidUploadSize) {
					web.JSONError(w, "invalid upload request", http.StatusBadRequest)
					return
				}
				web.JSONError(w, "storage unavailable", http.StatusServiceUnavailable)
				return
			}

			item := &storage.EvidenceItem{
				ItemType:       strings.TrimSpace(body.ItemType),
				StorageURI:     publicURL,
				FileHash:       strings.TrimSpace(body.FileHash),
				MimeType:       mimeType,
				SizeBytes:      body.SizeBytes,
				UploaderUserID: uploaderID,
			}
			if body.Latitude != 0 || body.Longitude != 0 {
				item.Latitude.Valid = true
				item.Latitude.Float64 = body.Latitude
				item.Longitude.Valid = true
				item.Longitude.Float64 = body.Longitude
			}

			added, err := vault.AddItem(req.Context(), id, item)
			if err != nil {
				if errors.Is(err, storage.ErrDossierSealed) {
					web.JSONError(w, "dossier is sealed", http.StatusConflict)
					return
				}
				if errors.Is(err, storage.ErrDossierNotFound) {
					web.JSONError(w, "dossier not found", http.StatusNotFound)
					return
				}
				web.JSONError(w, "failed to add item", http.StatusInternalServerError)
				return
			}

			web.JSONResponse(w, http.StatusCreated, map[string]any{
				"item":       added,
				"upload_url": uploadURL,
				"public_url": publicURL,
			})
		})

		r.Post("/{dossier_id}/seal", func(w http.ResponseWriter, req *http.Request) {
			if _, ok := auth.FromContext(req.Context()); !ok {
				web.JSONError(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			id := strings.TrimSpace(chi.URLParam(req, "dossier_id"))
			if !isSafePathSegment(id) {
				web.JSONError(w, "invalid dossier_id", http.StatusBadRequest)
				return
			}
			if err := vault.SealDossier(req.Context(), id); err != nil {
				if errors.Is(err, storage.ErrDossierNotFound) {
					web.JSONError(w, "dossier not found", http.StatusNotFound)
					return
				}
				if errors.Is(err, storage.ErrDossierSealed) {
					web.JSONError(w, "dossier is already sealed", http.StatusConflict)
					return
				}
				web.JSONError(w, "failed to seal dossier", http.StatusInternalServerError)
				return
			}
			web.JSONResponse(w, http.StatusOK, map[string]string{"status": "sealed"})
		})
	})
}

func resolveDossierActor(c auth.Claims) (tenantScope string, uploaderID string) {
	if c.Role == auth.RoleRetailer {
		orgID := sanitizeTenantSegment(auth.ResolveRetailerOrgID(c))
		userID := strings.TrimSpace(auth.ResolveRetailerUserID(c))
		if !isSafePathSegment(orgID) || userID == "" {
			return "", ""
		}
		return orgID, userID
	}
	userID := strings.TrimSpace(c.Subject)
	if userID == "" {
		userID = strings.TrimSpace(auth.ResolveRetailerUserID(c))
	}
	scope := strings.TrimSpace(c.SupplierID)
	if scope == "" {
		scope = auth.ResolveRetailerOrgID(c)
	}
	scope = sanitizeTenantSegment(scope)
	if !isSafePathSegment(scope) || userID == "" {
		return "", ""
	}
	return scope, userID
}

func sanitizeTenantSegment(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	return s
}

func isSafePathSegment(seg string) bool {
	if seg == "" || seg == "." || seg == ".." {
		return false
	}
	return !strings.ContainsAny(seg, `/\`) && !strings.Contains(seg, "..")
}
