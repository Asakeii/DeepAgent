package handler

import (
	"encoding/json"
	"net/http"

	"deepAgent/internal/infra"
)

type healthResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks,omitempty"`
}

func Healthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeHealth(w, http.StatusOK, healthResponse{Status: "ok"})
}

func Readyz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	checks := map[string]string{}
	ready := true

	if infra.DB == nil {
		checks["db"] = "not_initialized"
		ready = false
	} else if err := infra.DB.PingContext(r.Context()); err != nil {
		checks["db"] = err.Error()
		ready = false
	} else {
		checks["db"] = "ok"
	}

	if infra.ChatModel == nil {
		checks["model"] = "not_initialized"
		ready = false
	} else {
		checks["model"] = "ok"
	}

	if infra.RDB == nil {
		checks["redis"] = "disabled"
	} else if err := infra.RDB.Ping(r.Context()).Err(); err != nil {
		checks["redis"] = err.Error()
		ready = false
	} else {
		checks["redis"] = "ok"
	}

	status := http.StatusOK
	respStatus := "ready"
	if !ready {
		status = http.StatusServiceUnavailable
		respStatus = "not_ready"
	}
	writeHealth(w, status, healthResponse{Status: respStatus, Checks: checks})
}

func writeHealth(w http.ResponseWriter, status int, resp healthResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}
