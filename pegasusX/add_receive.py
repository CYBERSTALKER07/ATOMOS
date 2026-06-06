import re

with open("apps/backend-go/warehouse/ops_portal.go", "r") as f:
    content = f.read()

handler = """
func (s *Service) HandleReceiveTransfer(w http.ResponseWriter, r *http.Request) {
	s.ensurePortalSeed()
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}

	var body struct {
		TransferID string `json:"transfer_id"`
		Items      []struct {
			ProductID string `json:"product_id"`
			Quantity  int64  `json:"quantity"`
		} `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	defer r.Body.Close()

	warehouseID := "wh-1" // stub for single warehouse ops

	// For each item, update the inventory and emit INVENTORY_RECEIVED
	for _, item := range body.Items {
		err := s.repo.UpdateInventoryQuantity(r.Context(), warehouseID, item.ProductID, item.Quantity, func(txn outbox.TxnBuffer) error {
			payload := map[string]any{
				"type":        events.EventWarehouseInventoryReceived,
				"transfer_id": body.TransferID,
				"product_id":  item.ProductID,
				"quantity":    item.Quantity,
				"timestamp":   time.Now().UTC().Format(time.RFC3339),
			}
			return outbox.EmitJSON(r.Context(), txn, events.AggregateWarehouse, warehouseID, events.TopicMain, payload)
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed_to_receive_inventory"})
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "received"})
}
"""

content += "\n" + handler

with open("apps/backend-go/warehouse/ops_portal.go", "w") as f:
    f.write(content)

