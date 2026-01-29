package errors

import (
	"fmt"
	"time"
)

// ErrorCategory represents different types of errors that can occur
type ErrorCategory string

const (
	ErrorNetwork     ErrorCategory = "network"
	ErrorAuth        ErrorCategory = "auth"
	ErrorValidation  ErrorCategory = "validation"
	ErrorConfig      ErrorCategory = "config"
	ErrorAPI         ErrorCategory = "api"
	ErrorTimeout     ErrorCategory = "timeout"
	ErrorPermission  ErrorCategory = "permission"
	ErrorUnavailable ErrorCategory = "unavailable"
	ErrorInternal    ErrorCategory = "internal"
	ErrorStream      ErrorCategory = "stream" // For SSE/streaming errors
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity string

const (
	SeverityLow      ErrorSeverity = "low"
	SeverityMedium   ErrorSeverity = "medium"
	SeverityHigh     ErrorSeverity = "high"
	SeverityCritical ErrorSeverity = "critical"
)

// ArgonautError represents a structured error with comprehensive metadata
type ArgonautError struct {
	Category    ErrorCategory          `json:"category"`
	Severity    ErrorSeverity          `json:"severity"`
	Code        string                 `json:"code"`
	Message     string                 `json:"message"`
	Details     string                 `json:"details,omitempty"`
	Cause       error                  `json:"cause,omitempty"`
	Recoverable bool                   `json:"recoverable"`
	UserAction  string                 `json:"userAction,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

// Error implements the error interface
func (e *ArgonautError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%s:%s] %s: %s", e.Category, e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%s:%s] %s", e.Category, e.Code, e.Message)
}

// Unwrap implements the error unwrapping interface
func (e *ArgonautError) Unwrap() error {
	return e.Cause
}

// Is implements error comparison for error chains
func (e *ArgonautError) Is(target error) bool {
	if t, ok := target.(*ArgonautError); ok {
		return e.Category == t.Category && e.Code == t.Code
	}
	return false
}

// WithContext adds contextual information to the error
func (e *ArgonautError) WithContext(key string, value interface{}) *ArgonautError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithCause sets the underlying cause of this error
func (e *ArgonautError) WithCause(cause error) *ArgonautError {
	e.Cause = cause
	return e
}

// WithUserAction sets a suggested user action for resolving the error
func (e *ArgonautError) WithUserAction(action string) *ArgonautError {
	e.UserAction = action
	return e
}

// New creates a new ArgonautError with the specified parameters
func New(category ErrorCategory, code, message string) *ArgonautError {
	return &ArgonautError{
		Category:    category,
		Severity:    SeverityMedium, // Default severity
		Code:        code,
		Message:     message,
		Recoverable: false, // Default to non-recoverable
		Timestamp:   time.Now(),
	}
}

// Wrap creates a new ArgonautError that wraps an existing error
func Wrap(err error, category ErrorCategory, code, message string) *ArgonautError {
	return &ArgonautError{
		Category:    category,
		Severity:    SeverityMedium,
		Code:        code,
		Message:     message,
		Cause:       err,
		Recoverable: false,
		Timestamp:   time.Now(),
	}
}

// ValidationError creates a validation-related error
func ValidationError(code, message string) *ArgonautError {
	return New(ErrorValidation, code, message).
		WithSeverity(SeverityMedium).
		WithUserAction("Please check your input and try again")
}

// ConfigError creates a configuration-related error
func ConfigError(code, message string) *ArgonautError {
	return New(ErrorConfig, code, message).
		WithSeverity(SeverityHigh).
		WithUserAction("Please check your ArgoCD configuration")
}

// TimeoutError creates a timeout-related error
func TimeoutError(code, message string) *ArgonautError {
	return New(ErrorTimeout, code, message).
		WithSeverity(SeverityMedium).
		AsRecoverable().
		WithUserAction("The operation timed out. Please try again")
}

// Helper methods for fluent error construction

// WithSeverity sets the severity level
func (e *ArgonautError) WithSeverity(severity ErrorSeverity) *ArgonautError {
	e.Severity = severity
	return e
}

// WithDetails adds additional details to the error
func (e *ArgonautError) WithDetails(details string) *ArgonautError {
	e.Details = details
	return e
}

// AsRecoverable marks the error as recoverable
func (e *ArgonautError) AsRecoverable() *ArgonautError {
	e.Recoverable = true
	return e
}

// AsNonRecoverable marks the error as non-recoverable
func (e *ArgonautError) AsNonRecoverable() *ArgonautError {
	e.Recoverable = false
	return e
}

// IsRecoverable returns true if the error can be recovered from
func (e *ArgonautError) IsRecoverable() bool {
	return e.Recoverable
}

// IsCritical returns true if the error is critical severity
func (e *ArgonautError) IsCritical() bool {
	return e.Severity == SeverityCritical
}

// IsCategory checks if the error belongs to a specific category
func (e *ArgonautError) IsCategory(category ErrorCategory) bool {
	return e.Category == category
}

// IsCode checks if the error has a specific code
func (e *ArgonautError) IsCode(code string) bool {
	return e.Code == code
}
