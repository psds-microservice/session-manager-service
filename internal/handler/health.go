package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "session-manager-service",
		"time":    time.Now().Unix(),
	})
}

func Ready(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ready": true})
}
