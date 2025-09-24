package repository

// Factory describes access to different domain repositories.
type Factory interface {
	Users() UserRepository
	Orders() OrderRepository
	Balances() BalanceRepository
	Withdrawals() WithdrawalRepository
}
