package auth

import "net/http"

func writeScopeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	http.Error(w, `{"error":"`+message+`"}`, status)
}
