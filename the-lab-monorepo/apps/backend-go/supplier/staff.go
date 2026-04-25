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
	"backend-go/order"
	"backend-go/pkg/pin"

	"cloud.google.com/go/spanner"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/iterator"
)

// ── Request / Response DTOs ───────────────────────────────────────────────

type CreatePayloaderRequest struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

type CreatePayloaderResponse struct {
	WorkerID   string `json:"worker_id"`
	Name       string `json:"name"`
	Phone      string `json:"phone"`
	SupplierId string `json:"supplier_id"`
	Pin        string `json:"pin"` // Plaintext — shown ONCE
}

type PayloaderListItem struct {
	WorkerID  string `json:"worker_id"`
	Name      string `json:"name"`
	Phone     string `json:"phone"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
}

// ── Handlers ──────────────────────────────────────────────────────────────

// HandleStaffPayloaders routes GET (list) and POST (create) for /v1/supplier/staff/payloader
func HandleStaffPayloaders(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listPayloaders(w, r, spannerClient)
		case http.MethodPost:
			createPayloader(w, r, spannerClient)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

// HandlePayloaderDetail handles detail operations for a single payloader.
// POST /v1/supplier/staff/payloader/{id}/rotate-pin
func HandlePayloaderDetail(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workerID := r.URL.Path[len("/v1/supplier/staff/payloader/"):]

		// Route: POST .../rotate-pin
		if strings.HasSuffix(workerID, "/rotate-pin") {
			workerID = strings.TrimSuffix(workerID, "/rotate-pin")
			rotatePayloaderPIN(w, r, spannerClient, workerID)
			return
		}

		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// rotatePayloaderPIN generates a new globally-unique PIN for a supplier payloader.
func rotatePayloaderPIN(w http.ResponseWriter, r *http.Request, spannerClient *spanner.Client, workerID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if workerID == "" || strings.Contains(workerID, "/") {
		http.Error(w, `{"error":"worker_id required"}`, http.StatusBadRequest)
		return
	}

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	supplierID := claims.ResolveSupplierID()

	// Verify payloader belongs to this supplier.
	row, err := spannerClient.Single().ReadRow(r.Context(), "WarehouseStaff",
		spanner.Key{workerID}, []string{"SupplierId"})
	if err != nil {
		http.Error(w, `{"error":"payloader not found"}`, http.StatusNotFound)
		return
	}
	var ownerID string
	if err := row.Columns(&ownerID); err != nil || ownerID != supplierID {
		http.Error(w, `{"error":"payloader not found"}`, http.StatusNotFound)
		return
	}

	var pinResult *pin.Result
	_, err = spannerClient.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var rotErr error
		pinResult, rotErr = pin.Rotate(ctx, txn, pin.EntityWarehouseStaff, workerID)
		if rotErr != nil {
			return rotErr
		}
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("WarehouseStaff", []string{"WorkerId", "PinHash"}, []interface{}{workerID, pinResult.BcryptHash}),
		})
	})
	if err != nil {
		log.Printf("[PAYLOADER] rotate PIN failed for %s: %v", workerID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"worker_id": workerID,
		"pin":       pinResult.Plaintext,
	})
}

// HandlePayloaderLogin authenticates a warehouse worker with phone + PIN
// POST /v1/auth/payloader/login
func HandlePayloaderLogin(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Phone string `json:"phone"`
			Pin   string `json:"pin"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}
		if req.Phone == "" || req.Pin == "" {
			http.Error(w, `{"error":"phone and pin required"}`, http.StatusBadRequest)
			return
		}

		stmt := spanner.Statement{
			SQL: `SELECT ws.WorkerId, ws.SupplierId, ws.Name, ws.PinHash, ws.IsActive,
			             COALESCE(ws.WarehouseId, ''),
			             COALESCE(w.Name, ''), COALESCE(w.Lat, 0), COALESCE(w.Lng, 0)
			      FROM WarehouseStaff ws
			      LEFT JOIN Warehouses w ON ws.WarehouseId = w.WarehouseId
			      WHERE ws.Phone = @phone`,
			Params: map[string]interface{}{
				"phone": req.Phone,
			},
		}

		iter := spannerClient.Single().Query(r.Context(), stmt)
		defer iter.Stop()

		row, err := iter.Next()
		if err == iterator.Done {
			http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
			return
		}
		if err != nil {
			log.Printf("[PAYLOADER AUTH] query error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		var workerID, supplierID, name, pinHash string
		var warehouseID, warehouseName string
		var warehouseLat, warehouseLng float64
		var isActive bool
		if err := row.Columns(&workerID, &supplierID, &name, &pinHash, &isActive,
			&warehouseID, &warehouseName, &warehouseLat, &warehouseLng); err != nil {
			log.Printf("[PAYLOADER AUTH] parse error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if !isActive {
			http.Error(w, `{"error":"account deactivated"}`, http.StatusForbidden)
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(pinHash), []byte(req.Pin)); err != nil {
			http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
			return
		}

		token, err := auth.MintIdentityToken(&auth.LabClaims{
			UserID:        workerID,
			SupplierID:    supplierID,
			Role:          "PAYLOADER",
			WarehouseID:   warehouseID,
			WarehouseRole: "PAYLOADER",
		})
		if err != nil {
			log.Printf("[PAYLOADER AUTH] token generation error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Mint Firebase custom token (graceful degradation)
		var firebaseToken string
		if auth.FirebaseAuthClient != nil {
			var fbUid string
			_ = spannerClient.Single().Query(r.Context(), spanner.Statement{
				SQL:    "SELECT COALESCE(FirebaseUid, '') FROM WarehouseStaff WHERE WorkerId = @id",
				Params: map[string]interface{}{"id": workerID},
			}).Do(func(row *spanner.Row) error { return row.Columns(&fbUid) })
			if fbUid != "" {
				firebaseToken, _ = auth.MintCustomToken(r.Context(), fbUid, map[string]interface{}{"role": "PAYLOADER", "worker_id": workerID, "supplier_id": supplierID})
			}
		}

		resp := map[string]interface{}{
			"token":          token,
			"worker_id":      workerID,
			"supplier_id":    supplierID,
			"role":           "PAYLOADER",
			"name":           name,
			"warehouse_id":   warehouseID,
			"warehouse_name": warehouseName,
			"warehouse_lat":  warehouseLat,
			"warehouse_lng":  warehouseLng,
		}
		if firebaseToken != "" {
			resp["firebase_token"] = firebaseToken
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// ── Private Implementations ───────────────────────────────────────────────

func listPayloaders(w http.ResponseWriter, r *http.Request, client *spanner.Client) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	supplierID := claims.ResolveSupplierID()

	stmt := spanner.Statement{
		SQL: `SELECT WorkerId, Name, Phone, IsActive, CreatedAt
		      FROM WarehouseStaff
		      WHERE SupplierId = @supplierId
		      ORDER BY CreatedAt DESC`,
		Params: map[string]interface{}{
			"supplierId": supplierID,
		},
	}
	stmt = auth.AppendWarehouseFilterStmt(r.Context(), stmt, "WarehouseStaff")

	iter := client.Single().Query(r.Context(), stmt)
	defer iter.Stop()

	var workers []PayloaderListItem
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("[STAFF] list query error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		var p PayloaderListItem
		var createdAt time.Time
		if err := row.Columns(&p.WorkerID, &p.Name, &p.Phone, &p.IsActive, &createdAt); err != nil {
			log.Printf("[STAFF] list parse error: %v", err)
			continue
		}
		p.CreatedAt = createdAt.Format(time.RFC3339)
		workers = append(workers, p)
	}

	if workers == nil {
		workers = []PayloaderListItem{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workers)
}

func createPayloader(w http.ResponseWriter, r *http.Request, client *spanner.Client) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	supplierID := claims.ResolveSupplierID()

	var req CreatePayloaderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Phone == "" {
		http.Error(w, `{"error":"name and phone required"}`, http.StatusBadRequest)
		return
	}

	workerID := fmt.Sprintf("WRK-%s", order.GenerateSecureToken())

	warehouseID := auth.EffectiveWarehouseID(r.Context())

	var pinResult *pin.Result
	_, err := client.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var pinErr error
		pinResult, pinErr = pin.GenerateUnique(ctx, txn, pin.EntityWarehouseStaff, workerID)
		if pinErr != nil {
			return fmt.Errorf("generate PIN: %w", pinErr)
		}
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Insert("WarehouseStaff",
				[]string{"WorkerId", "SupplierId", "Name", "Phone", "PinHash", "IsActive", "CreatedAt", "WarehouseId"},
				[]interface{}{workerID, supplierID, req.Name, req.Phone, pinResult.BcryptHash, true, spanner.CommitTimestamp, nullStr(warehouseID)}),
		})
	})
	if err != nil {
		log.Printf("[STAFF] insert failed: %v", err)
		http.Error(w, "Failed to provision worker", http.StatusInternalServerError)
		return
	}

	// Create Firebase Auth user for payloader (phone-based, graceful degradation)
	fbUid, fbErr := auth.CreateFirebaseUser(r.Context(), "", "", req.Name, req.Phone, "PAYLOADER", map[string]interface{}{
		"worker_id":   workerID,
		"supplier_id": supplierID,
	})
	if fbErr == nil && fbUid != "" {
		_, _ = client.Apply(r.Context(), []*spanner.Mutation{
			spanner.Update("WarehouseStaff", []string{"WorkerId", "FirebaseUid"}, []interface{}{workerID, fbUid}),
		})
	}

	log.Printf("[STAFF] Provisioned payloader %s (%s) for supplier %s", workerID, req.Name, supplierID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(CreatePayloaderResponse{
		WorkerID:   workerID,
		Name:       req.Name,
		Phone:      req.Phone,
		SupplierId: supplierID,
		Pin:        pinResult.Plaintext,
	})
}

// HandlePayloaderTrucks returns vehicles belonging to the payloader's supplier.
// GET /v1/payloader/trucks
func HandlePayloaderTrucks(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Resolve supplier + warehouse from WarehouseStaff
		stmt := spanner.Statement{
			SQL:    `SELECT SupplierId, COALESCE(WarehouseId, '') FROM WarehouseStaff WHERE WorkerId = @wid`,
			Params: map[string]interface{}{"wid": claims.UserID},
		}
		iter := spannerClient.Single().Query(r.Context(), stmt)
		defer iter.Stop()

		row, err := iter.Next()
		if err == iterator.Done {
			http.Error(w, `{"error":"payloader not found"}`, http.StatusNotFound)
			return
		}
		if err != nil {
			log.Printf("[PAYLOADER TRUCKS] supplier lookup error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		var supplierID, warehouseID string
		if err := row.Columns(&supplierID, &warehouseID); err != nil {
			log.Printf("[PAYLOADER TRUCKS] parse error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Fetch active vehicles — scoped to payloader's warehouse when assigned
		vSQL := `SELECT VehicleId, COALESCE(Label, ''), COALESCE(LicensePlate, ''), VehicleClass
		         FROM Vehicles
		         WHERE SupplierId = @sid AND IsActive = true`
		vParams := map[string]interface{}{"sid": supplierID}
		if warehouseID != "" {
			vSQL += " AND WarehouseId = @whid"
			vParams["whid"] = warehouseID
		}
		vSQL += " ORDER BY Label"
		vStmt := spanner.Statement{SQL: vSQL, Params: vParams}
		vIter := spannerClient.Single().Query(r.Context(), vStmt)
		defer vIter.Stop()

		type truckItem struct {
			ID           string `json:"id"`
			Label        string `json:"label"`
			LicensePlate string `json:"license_plate"`
			VehicleClass string `json:"vehicle_class"`
		}

		trucks := []truckItem{}
		for {
			vRow, err := vIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[PAYLOADER TRUCKS] vehicles query error: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			var t truckItem
			if err := vRow.Columns(&t.ID, &t.Label, &t.LicensePlate, &t.VehicleClass); err != nil {
				log.Printf("[PAYLOADER TRUCKS] parse vehicle error: %v", err)
				continue
			}
			trucks = append(trucks, t)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(trucks)
	}
}

// HandlePayloaderOrders returns orders assigned to the payloader's supplier vehicles,
// hydrated with line items for manifest building.
// GET /v1/payloader/orders?vehicle_id=xxx&state=PENDING
func HandlePayloaderOrders(spannerClient *spanner.Client) http.HandlerFunc {
	type lineItem struct {
		LineItemID string `json:"line_item_id"`
		SkuID      string `json:"sku_id"`
		SkuName    string `json:"sku_name"`
		Quantity   int64  `json:"quantity"`
		UnitPrice  int64  `json:"unit_price"`
		Status     string `json:"status"`
	}
	type orderResp struct {
		OrderID        string     `json:"order_id"`
		RetailerID     string     `json:"retailer_id"`
		Amount         int64      `json:"amount"`
		PaymentGateway string     `json:"payment_gateway"`
		State          string     `json:"state"`
		RouteID        string     `json:"route_id"`
		Items          []lineItem `json:"items"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Resolve supplier + warehouse from WarehouseStaff
		stmt := spanner.Statement{
			SQL:    `SELECT SupplierId, COALESCE(WarehouseId, '') FROM WarehouseStaff WHERE WorkerId = @wid`,
			Params: map[string]interface{}{"wid": claims.UserID},
		}
		iter := spannerClient.Single().Query(r.Context(), stmt)
		defer iter.Stop()

		row, err := iter.Next()
		if err == iterator.Done {
			http.Error(w, `{"error":"payloader not found"}`, http.StatusNotFound)
			return
		}
		if err != nil {
			log.Printf("[PAYLOADER ORDERS] supplier lookup error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		var supplierID, warehouseID string
		if err := row.Columns(&supplierID, &warehouseID); err != nil {
			log.Printf("[PAYLOADER ORDERS] parse error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		vehicleID := r.URL.Query().Get("vehicle_id")
		stateFilter := r.URL.Query().Get("state")

		// Fetch orders whose RouteId matches a vehicle belonging to this supplier
		sql := `SELECT o.OrderId, o.RetailerId, o.Amount, o.PaymentGateway, o.State, o.RouteId
		        FROM Orders o
		        WHERE o.RouteId IN (SELECT VehicleId FROM Vehicles WHERE SupplierId = @sid AND IsActive = true)`
		params := map[string]interface{}{"sid": supplierID}

		if warehouseID != "" {
			sql += " AND o.WarehouseId = @whid"
			params["whid"] = warehouseID
		}

		if vehicleID != "" {
			sql += " AND o.RouteId = @vid"
			params["vid"] = vehicleID
		}
		if stateFilter != "" {
			sql += " AND o.State = @state"
			params["state"] = stateFilter
		}
		sql += " ORDER BY o.CreatedAt DESC LIMIT 200"

		oStmt := spanner.Statement{SQL: sql, Params: params}
		oIter := spannerClient.Single().Query(r.Context(), oStmt)
		defer oIter.Stop()

		orders := []orderResp{}
		orderIDs := []string{}
		for {
			oRow, err := oIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[PAYLOADER ORDERS] query error: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			var o orderResp
			var amount spanner.NullInt64
			var gateway, routeID spanner.NullString
			if err := oRow.Columns(&o.OrderID, &o.RetailerID, &amount, &gateway, &o.State, &routeID); err != nil {
				log.Printf("[PAYLOADER ORDERS] parse error: %v", err)
				continue
			}
			o.Amount = amount.Int64
			o.PaymentGateway = gateway.StringVal
			o.RouteID = routeID.StringVal
			o.Items = []lineItem{}
			orders = append(orders, o)
			orderIDs = append(orderIDs, o.OrderID)
		}

		// Hydrate line items
		if len(orderIDs) > 0 {
			liStmt := spanner.Statement{
				SQL: `SELECT li.OrderId, li.LineItemId, li.SkuId, COALESCE(sp.Name, li.SkuId), li.Quantity, li.UnitPrice, li.Status
				      FROM OrderLineItems li
				      LEFT JOIN SupplierProducts sp ON li.SkuId = sp.SkuId
				      WHERE li.OrderId IN UNNEST(@oids)`,
				Params: map[string]interface{}{"oids": orderIDs},
			}
			liIter := spannerClient.Single().Query(r.Context(), liStmt)
			defer liIter.Stop()

			itemMap := map[string][]lineItem{}
			for {
				liRow, err := liIter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					break
				}
				var ordID string
				var li lineItem
				if err := liRow.Columns(&ordID, &li.LineItemID, &li.SkuID, &li.SkuName, &li.Quantity, &li.UnitPrice, &li.Status); err != nil {
					continue
				}
				itemMap[ordID] = append(itemMap[ordID], li)
			}
			for i := range orders {
				if items, ok := itemMap[orders[i].OrderID]; ok {
					orders[i].Items = items
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(orders)
	}
}
