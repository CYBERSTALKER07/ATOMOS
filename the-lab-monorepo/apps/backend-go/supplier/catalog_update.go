package supplier

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"

	"cloud.google.com/go/spanner"
)

// HandleUpdateProduct applies partial updates to an existing supplier product.
// PUT /v1/supplier/products/{sku_id}
func HandleUpdateProduct(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if !ok || claims == nil || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		supplierId := claims.ResolveSupplierID()

		skuId := strings.TrimPrefix(r.URL.Path, "/v1/supplier/products/")
		if skuId == "" {
			http.Error(w, `{"error":"sku_id required in path"}`, http.StatusBadRequest)
			return
		}

		var update struct {
			Name            *string  `json:"name"`
			Description     *string  `json:"description"`
			ImageUrl        *string  `json:"image_url"`
			CategoryId      *string  `json:"category_id"`
			SellByBlock     *bool    `json:"sell_by_block"`
			UnitsPerBlock   *int64   `json:"units_per_block"`
			BasePrice    *int64   `json:"base_price"`
			MinimumOrderQty *int64   `json:"minimum_order_qty"`
			StepSize        *int64   `json:"step_size"`
			IsActive        *bool    `json:"is_active"`
			LengthCM        *float64 `json:"length_cm"`
			WidthCM         *float64 `json:"width_cm"`
			HeightCM        *float64 `json:"height_cm"`
		}
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			http.Error(w, `{"error":"malformed update payload"}`, http.StatusBadRequest)
			return
		}

		if update.Name != nil && strings.TrimSpace(*update.Name) == "" {
			http.Error(w, `{"error":"product name cannot be empty"}`, http.StatusBadRequest)
			return
		}
		if update.ImageUrl != nil && *update.ImageUrl != "" && !strings.HasPrefix(*update.ImageUrl, "https://") {
			http.Error(w, `{"error":"image_url must use HTTPS"}`, http.StatusBadRequest)
			return
		}
		if update.BasePrice != nil && *update.BasePrice <= 0 {
			http.Error(w, `{"error":"base_price must be greater than zero"}`, http.StatusBadRequest)
			return
		}

		if update.CategoryId != nil {
			if err := ensureCanonicalCategoriesSeeded(r.Context(), client); err != nil {
				http.Error(w, "Category catalog unavailable", http.StatusInternalServerError)
				return
			}
			if _, ok := canonicalCategoryIndex[*update.CategoryId]; !ok {
				http.Error(w, `{"error":"unknown category_id"}`, http.StatusBadRequest)
				return
			}
			isConfigured, operatingCategories, err := loadSupplierCategoryAccess(r.Context(), client, supplierId)
			if err != nil || !isConfigured {
				http.Error(w, "Supplier profile unavailable", http.StatusInternalServerError)
				return
			}
			if !containsCategoryID(operatingCategories, *update.CategoryId) {
				http.Error(w, `{"error":"category_id is not enabled for this supplier"}`, http.StatusBadRequest)
				return
			}
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			stmt := spanner.Statement{
				SQL: `SELECT SupplierId, Name, COALESCE(Description, ''), COALESCE(ImageUrl, ''),
				             COALESCE(CategoryId, ''), SellByBlock, UnitsPerBlock, BasePrice,
				             COALESCE(VolumetricUnit, 1.0),
				             COALESCE(MinimumOrderQty, 1), COALESCE(StepSize, 1),
				             IsActive, LengthCM, WidthCM, HeightCM
				      FROM SupplierProducts WHERE SkuId = @skuId`,
				Params: map[string]interface{}{"skuId": skuId},
			}
			iter := txn.Query(ctx, stmt)
			defer iter.Stop()

			row, err := iter.Next()
			if err != nil {
				return fmt.Errorf("product not found")
			}

			var ownerSid, name, description, imageUrl, categoryId string
			var sellByBlock bool
			var unitsPerBlock, basePrice, minimumOrderQty, stepSize int64
			var volumetricUnit float64
			var isActive bool
			var lengthCM, widthCM, heightCM spanner.NullFloat64

			if err := row.Columns(&ownerSid, &name, &description, &imageUrl, &categoryId,
				&sellByBlock, &unitsPerBlock, &basePrice, &volumetricUnit,
				&minimumOrderQty, &stepSize, &isActive, &lengthCM, &widthCM, &heightCM); err != nil {
				return fmt.Errorf("parse error: %w", err)
			}

			if ownerSid != supplierId {
				return fmt.Errorf("access denied")
			}

			if update.Name != nil {
				name = *update.Name
			}
			if update.Description != nil {
				description = *update.Description
			}
			if update.ImageUrl != nil {
				imageUrl = *update.ImageUrl
			}
			if update.CategoryId != nil {
				categoryId = *update.CategoryId
			}
			if update.SellByBlock != nil {
				sellByBlock = *update.SellByBlock
			}
			if update.UnitsPerBlock != nil {
				unitsPerBlock = *update.UnitsPerBlock
			}
			if update.BasePrice != nil {
				basePrice = *update.BasePrice
			}
			if update.MinimumOrderQty != nil {
				minimumOrderQty = *update.MinimumOrderQty
			}
			if update.StepSize != nil {
				stepSize = *update.StepSize
			}
			if update.IsActive != nil {
				isActive = *update.IsActive
			}
			if update.LengthCM != nil {
				lengthCM = spanner.NullFloat64{Float64: *update.LengthCM, Valid: true}
			}
			if update.WidthCM != nil {
				widthCM = spanner.NullFloat64{Float64: *update.WidthCM, Valid: true}
			}
			if update.HeightCM != nil {
				heightCM = spanner.NullFloat64{Float64: *update.HeightCM, Valid: true}
			}

			// Recompute VU if all dimensions present
			if lengthCM.Valid && widthCM.Valid && heightCM.Valid {
				computed := (lengthCM.Float64 * widthCM.Float64 * heightCM.Float64) / 5000.0
				if computed > 0 {
					volumetricUnit = computed
				}
			}

			if stepSize <= 0 {
				stepSize = 1
			}
			if minimumOrderQty <= 0 {
				minimumOrderQty = stepSize
			}
			if minimumOrderQty < stepSize {
				minimumOrderQty = stepSize
			}
			if unitsPerBlock <= 0 {
				unitsPerBlock = 1
			}

			txn.BufferWrite([]*spanner.Mutation{
				spanner.Update("SupplierProducts",
					[]string{"SkuId", "SupplierId", "Name", "Description", "ImageUrl", "CategoryId",
						"SellByBlock", "UnitsPerBlock", "BasePrice", "VolumetricUnit",
						"MinimumOrderQty", "StepSize", "IsActive", "LengthCM", "WidthCM", "HeightCM"},
					[]interface{}{skuId, supplierId, name, description, imageUrl, categoryId,
						sellByBlock, unitsPerBlock, basePrice, volumetricUnit,
						minimumOrderQty, stepSize, isActive, lengthCM, widthCM, heightCM},
				),
			})
			return nil
		})

		if err != nil {
			errMsg := err.Error()
			if strings.Contains(errMsg, "not found") {
				http.Error(w, `{"error":"product not found"}`, http.StatusNotFound)
			} else if strings.Contains(errMsg, "access denied") {
				http.Error(w, `{"error":"access denied"}`, http.StatusForbidden)
			} else {
				log.Printf("[SUPPLIER CATALOG] update error for %s/%s: %v", supplierId, skuId, err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "PRODUCT_UPDATED",
			"sku_id": skuId,
		})
	}
}

// HandleDeactivateProduct soft-deletes a product by setting IsActive = false.
// DELETE /v1/supplier/products/{sku_id}
func HandleDeactivateProduct(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if !ok || claims == nil || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		supplierId := claims.ResolveSupplierID()

		skuId := strings.TrimPrefix(r.URL.Path, "/v1/supplier/products/")
		if skuId == "" {
			http.Error(w, `{"error":"sku_id required in path"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			row, err := txn.ReadRow(ctx, "SupplierProducts", spanner.Key{skuId}, []string{"SupplierId"})
			if err != nil {
				return fmt.Errorf("product not found")
			}
			var ownerSid string
			if err := row.Columns(&ownerSid); err != nil {
				return err
			}
			if ownerSid != supplierId {
				return fmt.Errorf("access denied")
			}

			txn.BufferWrite([]*spanner.Mutation{
				spanner.Update("SupplierProducts",
					[]string{"SkuId", "SupplierId", "IsActive"},
					[]interface{}{skuId, supplierId, false},
				),
			})
			return nil
		})

		if err != nil {
			errMsg := err.Error()
			if strings.Contains(errMsg, "not found") {
				http.Error(w, `{"error":"product not found"}`, http.StatusNotFound)
			} else if strings.Contains(errMsg, "access denied") {
				http.Error(w, `{"error":"access denied"}`, http.StatusForbidden)
			} else {
				log.Printf("[SUPPLIER CATALOG] deactivate error for %s/%s: %v", supplierId, skuId, err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "PRODUCT_DEACTIVATED",
			"sku_id": skuId,
		})
	}
}

// HandleGetProduct returns a single product by SKU ID, scoped to the supplier.
// GET /v1/supplier/products/{sku_id}
func HandleGetProduct(client *spanner.Client) http.HandlerFunc {
	type ProductDetail struct {
		SkuID           string   `json:"sku_id"`
		Name            string   `json:"name"`
		Description     string   `json:"description"`
		ImageURL        string   `json:"image_url"`
		SellByBlock     bool     `json:"sell_by_block"`
		UnitsPerBlock   int64    `json:"units_per_block"`
		BasePrice    int64    `json:"base_price"`
		IsActive        bool     `json:"is_active"`
		CategoryID      string   `json:"category_id"`
		CategoryName    string   `json:"category_name"`
		VolumetricUnit  float64  `json:"volumetric_unit"`
		MinimumOrderQty int64    `json:"minimum_order_qty"`
		StepSize        int64    `json:"step_size"`
		CreatedAt       string   `json:"created_at"`
		LengthCM        *float64 `json:"length_cm,omitempty"`
		WidthCM         *float64 `json:"width_cm,omitempty"`
		HeightCM        *float64 `json:"height_cm,omitempty"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if !ok || claims == nil || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		supplierID := claims.ResolveSupplierID()

		skuId := strings.TrimPrefix(r.URL.Path, "/v1/supplier/products/")
		if skuId == "" {
			http.Error(w, `{"error":"sku_id required in path"}`, http.StatusBadRequest)
			return
		}

		stmt := spanner.Statement{
			SQL: `SELECT sp.SkuId, sp.Name, COALESCE(sp.Description, ''), COALESCE(sp.ImageUrl, ''),
			             sp.SellByBlock, sp.UnitsPerBlock, sp.BasePrice, sp.IsActive, COALESCE(sp.CategoryId, ''),
			             COALESCE(c.Name, ''),
			             COALESCE(sp.VolumetricUnit, COALESCE(sp.PalletFootprint, 1.0)),
			             COALESCE(sp.MinimumOrderQty, 1), COALESCE(sp.StepSize, 1), sp.CreatedAt,
			             sp.LengthCM, sp.WidthCM, sp.HeightCM
			      FROM SupplierProducts sp
			      LEFT JOIN Categories c ON c.CategoryId = sp.CategoryId
			      WHERE sp.SkuId = @skuId AND sp.SupplierId = @supplierId`,
			Params: map[string]interface{}{
				"skuId":      skuId,
				"supplierId": supplierID,
			},
		}

		iter := client.Single().Query(r.Context(), stmt)
		defer iter.Stop()

		row, err := iter.Next()
		if err != nil {
			http.Error(w, `{"error":"product not found"}`, http.StatusNotFound)
			return
		}

		var p ProductDetail
		var createdAt time.Time
		if err := row.Columns(&p.SkuID, &p.Name, &p.Description, &p.ImageURL,
			&p.SellByBlock, &p.UnitsPerBlock, &p.BasePrice, &p.IsActive, &p.CategoryID, &p.CategoryName,
			&p.VolumetricUnit, &p.MinimumOrderQty, &p.StepSize, &createdAt,
			&p.LengthCM, &p.WidthCM, &p.HeightCM); err != nil {
			log.Printf("[SUPPLIER CATALOG] single-product parse error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		p.CreatedAt = createdAt.Format(time.RFC3339)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(p)
	}
}
