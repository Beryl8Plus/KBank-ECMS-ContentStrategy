package app

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// IngressRuleManagement is the initial endpoint for the new Rule Management API.
func IngressRuleManagement(c *gin.Context) {
	c.Header("Content-Type", "application/json; charset=UTF-8")
	c.Header("Request-ID", c.GetHeader("requestID"))
	c.Header("Request-Time", time.Now().Format("2006-01-02T15:04:05.000"))
	c.Header("Status-Code", "200")
	c.Header("Status-Msg", "OK")
	c.Header("Access-Control-Expose-Headers", "Request-ID, Request-Time, Status-Code, Status-Msg")

	c.JSON(http.StatusOK, gin.H{
		"service": "rule-management",
		"status":  "initialized",
		"message": "New service API is ready for implementation.",
	})
}
