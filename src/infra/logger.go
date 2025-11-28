package infra

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// LogLevel represents the severity level of a log message
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ToZerologLevel converts LogLevel to zerolog.Level
func (l LogLevel) ToZerologLevel() zerolog.Level {
	switch l {
	case DEBUG:
		return zerolog.DebugLevel
	case INFO:
		return zerolog.InfoLevel
	case WARN:
		return zerolog.WarnLevel
	case ERROR:
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

// Logger interface defines the logging methods
type Logger interface {
	Info(msg string, fields map[string]interface{})
	Error(msg string, err error, fields map[string]interface{})
	Warn(msg string, fields map[string]interface{})
	Debug(msg string, fields map[string]interface{})
	SetLevel(level LogLevel)
}

// structuredLogger implements the Logger interface with structured logging using zerolog
type structuredLogger struct {
	logger zerolog.Logger
	level  LogLevel
	mu     sync.RWMutex
}

var (
	instance *structuredLogger
	once     sync.Once
)

// GetLogger returns the singleton logger instance
func GetLogger() Logger {
	once.Do(func() {
		level := getLogLevelFromEnv()

		// Check if prettify mode is enabled (default: true for development)
		prettify := getPrettifyFromEnv()

		// Set global zerolog level
		zerolog.SetGlobalLevel(level.ToZerologLevel())

		// Configure time format
		zerolog.TimeFieldFormat = time.RFC3339

		var logger zerolog.Logger

		if prettify {
			// Use console writer for pretty, colorized output
			output := zerolog.ConsoleWriter{
				Out:        os.Stdout,
				TimeFormat: time.RFC3339,
				NoColor:    false, // Enable colors
			}
			logger = zerolog.New(output).With().Timestamp().Logger()
		} else {
			// Use JSON output for production
			logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
		}

		instance = &structuredLogger{
			logger: logger,
			level:  level,
		}
	})
	return instance
}

// getLogLevelFromEnv reads the LOG_LEVEL environment variable and returns the corresponding LogLevel
func getLogLevelFromEnv() LogLevel {
	levelStr := os.Getenv("LOG_LEVEL")
	if levelStr == "" {
		return INFO // Default to INFO level
	}

	switch levelStr {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN":
		return WARN
	case "ERROR":
		return ERROR
	default:
		return INFO
	}
}

// getPrettifyFromEnv reads the LOG_PRETTIFY environment variable
// Returns true if prettify is enabled (default: true)
func getPrettifyFromEnv() bool {
	prettifyStr := os.Getenv("LOG_PRETTIFY")
	if prettifyStr == "" {
		return true // Default to prettify enabled
	}
	return prettifyStr == "true" || prettifyStr == "1" || prettifyStr == "yes"
}

// SetLevel sets the minimum log level
func (l *structuredLogger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
	zerolog.SetGlobalLevel(level.ToZerologLevel())
}

// Info logs an informational message
func (l *structuredLogger) Info(msg string, fields map[string]interface{}) {
	event := l.logger.Info()
	l.addFields(event, fields)
	event.Msg(msg)
}

// Error logs an error message
func (l *structuredLogger) Error(msg string, err error, fields map[string]interface{}) {
	event := l.logger.Error()
	if err != nil {
		event = event.Err(err)
	}
	l.addFields(event, fields)
	event.Msg(msg)
}

// Warn logs a warning message
func (l *structuredLogger) Warn(msg string, fields map[string]interface{}) {
	event := l.logger.Warn()
	l.addFields(event, fields)
	event.Msg(msg)
}

// Debug logs a debug message
func (l *structuredLogger) Debug(msg string, fields map[string]interface{}) {
	event := l.logger.Debug()
	l.addFields(event, fields)
	event.Msg(msg)
}

// addFields adds fields to a zerolog event
func (l *structuredLogger) addFields(event *zerolog.Event, fields map[string]interface{}) {
	for k, v := range fields {
		event = event.Interface(k, v)
	}
}

// FormatError is a helper function for formatting errors
func FormatError(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%v", err)
}
