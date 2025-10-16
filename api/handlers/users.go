package handlers

import (
	"encoding/json"
	"net/http"
)

func (api *API) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, err := GetUserFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := api.db.GetUserByID(r.Context(), userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"id": user.ID, "phone": user.Phone, "created_at": user.CreatedAt.Time.String()})
}
