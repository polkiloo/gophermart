package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	domainErrors "github.com/polkiloo/gophermart/internal/domain/errors"
	"github.com/polkiloo/gophermart/internal/server/http/dto"
)

// BalanceHandler manages balance-related endpoints.
type BalanceHandler struct {
	facade BalanceFacade
}

// NewBalanceHandler constructs BalanceHandler.
func NewBalanceHandler(facade BalanceFacade) *BalanceHandler {
	return &BalanceHandler{facade: facade}
}

// Summary handles GET /api/user/balance.
func (h *BalanceHandler) Summary(c *gin.Context) {
	userID := CurrentUserID(c)
	summary, err := h.facade.Balance(c.Request.Context(), userID)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, dto.BalanceResponse{Current: summary.Current, Withdrawn: summary.Withdrawn})
}

// Withdraw handles POST /api/user/balance/withdraw.
func (h *BalanceHandler) Withdraw(c *gin.Context) {
	userID := CurrentUserID(c)
	var req dto.WithdrawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	err := h.facade.Withdraw(c.Request.Context(), userID, req.Order, req.Sum)
	if err != nil {
		switch {
		case errors.Is(err, domainErrors.ErrInvalidOrderNumber), errors.Is(err, domainErrors.ErrInvalidAmount):
			c.Status(http.StatusUnprocessableEntity)
		case errors.Is(err, domainErrors.ErrInsufficientBalance):
			c.Status(http.StatusPaymentRequired)
		default:
			c.Status(http.StatusInternalServerError)
		}
		return
	}

	c.Status(http.StatusOK)
}

// Withdrawals handles GET /api/user/withdrawals.
func (h *BalanceHandler) Withdrawals(c *gin.Context) {
	userID := CurrentUserID(c)
	withdrawals, err := h.facade.Withdrawals(c.Request.Context(), userID)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	if len(withdrawals) == 0 {
		c.Status(http.StatusNoContent)
		return
	}

	resp := make([]dto.WithdrawalResponse, 0, len(withdrawals))
	for _, w := range withdrawals {
		resp = append(resp, dto.WithdrawalResponse{Order: w.OrderNumber, Sum: w.Sum, ProcessedAt: w.ProcessedAt})
	}
	c.JSON(http.StatusOK, resp)
}
