package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

type AccessResponse struct {
	UserID    string `json:"user_id"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

func HandleAccess(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]
	
	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}
	
	response := AccessResponse{
		UserID:    userID,
		Message:   "Access granted",
		Timestamp: "2025-01-01T00:00:00Z", // TODO: use actual timestamp
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}