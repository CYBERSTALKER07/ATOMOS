package supplier

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"backend-go/auth"
	internalKafka "backend-go/kafka"
	"backend-go/outbox"
	"backend-go/proximity"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// HandleRetailerRegister creates a new retailer account.
// POST /v1/auth/retailer/register → { phone_number, password, owner_name, store_name, address_text?, latitude?, longitude?, tax_id? }
func HandleRetailerRegister(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			PhoneNumber            string  `json:"phone_number"`
			Password               string  `json:"password"`
			OwnerName              string  `json:"owner_name"`
			StoreName              string  `json:"store_name"`
			AddressText            string  `json:"address_text"`
			Latitude               float64 `json:"latitude"`
			Longitude              float64 `json:"longitude"`
			TaxId                  string  `json:"tax_id"`
			ReceivingWindowOpen    string  `json:"receiving_window_open"`     // e.g. "08:00"
			ReceivingWindowClose   string  `json:"receiving_window_close"`    // e.g. "17:00"
			AccessType             string  `json:"access_type"`               // STREET_PARKING | ALLEYWAY | LOADING_DOCK
			StorageCeilingHeightCM float64 `json:"storage_ceiling_height_cm"` // 0 means not provided
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
			return
		}
		if req.PhoneNumber == "" || req.Password == "" || req.OwnerName == "" {
			http.Error(w, `{"error":"phone_number, password, and owner_name are required"}`, http.StatusBadRequest)
			return
		}
		if len(req.Password) < 4 {
			http.Error(w, `{"error":"password must be at least 4 characters"}`, http.StatusBadRequest)
			return
		}

		// Check phone uniqueness
		stmt := spanner.Statement{
			SQL:    `SELECT RetailerId FROM Retailers WHERE Phone = @phone LIMIT 1`,
			Params: map[string]interface{}{"phone": req.PhoneNumber},
		}
		iter := spannerClient.Single().Query(r.Context(), stmt)
		row, err := iter.Next()
		iter.Stop()
		if row != nil && err == nil {
			http.Error(w, `{"error":"phone number already registered"}`, http.StatusConflict)
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("[RETAILER REGISTER] bcrypt error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		retailerId := uuid.New().String()
		shopName := req.StoreName
		if shopName == "" {
			shopName = req.OwnerName
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		insertMap := map[string]interface{}{
			"RetailerId":              retailerId,
			"Name":                    req.OwnerName,
			"Phone":                   req.PhoneNumber,
			"ShopName":                shopName,
			"ShopLocation":            req.AddressText,
			"TaxIdentificationNumber": req.TaxId,
			"PasswordHash":            string(hash),
			"Status":                  "VERIFIED",
			"CreatedAt":               spanner.CommitTimestamp,
		}
		if req.Latitude != 0 || req.Longitude != 0 {
			insertMap["Latitude"] = req.Latitude
			insertMap["Longitude"] = req.Longitude
			insertMap["H3Index"] = proximity.LookupCell(req.Latitude, req.Longitude)
			// Address is a label only — lat/lng is the operational source of truth.
			// AddressVerified = false until reverse-geocode or admin confirmation.
			insertMap["AddressVerified"] = false
		}
		if req.ReceivingWindowOpen != "" {
			canon, err := proximity.ValidateReceivingWindow(req.ReceivingWindowOpen)
			if err != nil {
				http.Error(w, `{"error":"invalid receiving_window_open: expected HH:MM 24-hour format"}`, http.StatusBadRequest)
				return
			}
			insertMap["ReceivingWindowOpen"] = canon
		}
		if req.ReceivingWindowClose != "" {
			canon, err := proximity.ValidateReceivingWindow(req.ReceivingWindowClose)
			if err != nil {
				http.Error(w, `{"error":"invalid receiving_window_close: expected HH:MM 24-hour format"}`, http.StatusBadRequest)
				return
			}
			insertMap["ReceivingWindowClose"] = canon
		}
		if req.AccessType != "" {
			insertMap["AccessType"] = req.AccessType
		}
		if req.StorageCeilingHeightCM > 0 {
			insertMap["StorageCeilingHeightCM"] = req.StorageCeilingHeightCM
		}
		m := spanner.InsertMap("Retailers", insertMap)
		if _, err := spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			if err := txn.BufferWrite([]*spanner.Mutation{m}); err != nil {
				return err
			}
			h3Cell := ""
			if v, ok := insertMap["H3Index"].(string); ok {
				h3Cell = v
			}
			lat, _ := insertMap["Latitude"].(float64)
			lng, _ := insertMap["Longitude"].(float64)
			event := internalKafka.RetailerRegisteredEvent{
				RetailerId:  retailerId,
				OwnerName:   req.OwnerName,
				ShopName:    shopName,
				PhoneNumber: req.PhoneNumber,
				Lat:         lat,
				Lng:         lng,
				H3Cell:      h3Cell,
				Timestamp:   time.Now().UTC(),
			}
			return outbox.EmitJSON(txn, "Retailer", retailerId,
				internalKafka.EventRetailerRegistered, internalKafka.TopicMain, event,
				telemetry.TraceIDFromContext(ctx))
		}); err != nil {
			log.Printf("[RETAILER REGISTER] spanner insert error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		token, err := auth.GenerateTestToken(retailerId, "RETAILER")
		if err != nil {
			log.Printf("[RETAILER REGISTER] token error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Create Firebase Auth user for retailer (phone-based, graceful degradation)
		var firebaseToken string
		fbUid, fbErr := auth.CreateFirebaseUser(ctx, "", "", req.OwnerName, req.PhoneNumber, "RETAILER", map[string]interface{}{
			"retailer_id": retailerId,
		})
		if fbErr == nil && fbUid != "" {
			_, _ = spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
				return txn.BufferWrite([]*spanner.Mutation{
					spanner.Update("Retailers", []string{"RetailerId", "FirebaseUid"}, []interface{}{retailerId, fbUid}),
				})
			})
			firebaseToken, _ = auth.MintCustomToken(ctx, fbUid, map[string]interface{}{"role": "RETAILER", "retailer_id": retailerId})
		}

		resp := map[string]interface{}{
			"token": token,
			"user": map[string]interface{}{
				"id":         retailerId,
				"name":       req.OwnerName,
				"company":    shopName,
				"email":      "",
				"avatar_url": nil,
			},
		}
		if firebaseToken != "" {
			resp["firebase_token"] = firebaseToken
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}
}
