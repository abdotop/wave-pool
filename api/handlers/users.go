package handlers

import (
	"encoding/json"
	"net/http"
)

// GetMe retrieves the authenticated user's information
type businessResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Country  string `json:"country"`
	Currency string `json:"currency"`
}

// userResponse represents the response for user information
type userResponse struct {
	ID        string           `json:"id"`
	Phone     string           `json:"phone"`
	CreatedAt string           `json:"created_at"`
	Business  businessResponse `json:"business"`
}

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

	business, err := api.db.GetBusinessByOwnerID(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "Business not found", http.StatusNotFound)
		return
	}

	resp := userResponse{
		ID:        user.ID,
		Phone:     user.Phone,
		CreatedAt: user.CreatedAt.Time.String(),
		Business: businessResponse{
			ID:       business.ID,
			Name:     business.Name,
			Country:  business.Country,
			Currency: business.Currency,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
