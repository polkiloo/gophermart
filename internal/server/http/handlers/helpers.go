package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/polkiloo/gophermart/internal/server/http/middleware"
)

// CurrentUserID extracts authenticated user identifier from context.
func CurrentUserID(c *gin.Context) int64 {
	val, ok := c.Get(middleware.UserIDContextKey)
	if !ok {
		return 0
	}
	id, _ := val.(int64)
	return id
}
