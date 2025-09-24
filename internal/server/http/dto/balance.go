package dto

// BalanceResponse represents summary of loyalty points.
type BalanceResponse struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}
