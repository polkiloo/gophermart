package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/polkiloo/gophermart/internal/app"
	pkgAuth "github.com/polkiloo/gophermart/internal/pkg/auth"
)

const (
	// UserIDContextKey is a gin context key for authenticated user identifier.
	UserIDContextKey = "userID"
	authCookieName   = "gophermart_token"
)

// AuthRequired ensures user is authenticated before accessing handler.
func AuthRequired(facade *app.LoyaltyFacade) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		userID, err := facade.ParseToken(token)
		if err != nil {
			if err == pkgAuth.ErrInvalidToken {
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.Set(UserIDContextKey, userID)
		c.Next()
	}
}

func extractToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return strings.TrimSpace(authHeader[7:])
	}

	if cookie, err := c.Cookie(authCookieName); err == nil {
		return cookie
	}
	return ""
}

// SetAuthCookie writes auth token cookie to response.
func SetAuthCookie(c *gin.Context, token string) {
	c.SetCookie(authCookieName, token, 0, "/", "", false, true)
	c.Header("Authorization", "Bearer "+token)
}
