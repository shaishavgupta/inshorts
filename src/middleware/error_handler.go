package middleware

import (
	"news-inshorts/src/infra"

	"github.com/gofiber/fiber/v2"
)

// AppError represents a custom application error with HTTP status code
type AppError struct {
	Code    int    `json:"-"`
	Message string `json:"error"`
	Err     error  `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

// NewAppError creates a new AppError
func NewAppError(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Predefined error types
var (
	ErrInvalidInput   = &AppError{Code: 400, Message: "Invalid input parameters"}
	ErrNotFound       = &AppError{Code: 404, Message: "Resource not found"}
	ErrInternalServer = &AppError{Code: 500, Message: "Internal server error"}
	ErrLLMUnavailable = &AppError{Code: 503, Message: "LLM service unavailable"}
	ErrDatabaseError  = &AppError{Code: 503, Message: "Database connection error"}
)

// ErrorHandler is a Fiber error handling middleware
func ErrorHandler(c *fiber.Ctx, err error) error {
	log := infra.GetLogger()

	// Default to src server error
	appErr, ok := err.(*AppError)
	if !ok {
		// Check if it's a Fiber error
		if fiberErr, ok := err.(*fiber.Error); ok {
			appErr = &AppError{
				Code:    fiberErr.Code,
				Message: fiberErr.Message,
				Err:     err,
			}
		} else {
			// Unknown error type
			appErr = &AppError{
				Code:    500,
				Message: "Internal server error",
				Err:     err,
			}
		}
	}

	// Log the error with context
	log.Error("Request failed", appErr.Err, map[string]interface{}{
		"path":   c.Path(),
		"method": c.Method(),
		"code":   appErr.Code,
		"ip":     c.IP(),
	})

	// Return JSON error response
	return c.Status(appErr.Code).JSON(fiber.Map{
		"error": appErr.Message,
	})
}
