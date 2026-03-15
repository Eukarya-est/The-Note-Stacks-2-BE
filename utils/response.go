package utils

import (
	"github.com/gin-gonic/gin"
)

// ErrorResponse represents a standard error response structure
type ErrorResponse struct {
	Error string `json:"error"` // Human-readable error message
}

// SuccessResponse represents a standard success response with data
type SuccessResponse struct {
	Data interface{} `json:"data"` // Actual response data
}

// RespondWithError sends a JSON error response with the specified status code
// c: Gin context
// code: HTTP status code
// message: Error message to send to client
func RespondWithError(c *gin.Context, code int, message string) {
	c.JSON(code, ErrorResponse{Error: message})
}

// RespondWithSuccess sends a JSON success response with the specified status code and data
// c: Gin context
// code: HTTP status code
// data: Data to include in the response
func RespondWithSuccess(c *gin.Context, code int, data interface{}) {
	c.JSON(code, SuccessResponse{Data: data})
}

// RespondWithJSON sends a JSON response with the specified status code and payload
// c: Gin context
// code: HTTP status code
// payload: Any data structure to serialize as JSON
func RespondWithJSON(c *gin.Context, code int, payload interface{}) {
	c.JSON(code, payload)
}
