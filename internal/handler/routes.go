package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetupRoutes registers all HTTP routes.
func SetupRoutes(r *gin.Engine) {
	r.GET("/health", HealthCheck)
	r.GET("/ready", ReadyCheck)

	v1 := r.Group("/api/v1")
	{
		images := v1.Group("/images")
		{
			images.GET("", placeholderHandler)
			images.GET("/:id", placeholderHandler)
			images.POST("/scan", placeholderHandler)
		}

		tags := v1.Group("/tags")
		{
			tags.GET("", placeholderHandler)
			tags.POST("", placeholderHandler)
		}

		collections := v1.Group("/collections")
		{
			collections.GET("", placeholderHandler)
			collections.POST("", placeholderHandler)
		}
	}
}

func placeholderHandler(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "not implemented",
		"hint":  "This endpoint will be implemented in a future phase",
	})
}
