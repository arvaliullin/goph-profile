package handlers

import (
	"encoding/json"
	"net/http"
)

// writeJSON пишет тело ответа в формате JSON.
func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return
	}
}

// writeError пишет JSON-ответ с кодом ошибки.
func writeError(w http.ResponseWriter, code int, body any) {
	writeJSON(w, code, body)
}
