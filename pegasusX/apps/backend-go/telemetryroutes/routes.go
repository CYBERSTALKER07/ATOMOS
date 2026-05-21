package telemetryroutes

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
)

type Deps struct {
	TelemetryHub        *ws.Hub
	FirebaseAuthEnabled bool
	FirebaseVerifier    auth.FirebaseVerifier
}

type LocationUpdate struct {
	DriverID  string  `json:"driver_id"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	Timestamp string  `json:"timestamp"`
}

func RegisterRoutes(r chi.Router, d Deps) {
	if d.TelemetryHub == nil {
		return
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var loc LocationUpdate
		if err := json.NewDecoder(req.Body).Decode(&loc); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		
		ident, ok := auth.FromContext(req.Context())
		if ok && ident.Role == auth.RoleDriver {
			loc.DriverID = ident.Subject // Enforce authenticity
		}

		payload, _ := json.Marshal(loc)
		// Broadcast to all telemetry subscribers (Supplier/Retailers tracking)
		d.TelemetryHub.Broadcast(context.Background(), "telemetry:live", payload)

		w.WriteHeader(http.StatusAccepted)
	})

	if d.FirebaseAuthEnabled && d.FirebaseVerifier != nil {
		r.With(auth.FirebaseAuth(d.FirebaseVerifier), auth.RequireRole(auth.RoleDriver)).Post("/v1/telemetry/location", handler)
	} else {
		r.Post("/v1/telemetry/location", handler)
	}
}
