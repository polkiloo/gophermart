package model

// BalanceSummary aggregates current and withdrawn loyalty points.
type BalanceSummary struct {
	Current   float64
	Withdrawn float64
}
