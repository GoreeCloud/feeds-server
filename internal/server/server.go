package server

import (
	"encoding/json"
	"net/http"
)

const (
	ProductName     = "GoreeCloud Feeds Server"
	APIVersion      = "v1"
	ProtocolVersion = "0.1.0-dev"
	Lifecycle       = "development"
)

type capabilityResponse struct {
	Product         string   `json:"product"`
	APIVersion      string   `json:"api_version"`
	ProtocolVersion string   `json:"protocol_version"`
	Lifecycle       string   `json:"lifecycle"`
	Capabilities    []string `json:"capabilities"`
}

type healthResponse struct {
	Status string `json:"status"`
}

func NewHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/capabilities", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, capabilityResponse{
			Product:         ProductName,
			APIVersion:      APIVersion,
			ProtocolVersion: ProtocolVersion,
			Lifecycle:       Lifecycle,
			Capabilities:    []string{},
		})
	})

	mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
	})

	mux.HandleFunc("GET /health/ready", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
	})

	return mux
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(true)
	_ = encoder.Encode(value)
}
