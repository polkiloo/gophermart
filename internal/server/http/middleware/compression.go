package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// DecompressRequest transparently handles gzip encoded requests.
func DecompressRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.Contains(c.GetHeader("Content-Encoding"), "gzip") {
			c.Next()
			return
		}

		originalBody := c.Request.Body
		reader, err := gzip.NewReader(originalBody)
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		defer reader.Close()
		defer originalBody.Close()

		c.Request.Body = io.NopCloser(reader)
		c.Request.Header.Del("Content-Encoding")
		c.Next()
	}
}
