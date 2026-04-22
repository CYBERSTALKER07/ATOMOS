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
	MemberID   string `json:"member_id" spanner:"MemberId"`
	RetailerID string `json:"retailer_id" spanner:"RetailerId"`
	Nickname   string `json:"nickname" spanner:"Nickname"`
	PhotoURL   string `json:"photo_url,omitempty" spanner:"PhotoUrl"`
	CreatedAt  string `json:"created_at" spanner:"CreatedAt"`
}

// HandleListFamilyMembers returns all family members for the logged-in retailer.
// GET /v1/retailer/family-members (RETAILER role)
func HandleListFamilyMembers(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, ok := r.Context().Value(ClaimsContextKey).(*LabClaims)
		if !ok || claims == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		stmt := spanner.Statement{
			SQL:    `SELECT MemberId, RetailerId, Nickname, PhotoUrl, CreatedAt FROM RetailerFamilyMembers WHERE RetailerId = @rid ORDER BY CreatedAt`,
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
			var createdAt time.Time
			if err := row.Columns(&m.MemberID, &m.RetailerID, &m.Nickname, &photo, &createdAt); err != nil {
				continue
			}
			if photo.Valid {
				m.PhotoURL = photo.StringVal
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
func HandleCreateFamilyMember(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, ok := r.Context().Value(ClaimsContextKey).(*LabClaims)
		if !ok || claims == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req struct {
			Nickname string `json:"nickname"`
			PhotoURL string `json:"photo_url"`
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
func HandleDeleteFamilyMember(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, ok := r.Context().Value(ClaimsContextKey).(*LabClaims)
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

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":    "DELETED",
			"member_id": memberID,
		})
	}
}
