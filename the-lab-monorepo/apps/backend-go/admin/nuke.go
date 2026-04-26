package admin

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/spanner"
)

// HandleNukeAllData deletes ALL data from every table in the Spanner database.
// DELETE /v1/admin/nuke — ADMIN-only, irreversible. For dev/emulator use only.
func HandleNukeAllData(spannerClient *spanner.Client) http.HandlerFunc {
	tables := []string{
		"OrderLineItems",
		"LedgerEntries",
		"LedgerAnomalies",
		"SupplierReturns",
		"InventoryAuditLog",
		"SupplierInventory",
		"RetailerProductSettings",
		"RetailerSupplierSettings",
		"RetailerGlobalSettings",
		"RetailerSuppliers",
		"SupplierProducts",
		"Products",
		"PricingTiers",
		"Orders",
		"MasterInvoices",
		"AIPredictions",
		"Drivers",
		"Retailers",
		"Suppliers",
		"Admins",
		"Categories",
		"PlatformCategories",
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		var mutations []*spanner.Mutation
		for _, t := range tables {
			mutations = append(mutations, spanner.Delete(t, spanner.AllKeys()))
		}

		if _, err := spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			return txn.BufferWrite(mutations)
		}); err != nil {
			log.Printf("[NUKE] Spanner delete failed: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		log.Println("[NUKE] All data purged from Spanner emulator.")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "NUKED",
			"tables":  tables,
			"message": "All data has been purged from every table.",
		})
	}
}
