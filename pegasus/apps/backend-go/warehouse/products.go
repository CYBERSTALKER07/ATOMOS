package warehouse

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"backend-go/auth"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ─── Products (catalog view — warehouse can view supplier's products) ─────────

type ProductItem struct {
	SkuID      string  `json:"sku_id"`
	Name       string  `json:"name"`
	CategoryID string  `json:"category_id,omitempty"`
	UnitPrice  int64   `json:"unit_price"`
	VolumeVU   float64 `json:"volume_vu"`
	IsActive   bool    `json:"is_active"`
	ImageURL   string  `json:"image_url,omitempty"`
}

// HandleOpsProducts — GET for /v1/warehouse/ops/products (read-only catalog view)
func HandleOpsProducts(spannerClient *spanner.Client) http.HandlerFunc {
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

		sql := `SELECT SkuId, Name, COALESCE(CategoryId, ''), COALESCE(Price, 0),
		               COALESCE(VolumeVU, 0), COALESCE(IsActive, true), COALESCE(ImageUrl, '')
		        FROM SupplierProducts
		        WHERE SupplierId = @sid`
		params := map[string]interface{}{"sid": ops.SupplierID}

		if q := r.URL.Query().Get("q"); q != "" {
			sql += " AND LOWER(Name) LIKE @search"
			params["search"] = "%" + strings.ToLower(q) + "%"
		}
		if catID := r.URL.Query().Get("category_id"); catID != "" {
			sql += " AND CategoryId = @catId"
			params["catId"] = catID
		}
		if r.URL.Query().Get("active") != "false" {
			sql += " AND IsActive = true"
		}

		sql += " ORDER BY Name LIMIT 500"

		stmt := spanner.Statement{SQL: sql, Params: params}
		iter := spannerClient.Single().Query(r.Context(), stmt)
		defer iter.Stop()

		var products []ProductItem
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[WH PRODUCTS] list error: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			var p ProductItem
			if err := row.Columns(&p.SkuID, &p.Name, &p.CategoryID,
				&p.UnitPrice, &p.VolumeVU, &p.IsActive, &p.ImageURL); err != nil {
				log.Printf("[WH PRODUCTS] parse: %v", err)
				continue
			}
			products = append(products, p)
		}
		if products == nil {
			products = []ProductItem{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"products": products, "total": len(products)})
	}
}
