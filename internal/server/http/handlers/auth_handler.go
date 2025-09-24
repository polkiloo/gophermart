package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	domainErrors "github.com/polkiloo/gophermart/internal/domain/errors"
	"github.com/polkiloo/gophermart/internal/server/http/dto"
	"github.com/polkiloo/gophermart/internal/server/http/middleware"
)

// AuthHandler processes registration and login.
type AuthHandler struct {
	facade AuthFacade
}

// NewAuthHandler creates AuthHandler instance.
func NewAuthHandler(facade AuthFacade) *AuthHandler {
	return &AuthHandler{facade: facade}
}

// Register handles POST /api/user/register.
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.AuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	token, err := h.facade.Register(c.Request.Context(), req.Login, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, domainErrors.ErrInvalidCredentials):
			c.Status(http.StatusBadRequest)
		case errors.Is(err, domainErrors.ErrAlreadyExists):
			c.Status(http.StatusConflict)
		default:
			c.Status(http.StatusInternalServerError)
		}
		return
	}

	middleware.SetAuthCookie(c, token)
	c.Status(http.StatusOK)
}

// Login handles POST /api/user/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.AuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	token, err := h.facade.Authenticate(c.Request.Context(), req.Login, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, domainErrors.ErrInvalidCredentials):
			c.Status(http.StatusUnauthorized)
		default:
			c.Status(http.StatusInternalServerError)
		}
		return
	}

	middleware.SetAuthCookie(c, token)
	c.Status(http.StatusOK)
}
