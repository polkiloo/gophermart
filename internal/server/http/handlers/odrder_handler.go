package handlers

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/polkiloo/gophermart/internal/app"
	domainErrors "github.com/polkiloo/gophermart/internal/domain/errors"
	"github.com/polkiloo/gophermart/internal/domain/model"
	"github.com/polkiloo/gophermart/internal/server/http/dto"
)

// OrderHandler manages order-related endpoints.
type OrderHandler struct {
	facade *app.LoyaltyFacade
}

// NewOrderHandler constructs OrderHandler.
func NewOrderHandler(facade *app.LoyaltyFacade) *OrderHandler {
	return &OrderHandler{facade: facade}
}

// Upload handles POST /api/user/orders.
func (h *OrderHandler) Upload(c *gin.Context) {
	userID := CurrentUserID(c)

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	number := strings.TrimSpace(string(body))
	if number == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	order, created, err := h.facade.UploadOrder(c.Request.Context(), userID, number)
	if err != nil {
		switch {
		case errors.Is(err, domainErrors.ErrInvalidOrderNumber):
			c.Status(http.StatusUnprocessableEntity)
		case errors.Is(err, domainErrors.ErrAlreadyExists):
			c.Status(http.StatusConflict)
		default:
			c.Status(http.StatusInternalServerError)
		}
		return
	}

	if !created && order != nil {
		c.Status(http.StatusOK)
		return
	}

	c.Status(http.StatusAccepted)
}

// List handles GET /api/user/orders.
func (h *OrderHandler) List(c *gin.Context) {
	userID := CurrentUserID(c)
	orders, err := h.facade.Orders(c.Request.Context(), userID)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	if len(orders) == 0 {
		c.Status(http.StatusNoContent)
		return
	}

	response := make([]dto.OrderResponse, 0, len(orders))
	for _, o := range orders {
		response = append(response, toOrderResponse(o))
	}

	c.JSON(http.StatusOK, response)
}

func toOrderResponse(order model.Order) dto.OrderResponse {
	return dto.OrderResponse{
		Number:     order.Number,
		Status:     string(order.Status),
		Accrual:    order.Accrual,
		UploadedAt: order.UploadedAt,
	}
}
