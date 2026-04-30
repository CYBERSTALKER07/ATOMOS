package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

type AuditLogRecord struct {
	LogID        string `json:"log_id"`
	ActorID      string `json:"actor_id"`
	ActorRole    string `json:"actor_role"`
	Action       string `json:"action"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	Metadata     string `json:"metadata"`
	CreatedAt    string `json:"created_at"`
}

type AuditLogResponse struct {
	Data   []AuditLogRecord `json:"data"`
	Limit  int64            `json:"limit"`
	Offset int64            `json:"offset"`
}

// HandleGetAuditLog returns paginated audit log entries with optional filters.
// Query params: limit, offset, actor_id, resource_type, action, from, to
func HandleGetAuditLog(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		limit := int64(50)
		offset := int64(0)
		if l, err := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64); err == nil && l > 0 {
			limit = l
			if limit > 500 {
				limit = 500
			}
		}
		if o, err := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 64); err == nil && o >= 0 {
			offset = o
		}

		sql := `SELECT LogId, ActorId, ActorRole, Action, ResourceType, ResourceId, Metadata, CreatedAt
			FROM AuditLog
			WHERE 1=1`
		params := map[string]interface{}{}

		if actorID := strings.TrimSpace(r.URL.Query().Get("actor_id")); actorID != "" {
			sql += ` AND ActorId = @actorId`
			params["actorId"] = actorID
		}
		if resourceType := strings.TrimSpace(r.URL.Query().Get("resource_type")); resourceType != "" {
			sql += ` AND ResourceType = @resourceType`
			params["resourceType"] = resourceType
		}
		if action := strings.TrimSpace(r.URL.Query().Get("action")); action != "" {
			sql += ` AND Action = @action`
			params["action"] = action
		}
		if fromStr := strings.TrimSpace(r.URL.Query().Get("from")); fromStr != "" {
			if from, err := time.Parse(time.RFC3339, fromStr); err == nil {
				sql += ` AND CreatedAt >= @from`
				params["from"] = from
			}
		}
		if toStr := strings.TrimSpace(r.URL.Query().Get("to")); toStr != "" {
			if to, err := time.Parse(time.RFC3339, toStr); err == nil {
				sql += ` AND CreatedAt <= @to`
				params["to"] = to
			}
		}

		sql += ` ORDER BY CreatedAt DESC LIMIT @limit OFFSET @offset`
		params["limit"] = limit
		params["offset"] = offset

		iter := client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
		defer iter.Stop()

		resp := AuditLogResponse{Data: []AuditLogRecord{}, Limit: limit, Offset: offset}
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				http.Error(w, `{"error":"audit_log_query_failed"}`, http.StatusInternalServerError)
				return
			}

			var rec AuditLogRecord
			var metadata spanner.NullString
			var createdAt time.Time
			if err := row.Columns(&rec.LogID, &rec.ActorID, &rec.ActorRole, &rec.Action, &rec.ResourceType, &rec.ResourceID, &metadata, &createdAt); err != nil {
				continue
			}
			if metadata.Valid {
				rec.Metadata = metadata.StringVal
			}
			rec.CreatedAt = createdAt.Format(time.RFC3339)
			resp.Data = append(resp.Data, rec)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
