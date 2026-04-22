package admin

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

type SystemConfigEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// HandleSystemConfig serves GET (list all) and PUT (upsert entries) for /v1/admin/config
func HandleSystemConfig(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getSystemConfig(w, r, client)
		case http.MethodPut:
			putSystemConfig(w, r, client)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

func getSystemConfig(w http.ResponseWriter, r *http.Request, client *spanner.Client) {
	stmt := spanner.Statement{SQL: `SELECT ConfigKey, ConfigValue FROM SystemConfig ORDER BY ConfigKey`}
	iter := client.Single().Query(r.Context(), stmt)
	defer iter.Stop()

	result := map[string]string{}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("[ADMIN CONFIG] query error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		var key, value string
		if err := row.Columns(&key, &value); err != nil {
			log.Printf("[ADMIN CONFIG] parse error: %v", err)
			continue
		}
		result[key] = value
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func putSystemConfig(w http.ResponseWriter, r *http.Request, client *spanner.Client) {
	var entries []SystemConfigEntry
	if err := json.NewDecoder(r.Body).Decode(&entries); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}
	if len(entries) == 0 {
		http.Error(w, "entries array required", http.StatusBadRequest)
		return
	}

	mutations := make([]*spanner.Mutation, 0, len(entries))
	for _, e := range entries {
		if e.Key == "" {
			continue
		}
		mutations = append(mutations, spanner.InsertOrUpdate("SystemConfig",
			[]string{"ConfigKey", "ConfigValue", "UpdatedAt"},
			[]interface{}{e.Key, e.Value, spanner.CommitTimestamp},
		))
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if _, err := client.Apply(ctx, mutations); err != nil {
		log.Printf("[ADMIN CONFIG] write error: %v", err)
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
