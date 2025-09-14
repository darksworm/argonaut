package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	apperrors "github.com/darksworm/argonaut/pkg/errors"
)

// LogLevel represents different log levels
type LogLevel string

const (
	LevelDebug LogLevel = "DEBUG"
	LevelInfo  LogLevel = "INFO"
	LevelWarn  LogLevel = "WARN"
	LevelError LogLevel = "ERROR"
	LevelFatal LogLevel = "FATAL"
)

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	Level       LogLevel               `json:"level"`
	Message     string                 `json:"message"`
	Component   string                 `json:"component,omitempty"`
	Operation   string                 `json:"operation,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Error       string                 `json:"error,omitempty"`
	ErrorCode   string                 `json:"errorCode,omitempty"`
	ErrorCategory string               `json:"errorCategory,omitempty"`
	Duration    *time.Duration         `json:"duration,omitempty"`
	RequestID   string                 `json:"requestId,omitempty"`
}

// Logger interface defines logging operations
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Fatal(msg string, args ...interface{})

	// Structured logging methods
	LogError(err *apperrors.ArgonautError)
	LogOperation(operation string, duration time.Duration, err error)
	LogRequest(method, path string, statusCode int, duration time.Duration)

	// Context-aware logging
	WithContext(ctx context.Context) Logger
	WithComponent(component string) Logger
	WithOperation(operation string) Logger

	// Configuration
	SetLevel(level LogLevel)
	Close() error
}

// StructuredLogger provides a concrete implementation of Logger
type StructuredLogger struct {
	level        LogLevel
	component    string
	operation    string
	context      map[string]interface{}
	output       *os.File
	encoder      *json.Encoder
	stdLogger    *log.Logger
	mu           sync.RWMutex
	useJSON      bool
}

// LoggerConfig configures the structured logger
type LoggerConfig struct {
	Level      LogLevel
	OutputPath string
	UseJSON    bool
	Component  string
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(config LoggerConfig) (*StructuredLogger, error) {
	logger := &StructuredLogger{
		level:     config.Level,
		component: config.Component,
		context:   make(map[string]interface{}),
		useJSON:   config.UseJSON,
	}

	// Set up output
	if config.OutputPath != "" {
		// Ensure log directory exists
		logDir := filepath.Dir(config.OutputPath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		// Open log file
		file, err := os.OpenFile(config.OutputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}

		logger.output = file
		if config.UseJSON {
			logger.encoder = json.NewEncoder(file)
		} else {
			logger.stdLogger = log.New(file, "", 0) // No default formatting
		}
	} else {
		logger.output = os.Stderr
		if config.UseJSON {
			logger.encoder = json.NewEncoder(os.Stderr)
		} else {
			logger.stdLogger = log.New(os.Stderr, "", 0)
		}
	}

	return logger, nil
}

// Close closes the logger and any associated resources
func (l *StructuredLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.output != nil && l.output != os.Stderr && l.output != os.Stdout {
		return l.output.Close()
	}
	return nil
}

// SetLevel sets the logging level
func (l *StructuredLogger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// shouldLog checks if a message should be logged based on level
func (l *StructuredLogger) shouldLog(level LogLevel) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	levelPriority := map[LogLevel]int{
		LevelDebug: 0,
		LevelInfo:  1,
		LevelWarn:  2,
		LevelError: 3,
		LevelFatal: 4,
	}

	currentPriority := levelPriority[l.level]
	msgPriority := levelPriority[level]

	return msgPriority >= currentPriority
}

// createEntry creates a new log entry
func (l *StructuredLogger) createEntry(level LogLevel, msg string) *LogEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	entry := &LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   msg,
		Component: l.component,
		Operation: l.operation,
	}

	// Copy context to avoid race conditions
	if len(l.context) > 0 {
		entry.Context = make(map[string]interface{})
		for k, v := range l.context {
			entry.Context[k] = v
		}
	}

	return entry
}

// writeEntry writes a log entry to the output
func (l *StructuredLogger) writeEntry(entry *LogEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.useJSON {
		l.encoder.Encode(entry)
	} else {
		// Human-readable format
		timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")

		// Build context string
		var contextStr string
		if entry.Context != nil {
			var parts []string
			for k, v := range entry.Context {
				parts = append(parts, fmt.Sprintf("%s=%v", k, v))
			}
			if len(parts) > 0 {
				contextStr = fmt.Sprintf(" [%s]", strings.Join(parts, " "))
			}
		}

		// Build component/operation prefix
		var prefix string
		if entry.Component != "" {
			prefix = fmt.Sprintf("[%s", entry.Component)
			if entry.Operation != "" {
				prefix += fmt.Sprintf(":%s", entry.Operation)
			}
			prefix += "] "
		}

		logMsg := fmt.Sprintf("%s %s %s%s%s",
			timestamp, entry.Level, prefix, entry.Message, contextStr)

		if entry.Error != "" {
			logMsg += fmt.Sprintf(" | Error: %s", entry.Error)
		}

		l.stdLogger.Println(logMsg)
	}
}

// Debug logs a debug message
func (l *StructuredLogger) Debug(msg string, args ...interface{}) {
	if !l.shouldLog(LevelDebug) {
		return
	}

	entry := l.createEntry(LevelDebug, fmt.Sprintf(msg, args...))
	l.writeEntry(entry)
}

// Info logs an info message
func (l *StructuredLogger) Info(msg string, args ...interface{}) {
	if !l.shouldLog(LevelInfo) {
		return
	}

	entry := l.createEntry(LevelInfo, fmt.Sprintf(msg, args...))
	l.writeEntry(entry)
}

// Warn logs a warning message
func (l *StructuredLogger) Warn(msg string, args ...interface{}) {
	if !l.shouldLog(LevelWarn) {
		return
	}

	entry := l.createEntry(LevelWarn, fmt.Sprintf(msg, args...))
	l.writeEntry(entry)
}

// Error logs an error message
func (l *StructuredLogger) Error(msg string, args ...interface{}) {
	if !l.shouldLog(LevelError) {
		return
	}

	entry := l.createEntry(LevelError, fmt.Sprintf(msg, args...))
	l.writeEntry(entry)
}

// Fatal logs a fatal message and exits
func (l *StructuredLogger) Fatal(msg string, args ...interface{}) {
	entry := l.createEntry(LevelFatal, fmt.Sprintf(msg, args...))
	l.writeEntry(entry)
	os.Exit(1)
}

// LogError logs a structured error
func (l *StructuredLogger) LogError(err *apperrors.ArgonautError) {
	if !l.shouldLog(LevelError) || err == nil {
		return
	}

	entry := l.createEntry(LevelError, err.Message)
	entry.Error = err.Error()
	entry.ErrorCode = err.Code
	entry.ErrorCategory = string(err.Category)

	// Add error context
	if entry.Context == nil {
		entry.Context = make(map[string]interface{})
	}
	for k, v := range err.Context {
		entry.Context[k] = v
	}

	l.writeEntry(entry)
}

// LogOperation logs an operation with duration and optional error
func (l *StructuredLogger) LogOperation(operation string, duration time.Duration, err error) {
	level := LevelInfo
	if err != nil {
		level = LevelError
	}

	if !l.shouldLog(level) {
		return
	}

	msg := fmt.Sprintf("Operation %s completed", operation)
	if err != nil {
		msg = fmt.Sprintf("Operation %s failed", operation)
	}

	entry := l.createEntry(level, msg)
	entry.Operation = operation
	entry.Duration = &duration

	if err != nil {
		entry.Error = err.Error()

		// Add structured error information if available
		if argErr, ok := err.(*apperrors.ArgonautError); ok {
			entry.ErrorCode = argErr.Code
			entry.ErrorCategory = string(argErr.Category)
		}
	}

	l.writeEntry(entry)
}

// LogRequest logs HTTP request information
func (l *StructuredLogger) LogRequest(method, path string, statusCode int, duration time.Duration) {
	level := LevelInfo
	if statusCode >= 400 {
		level = LevelWarn
	}
	if statusCode >= 500 {
		level = LevelError
	}

	if !l.shouldLog(level) {
		return
	}

	msg := fmt.Sprintf("%s %s - %d", method, path, statusCode)

	entry := l.createEntry(level, msg)
	entry.Duration = &duration

	if entry.Context == nil {
		entry.Context = make(map[string]interface{})
	}
	entry.Context["httpMethod"] = method
	entry.Context["httpPath"] = path
	entry.Context["httpStatus"] = statusCode

	l.writeEntry(entry)
}

// WithContext returns a logger with additional context
func (l *StructuredLogger) WithContext(ctx context.Context) Logger {
	newLogger := l.clone()

	// Extract context values if available
	if requestID, ok := ctx.Value("requestId").(string); ok {
		if newLogger.context == nil {
			newLogger.context = make(map[string]interface{})
		}
		newLogger.context["requestId"] = requestID
	}

	return newLogger
}

// WithComponent returns a logger with a component name
func (l *StructuredLogger) WithComponent(component string) Logger {
	newLogger := l.clone()
	newLogger.component = component
	return newLogger
}

// WithOperation returns a logger with an operation name
func (l *StructuredLogger) WithOperation(operation string) Logger {
	newLogger := l.clone()
	newLogger.operation = operation
	return newLogger
}

// clone creates a copy of the logger for context-specific logging
func (l *StructuredLogger) clone() *StructuredLogger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	newLogger := &StructuredLogger{
		level:     l.level,
		component: l.component,
		operation: l.operation,
		output:    l.output,
		encoder:   l.encoder,
		stdLogger: l.stdLogger,
		useJSON:   l.useJSON,
		context:   make(map[string]interface{}),
	}

	// Copy context
	for k, v := range l.context {
		newLogger.context[k] = v
	}

	return newLogger
}

// Default logger instance
var defaultLogger *StructuredLogger
var defaultLoggerOnce sync.Once

// GetDefaultLogger returns the default logger instance
func GetDefaultLogger() Logger {
	defaultLoggerOnce.Do(func() {
		config := LoggerConfig{
			Level:      LevelInfo,
			OutputPath: "logs/app.log",
			UseJSON:    false, // Human-readable by default
			Component:  "argonaut",
		}

		logger, err := NewStructuredLogger(config)
		if err != nil {
			// Fallback to stderr logger
			logger = &StructuredLogger{
				level:     LevelInfo,
				component: "argonaut",
				context:   make(map[string]interface{}),
				output:    os.Stderr,
				stdLogger: log.New(os.Stderr, "", 0),
				useJSON:   false,
			}
		}

		defaultLogger = logger
	})

	return defaultLogger
}

// Package-level convenience functions

// Debug logs a debug message using the default logger
func Debug(msg string, args ...interface{}) {
	GetDefaultLogger().Debug(msg, args...)
}

// Info logs an info message using the default logger
func Info(msg string, args ...interface{}) {
	GetDefaultLogger().Info(msg, args...)
}

// Warn logs a warning message using the default logger
func Warn(msg string, args ...interface{}) {
	GetDefaultLogger().Warn(msg, args...)
}

// Error logs an error message using the default logger
func Error(msg string, args ...interface{}) {
	GetDefaultLogger().Error(msg, args...)
}

// Fatal logs a fatal message using the default logger
func Fatal(msg string, args ...interface{}) {
	GetDefaultLogger().Fatal(msg, args...)
}

// LogError logs a structured error using the default logger
func LogError(err *apperrors.ArgonautError) {
	GetDefaultLogger().LogError(err)
}

// LogOperation logs an operation using the default logger
func LogOperation(operation string, duration time.Duration, err error) {
	GetDefaultLogger().LogOperation(operation, duration, err)
}

// LogRequest logs a request using the default logger
func LogRequest(method, path string, statusCode int, duration time.Duration) {
	GetDefaultLogger().LogRequest(method, path, statusCode, duration)
}