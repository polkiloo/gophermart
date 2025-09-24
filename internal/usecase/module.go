package usecase

import "go.uber.org/fx"

// Module provides core business use cases to the fx container.
var Module = fx.Provide(
	NewAuthUseCase,
	NewOrderUseCase,
	NewBalanceUseCase,
)
