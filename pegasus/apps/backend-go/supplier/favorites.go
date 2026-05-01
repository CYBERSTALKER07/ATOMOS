package supplier

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ── Retailer Supplier Favorites ───────────────────────────────────────────

// HandleRetailerSuppliers manages the retailer's saved/favorite suppliers.
// GET  /v1/retailer/suppliers           → List my suppliers
// POST /v1/retailer/suppliers/{id}/add  → Add to favorites
// POST /v1/retailer/suppliers/{id}/remove → Remove from favorites
func HandleRetailerSuppliers(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		retailerID := claims.UserID

		path := r.URL.Path

		switch {
		case r.Method == http.MethodGet && path == "/v1/retailer/suppliers":
			listRetailerSuppliers(w, r, client, retailerID)

		case r.Method == http.MethodPost && strings.HasSuffix(path, "/add"):
			supplierID := extractSupplierIDFromPath(path)
			if supplierID == "" {
				http.Error(w, "Missing supplier ID", http.StatusBadRequest)
				return
			}
			addRetailerSupplier(w, r, client, retailerID, supplierID)

		case r.Method == http.MethodPost && strings.HasSuffix(path, "/remove"):
			supplierID := extractSupplierIDFromPath(path)
			if supplierID == "" {
				http.Error(w, "Missing supplier ID", http.StatusBadRequest)
				return
			}
			removeRetailerSupplier(w, r, client, retailerID, supplierID)

		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

func listRetailerSuppliers(w http.ResponseWriter, r *http.Request, client *spanner.Client, retailerID string) {
	ctx := r.Context()

	stmt := spanner.Statement{
		SQL: `SELECT s.SupplierId, s.Name, s.LogoUrl, s.Category, COALESCE(s.OperatingCategories, []),
		             (SELECT COUNT(*) FROM Orders o WHERE o.RetailerId = @retailerId
		              AND EXISTS (SELECT 1 FROM OrderLineItems oli
		                          JOIN SupplierProducts sp ON oli.SkuId = sp.SkuId
		                          WHERE oli.OrderId = o.OrderId AND sp.SupplierId = s.SupplierId)) AS OrderCount,
		             IFNULL(s.ManualOffShift, false),
		             COALESCE(TO_JSON_STRING(s.OperatingSchedule), '{}')
		      FROM RetailerSuppliers rs
		      JOIN Suppliers s ON rs.SupplierId = s.SupplierId
		      WHERE rs.RetailerId = @retailerId
		      ORDER BY rs.AddedAt DESC`,
		Params: map[string]interface{}{
			"retailerId": retailerID,
		},
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	type MySupplier struct {
		ID                     string   `json:"id"`
		Name                   string   `json:"name"`
		LogoURL                string   `json:"logo_url"`
		Category               string   `json:"category"`
		PrimaryCategoryID      string   `json:"primary_category_id,omitempty"`
		OperatingCategoryIDs   []string `json:"operating_category_ids,omitempty"`
		OperatingCategoryNames []string `json:"operating_category_names,omitempty"`
		OrderCount             int64    `json:"order_count"`
		IsActive               bool     `json:"is_active"`
	}

	now := time.Now()
	var suppliers []MySupplier
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("[favorites] Failed to query retailer suppliers: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		var s MySupplier
		var logoUrl, category spanner.NullString
		var operatingCategoryIDs []string
		var manualOff bool
		var schedJSON string
		if err := row.Columns(&s.ID, &s.Name, &logoUrl, &category, &operatingCategoryIDs, &s.OrderCount, &manualOff, &schedJSON); err != nil {
			log.Printf("[favorites] Failed to parse supplier row: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if logoUrl.Valid {
			s.LogoURL = logoUrl.StringVal
		}
		if category.Valid {
			s.Category = category.StringVal
		}
		s.OperatingCategoryIDs = operatingCategoryIDs
		s.OperatingCategoryNames = categoryDisplayNames(operatingCategoryIDs)
		if len(operatingCategoryIDs) > 0 {
			s.PrimaryCategoryID = operatingCategoryIDs[0]
		}
		s.IsActive = resolveIsActive(schedJSON, manualOff, now)
		suppliers = append(suppliers, s)
	}

	if suppliers == nil {
		suppliers = []MySupplier{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suppliers)
}

func addRetailerSupplier(w http.ResponseWriter, r *http.Request, client *spanner.Client, retailerID, supplierID string) {
	ctx := r.Context()

	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertOrUpdate("RetailerSuppliers", []string{"RetailerId", "SupplierId", "AddedAt"},
				[]interface{}{retailerID, supplierID, spanner.CommitTimestamp}),
		})
	})
	if err != nil {
		log.Printf("[favorites] Failed to add supplier %s for retailer %s: %v", supplierID, retailerID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "added",
		"supplierId": supplierID,
		"addedAt":    time.Now().UTC().Format(time.RFC3339),
	})
}

func removeRetailerSupplier(w http.ResponseWriter, r *http.Request, client *spanner.Client, retailerID, supplierID string) {
	ctx := r.Context()

	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Delete("RetailerSuppliers", spanner.Key{retailerID, supplierID}),
		})
	})
	if err != nil {
		log.Printf("[favorites] Failed to remove supplier %s for retailer %s: %v", supplierID, retailerID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "removed",
		"supplierId": supplierID,
	})
}

// extractSupplierIDFromPath extracts supplier ID from paths like:
// /v1/retailer/suppliers/{id}/add → {id}
// /v1/retailer/suppliers/{id}/remove → {id}
func extractSupplierIDFromPath(path string) string {
	parts := splitPath(path)
	// ["v1", "retailer", "suppliers", "{id}", "add|remove"]
	if len(parts) >= 5 {
		return parts[3]
	}
	return ""
}
