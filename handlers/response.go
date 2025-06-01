package handlers

import (
	"github.com/gin-gonic/gin"
)

// APIError represents a standard error response format.
type APIError struct {
	Error  string `json:"error"`
	Status int    `json:"status,omitempty"` // Optional: include HTTP status in body
}

// RespondWithError sends a JSON error response.
func RespondWithError(c *gin.Context, code int, message string) {
	c.JSON(code, APIError{Error: message, Status: code})
}

// RespondWithJSON sends a JSON success response.
func RespondWithJSON(c *gin.Context, code int, payload interface{}) {
	c.JSON(code, payload)
}
