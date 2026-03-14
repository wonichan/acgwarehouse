package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthCheck reports service liveness.
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "ACGWarehouse is running",
		"version": "1.0.0",
	})
}

// ReadyCheck reports if server can accept requests.
func ReadyCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
	})
}
