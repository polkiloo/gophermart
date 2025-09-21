package dto

// AuthRequest describes login/password payload.
type AuthRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}
