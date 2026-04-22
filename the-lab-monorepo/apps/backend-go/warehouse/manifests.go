package warehouse

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"backend-go/auth"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ─── Manifests ────────────────────────────────────────────────────────────────
// Warehouse-scoped dispatch manifests view.

type ManifestItem struct {
	ManifestID   string `json:"manifest_id"`
	DriverID     string `json:"driver_id"`
	DriverName   string `json:"driver_name"`
	VehicleID    string `json:"vehicle_id,omitempty"`
	VehicleClass string `json:"vehicle_class,omitempty"`
	State        string `json:"state"`
	StopCount    int64  `json:"stop_count"`
	CreatedAt    string `json:"created_at"`
	SealedAt     string `json:"sealed_at,omitempty"`
}

// HandleOpsManifests — GET for /v1/warehouse/ops/manifests
func HandleOpsManifests(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		ops := auth.GetWarehouseOps(r.Context())
		if ops == nil {
			http.Error(w, "Warehouse scope required", http.StatusForbidden)
			return
		}

		dateStr := r.URL.Query().Get("date")
		if dateStr == "" {
			dateStr = time.Now().Format("2006-01-02")
		}
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			http.Error(w, `{"error":"invalid date, use YYYY-MM-DD"}`, http.StatusBadRequest)
			return
		}
		dayStart := date.Truncate(24 * time.Hour)
		dayEnd := dayStart.Add(24 * time.Hour)

		stmt := spanner.Statement{
			SQL: `SELECT m.ManifestId, m.DriverId, COALESCE(d.Name, ''),
			             COALESCE(m.VehicleId, ''), COALESCE(v.VehicleClass, ''),
			             m.State,
			             (SELECT COUNT(*) FROM ManifestStops ms WHERE ms.ManifestId = m.ManifestId),
			             m.CreatedAt, m.SealedAt
			      FROM Manifests m
			      LEFT JOIN Drivers d ON m.DriverId = d.DriverId
			      LEFT JOIN Vehicles v ON m.VehicleId = v.VehicleId
			      WHERE m.SupplierId = @sid AND m.WarehouseId = @whId
			        AND m.CreatedAt >= @dayStart AND m.CreatedAt < @dayEnd
			      ORDER BY m.CreatedAt DESC`,
			Params: map[string]interface{}{
				"sid": ops.SupplierID, "whId": ops.WarehouseID,
				"dayStart": dayStart, "dayEnd": dayEnd,
			},
		}

		iter := spannerClient.Single().Query(r.Context(), stmt)
		defer iter.Stop()

		var manifests []ManifestItem
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[WH MANIFESTS] list error: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			var m ManifestItem
			var createdAt time.Time
			var sealedAt spanner.NullTime
			if err := row.Columns(&m.ManifestID, &m.DriverID, &m.DriverName,
				&m.VehicleID, &m.VehicleClass, &m.State, &m.StopCount,
				&createdAt, &sealedAt); err != nil {
				log.Printf("[WH MANIFESTS] parse: %v", err)
				continue
			}
			m.CreatedAt = createdAt.Format(time.RFC3339)
			if sealedAt.Valid {
				m.SealedAt = sealedAt.Time.Format(time.RFC3339)
			}
			manifests = append(manifests, m)
		}
		if manifests == nil {
			manifests = []ManifestItem{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"manifests": manifests, "total": len(manifests)})
	}
}
