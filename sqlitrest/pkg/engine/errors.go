package engine

import (
	"fmt"
	"net/http"
)

// ErrorType représente les types d'erreurs PostgREST
type ErrorType string

const (
	ErrorTypeAuth       ErrorType = "PGRST301" // Authentication failed
	ErrorTypePermission ErrorType = "PGRST302" // Permission denied
	ErrorTypeNotFound   ErrorType = "PGRST303" // Resource not found
	ErrorTypeValidation ErrorType = "PGRST304" // Validation error
	ErrorTypeInternal   ErrorType = "PGRST500" // Internal server error
	ErrorTypeDatabase   ErrorType = "PGRST501" // Database error
)

// APIError représente une erreur API compatible PostgREST
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
	Hint    string `json:"hint,omitempty"`
	Status  int    `json:"-"`
}

// Error implémente l'interface error
func (e *APIError) Error() string {
	return e.Message
}

// NewAPIError crée une nouvelle erreur API
func NewAPIError(errorType ErrorType, message string, details ...string) *APIError {
	errorMap := map[ErrorType]struct {
		code   string
		status int
	}{
		ErrorTypeAuth:       {"PGRST301", http.StatusUnauthorized},
		ErrorTypePermission: {"PGRST302", http.StatusForbidden},
		ErrorTypeNotFound:   {"PGRST303", http.StatusNotFound},
		ErrorTypeValidation: {"PGRST304", http.StatusBadRequest},
		ErrorTypeInternal:   {"PGRST500", http.StatusInternalServerError},
		ErrorTypeDatabase:   {"PGRST501", http.StatusInternalServerError},
	}

	info, exists := errorMap[errorType]
	if !exists {
		info = errorMap[ErrorTypeInternal]
	}

	err := &APIError{
		Code:    info.code,
		Message: message,
		Status:  info.status,
	}

	if len(details) > 0 {
		err.Details = details[0]
	}

	return err
}

// Erreurs prédéfinies
var (
	ErrAuthenticationFailed = NewAPIError(ErrorTypeAuth, "Authentication failed", "Invalid or missing JWT token")
	ErrPermissionDenied     = NewAPIError(ErrorTypePermission, "Permission denied", "You don't have permission to access this resource")
	ErrResourceNotFound     = NewAPIError(ErrorTypeNotFound, "Resource not found", "The requested resource does not exist")
	ErrInvalidRequest       = NewAPIError(ErrorTypeValidation, "Invalid request", "The request parameters are invalid")
	ErrDatabaseError        = NewAPIError(ErrorTypeDatabase, "Database error", "An error occurred while executing the database query")
	ErrInternalError        = NewAPIError(ErrorTypeInternal, "Internal server error", "An unexpected error occurred")
)

// ErrorHandler gère les erreurs de manière cohérente
type ErrorHandler struct{}

// NewErrorHandler crée un nouveau gestionnaire d'erreurs
func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{}
}

// HandleError convertit une erreur en APIError
func (h *ErrorHandler) HandleError(err error) *APIError {
	if err == nil {
		return nil
	}

	// Si c'est déjà une APIError, la retourner directement
	if apiErr, ok := err.(*APIError); ok {
		return apiErr
	}

	// Analyser le message d'erreur pour déterminer le type
	errMsg := err.Error()

	// Erreurs d'authentification
	if contains(errMsg, []string{"authentication", "unauthorized", "invalid token", "jwt"}) {
		return NewAPIError(ErrorTypeAuth, "Authentication failed", errMsg)
	}

	// Erreurs de permission
	if contains(errMsg, []string{"permission", "forbidden", "access denied", "policy"}) {
		return NewAPIError(ErrorTypePermission, "Permission denied", errMsg)
	}

	// Erreurs de validation
	if contains(errMsg, []string{"invalid", "bad request", "validation", "parse"}) {
		return NewAPIError(ErrorTypeValidation, "Invalid request", errMsg)
	}

	// Erreurs de base de données
	if contains(errMsg, []string{"database", "sql", "query", "execution failed"}) {
		return NewAPIError(ErrorTypeDatabase, "Database error", errMsg)
	}

	// Erreur interne par défaut
	return NewAPIError(ErrorTypeInternal, "Internal server error", errMsg)
}

// contains vérifie si le message contient une des chaînes
func contains(message string, substrings []string) bool {
	for _, substr := range substrings {
		if len(message) >= len(substr) {
			for i := 0; i <= len(message)-len(substr); i++ {
				if message[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}

// WrapError enveloppe une erreur avec contexte
func (h *ErrorHandler) WrapError(err error, context string) *APIError {
	if err == nil {
		return nil
	}

	apiErr := h.HandleError(err)
	if apiErr.Details != "" {
		apiErr.Details = fmt.Sprintf("%s: %s", context, apiErr.Details)
	} else {
		apiErr.Details = context
	}

	return apiErr
}

// ValidationError crée une erreur de validation
func (h *ErrorHandler) ValidationError(field, message string) *APIError {
	return NewAPIError(ErrorTypeValidation, fmt.Sprintf("Validation error on field %s", field), message)
}

// DatabaseError crée une erreur de base de données
func (h *ErrorHandler) DatabaseError(query string, err error) *APIError {
	return h.WrapError(err, fmt.Sprintf("Database query failed: %s", query))
}

// AuthError crée une erreur d'authentification
func (h *ErrorHandler) AuthError(reason string) *APIError {
	return NewAPIError(ErrorTypeAuth, "Authentication failed", reason)
}

// PermissionError crée une erreur de permission
func (h *ErrorHandler) PermissionError(resource string) *APIError {
	return NewAPIError(ErrorTypePermission, "Permission denied", fmt.Sprintf("No permission to access %s", resource))
}
