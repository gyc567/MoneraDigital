// internal/logger/logger.go
package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is the global sugared logger instance
var Logger *zap.SugaredLogger

// Init initializes the global logger
func Init(environment string) error {
	var config zap.Config

	if environment == "production" {
		config = zap.NewProductionConfig()
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.LevelKey = "level"

	zapLogger, err := config.Build()
	if err != nil {
		return err
	}

	Logger = zapLogger.Sugar()
	return nil
}

// GetLogger returns the global logger instance
func GetLogger() *zap.SugaredLogger {
	return Logger
}

// Info logs an info message
func Info(message string, keysAndValues ...interface{}) {
	Logger.Infow(message, keysAndValues...)
}

// Warn logs a warning message
func Warn(message string, keysAndValues ...interface{}) {
	Logger.Warnw(message, keysAndValues...)
}

// Error logs an error message
func Error(message string, keysAndValues ...interface{}) {
	Logger.Errorw(message, keysAndValues...)
}

// Debug logs a debug message
func Debug(message string, keysAndValues ...interface{}) {
	Logger.Debugw(message, keysAndValues...)
}

// Fatal logs a fatal message and then exits
func Fatal(message string, keysAndValues ...interface{}) {
	Logger.Fatalw(message, keysAndValues...)
}

// With creates a child logger with the given fields
func With(keysAndValues ...interface{}) *zap.SugaredLogger {
	return Logger.With(keysAndValues...)
}
