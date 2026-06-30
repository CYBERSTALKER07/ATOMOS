package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"backend-go/hotspot"

	"cloud.google.com/go/spanner"
)

// FamilyMember represents a cosmetic sub-profile for family-run shops.
type FamilyMember struct {
	MemberID         string `json:"member_id" spanner:"MemberId"`
	RetailerID       string `json:"retailer_id" spanner:"RetailerId"`
	Nickname         string `json:"nickname" spanner:"Nickname"`
	PhotoURL         string `json:"photo_url,omitempty" spanner:"PhotoUrl"`
	SpendingLimitUzs int64  `json:"spending_limit_uzs,omitempty" spanner:"SpendingLimitUzs"`
	CreatedAt        string `json:"created_at" spanner:"CreatedAt"`
}

// HandleListFamilyMembers returns all family members for the logged-in retailer.
// GET /v1/retailer/family-members (RETAILER role)
func HandleListFamilyMembers(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, ok := r.Context().Value(ClaimsContextKey).(*PegasusClaims)
		if !ok || claims == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		stmt := spanner.Statement{
			SQL:    `SELECT MemberId, RetailerId, Nickname, PhotoUrl, SpendingLimitUzs, CreatedAt FROM RetailerFamilyMembers WHERE RetailerId = @rid ORDER BY CreatedAt`,
			Params: map[string]interface{}{"rid": claims.UserID},
		}
		iter := client.Single().Query(ctx, stmt)
		defer iter.Stop()

		var members []FamilyMember
		for {
			row, err := iter.Next()
			if err != nil {
				break
			}
			var m FamilyMember
			var photo spanner.NullString
			var spendingLimit spanner.NullInt64
			var createdAt time.Time
			if err := row.Columns(&m.MemberID, &m.RetailerID, &m.Nickname, &photo, &spendingLimit, &createdAt); err != nil {
				continue
			}
			if photo.Valid {
				m.PhotoURL = photo.StringVal
			}
			if spendingLimit.Valid {
				m.SpendingLimitUzs = spendingLimit.Int64
			}
			m.CreatedAt = createdAt.Format(time.RFC3339)
			members = append(members, m)
		}

		if members == nil {
			members = []FamilyMember{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"members": members,
		})
	}
}

// HandleCreateFamilyMember adds a cosmetic sub-profile for a family-run shop.
// POST /v1/retailer/family-members (RETAILER role)
func HandleCreateFamilyMember(client *spanner.Client, invalidate func(context.Context, ...string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, ok := r.Context().Value(ClaimsContextKey).(*PegasusClaims)
		if !ok || claims == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req struct {
			Nickname         string `json:"nickname"`
			PhotoURL         string `json:"photo_url"`
			SpendingLimitUzs int64  `json:"spending_limit_uzs"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Nickname == "" {
			http.Error(w, `{"error":"nickname required"}`, http.StatusBadRequest)
			return
		}

		if len(req.Nickname) > 50 {
			http.Error(w, `{"error":"nickname too long (max 50)"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		// Max 10 members per retailer
		countStmt := spanner.Statement{
			SQL:    `SELECT COUNT(*) FROM RetailerFamilyMembers WHERE RetailerId = @rid`,
			Params: map[string]interface{}{"rid": claims.UserID},
		}
		iter := client.Single().Query(ctx, countStmt)
		row, err := iter.Next()
		iter.Stop()
		if err == nil {
			var count int64
			if row.Columns(&count) == nil && count >= 10 {
				http.Error(w, `{"error":"maximum 10 family members allowed"}`, http.StatusConflict)
				return
			}
		}

		memberID := hotspot.NewOpaqueID()
		cols := []string{"MemberId", "RetailerId", "Nickname", "CreatedAt"}
		vals := []interface{}{memberID, claims.UserID, req.Nickname, spanner.CommitTimestamp}
		if req.PhotoURL != "" {
			cols = append(cols, "PhotoUrl")
			vals = append(vals, req.PhotoURL)
		}
		if req.SpendingLimitUzs > 0 {
			cols = append(cols, "SpendingLimitUzs")
			vals = append(vals, req.SpendingLimitUzs)
		}

		_, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			return txn.BufferWrite([]*spanner.Mutation{
				spanner.Insert("RetailerFamilyMembers", cols, vals),
			})
		})
		if err != nil {
			log.Printf("[FAMILY_MEMBER] Create failed: %v", err)
			http.Error(w, fmt.Sprintf(`{"error":"create failed: %s"}`, err.Error()), http.StatusInternalServerError)
			return
		}

		if invalidate != nil {
			invalidate(r.Context(), "profile:retailer:"+claims.UserID)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"member_id": memberID,
			"nickname":  req.Nickname,
		})
	}
}

// HandleDeleteFamilyMember removes a family member sub-profile.
// DELETE /v1/retailer/family-members/{id} (RETAILER role)
func HandleDeleteFamilyMember(client *spanner.Client, invalidate func(context.Context, ...string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, ok := r.Context().Value(ClaimsContextKey).(*PegasusClaims)
		if !ok || claims == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		// Extract member ID from URL path: /v1/retailer/family-members/{id}
		parts := strings.Split(strings.TrimRight(r.URL.Path, "/"), "/")
		memberID := ""
		if len(parts) > 0 {
			memberID = parts[len(parts)-1]
		}
		if memberID == "" || memberID == "family-members" {
			http.Error(w, `{"error":"member_id required"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		// Verify member belongs to this retailer
		row, err := client.Single().ReadRow(ctx, "RetailerFamilyMembers",
			spanner.Key{claims.UserID, memberID}, // PK: RetailerId, MemberId
			[]string{"MemberId"})
		if err != nil || row == nil {
			http.Error(w, `{"error":"family member not found"}`, http.StatusNotFound)
			return
		}

		_, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			return txn.BufferWrite([]*spanner.Mutation{
				spanner.Delete("RetailerFamilyMembers", spanner.Key{claims.UserID, memberID}),
			})
		})
		if err != nil {
			log.Printf("[FAMILY_MEMBER] Delete failed: %v", err)
			http.Error(w, `{"error":"delete failed"}`, http.StatusInternalServerError)
			return
		}

		if invalidate != nil {
			invalidate(ctx, "profile:retailer:"+claims.UserID)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "DELETED",
			"member_id": memberID,
		})
	}
}

// CheckFamilySpendingLimit rejects checkout when a family sub-profile exceeds its cap.
// SpendingLimitUzs == 0 means unlimited.
func CheckFamilySpendingLimit(ctx context.Context, client *spanner.Client, retailerID, memberID string, orderTotal int64) error {
	if client == nil || retailerID == "" || memberID == "" || orderTotal <= 0 {
		return nil
	}
	row, err := client.Single().ReadRow(ctx, "RetailerFamilyMembers",
		spanner.Key{retailerID, memberID},
		[]string{"SpendingLimitUzs"},
	)
	if err != nil {
		return fmt.Errorf("family member not found")
	}
	var limit spanner.NullInt64
	if err := row.Columns(&limit); err != nil {
		return fmt.Errorf("family member limit lookup failed")
	}
	if !limit.Valid || limit.Int64 <= 0 {
		return nil
	}
	if orderTotal > limit.Int64 {
		return fmt.Errorf("order total exceeds family member spending limit")
	}
	return nil
}

// HandleUpdateFamilyMember updates cosmetic fields for a family sub-profile.
// PATCH /v1/retailer/family-members/{id}
func HandleUpdateFamilyMember(client *spanner.Client, invalidate func(context.Context, ...string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, ok := r.Context().Value(ClaimsContextKey).(*PegasusClaims)
		if !ok || claims == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.Split(strings.TrimRight(r.URL.Path, "/"), "/")
		memberID := ""
		if len(parts) > 0 {
			memberID = parts[len(parts)-1]
		}
		if memberID == "" || memberID == "family-members" {
			http.Error(w, `{"error":"member_id required"}`, http.StatusBadRequest)
			return
		}

		var req struct {
			SpendingLimitUzs *int64 `json:"spending_limit_uzs"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		cols := []string{"RetailerId", "MemberId"}
		vals := []interface{}{claims.UserID, memberID}
		if req.SpendingLimitUzs != nil {
			cols = append(cols, "SpendingLimitUzs")
			vals = append(vals, *req.SpendingLimitUzs)
		}
		if len(cols) == 2 {
			http.Error(w, `{"error":"no fields to update"}`, http.StatusBadRequest)
			return
		}

		_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			return txn.BufferWrite([]*spanner.Mutation{
				spanner.Update("RetailerFamilyMembers", cols, vals),
			})
		})
		if err != nil {
			log.Printf("[FAMILY_MEMBER] Update failed: %v", err)
			http.Error(w, `{"error":"update failed"}`, http.StatusInternalServerError)
			return
		}

		if invalidate != nil {
			invalidate(ctx, "profile:retailer:"+claims.UserID)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":    "UPDATED",
			"member_id": memberID,
		})
	}
}
