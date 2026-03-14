package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger logs request method, path, status and latency.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		latency := time.Since(start)
		log.Printf("method=%s path=%s status=%d latency=%s", c.Request.Method, c.Request.URL.Path, c.Writer.Status(), latency)
	}
}
