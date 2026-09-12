package storageroutes

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/web"
	"github.com/pegasusx/pegasusx/apps/backend-go/storage"
)

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
	r.Route("/dossiers", func(r chi.Router) {
		r.Use(auth.RequireAnyAuthenticated()) // Requires a valid user token

		r.Post("/", func(w http.ResponseWriter, req *http.Request) {
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
			id := chi.URLParam(req, "dossier_id")
			dossier, items, err := vault.GetDossier(req.Context(), id)
			if err != nil {
				if err == storage.ErrDossierNotFound {
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
			id := chi.URLParam(req, "dossier_id")
			var body AddItemRequest
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				web.JSONError(w, "invalid request", http.StatusBadRequest)
				return
			}
			if body.FileHash == "" || body.ItemType == "" {
				web.JSONError(w, "file_hash and item_type are required", http.StatusBadRequest)
				return
			}

			// Generate upload ticket first
			uploadURL, publicURL, err := storage.GenerateUploadTicketFor("evidence/"+id, body.Extension)
			if err != nil {
				web.JSONError(w, "storage unavailable", http.StatusServiceUnavailable)
				return
			}

			claims, ok := auth.FromContext(req.Context())
			if !ok {
				web.JSONError(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			
			item := &storage.EvidenceItem{
				ItemType:       body.ItemType,
				StorageURI:     publicURL,
				FileHash:       body.FileHash,
				MimeType:       body.MimeType,
				SizeBytes:      body.SizeBytes,
				UploaderUserID: claims.Subject,
			}
			if body.Latitude != 0 || body.Longitude != 0 {
				item.Latitude.Valid = true
				item.Latitude.Float64 = body.Latitude
				item.Longitude.Valid = true
				item.Longitude.Float64 = body.Longitude
			}

			added, err := vault.AddItem(req.Context(), id, item)
			if err != nil {
				if err == storage.ErrDossierSealed {
					web.JSONError(w, "dossier is sealed", http.StatusConflict)
					return
				}
				if err == storage.ErrDossierNotFound {
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
			id := chi.URLParam(req, "dossier_id")
			if err := vault.SealDossier(req.Context(), id); err != nil {
				if err == storage.ErrDossierNotFound {
					web.JSONError(w, "dossier not found", http.StatusNotFound)
					return
				}
				if err == storage.ErrDossierSealed {
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
