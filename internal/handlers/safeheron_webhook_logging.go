package handlers

import (
	"time"

	"monera-digital/internal/logger"
)

type safeheronWebhookLogLevel uint8

const (
	safeheronWebhookLogInfo safeheronWebhookLogLevel = iota
	safeheronWebhookLogWarn
	safeheronWebhookLogError
)

func emitSafeheronWebhookLog(
	level safeheronWebhookLogLevel,
	message string,
	result string,
	httpStatus int,
	startedAt time.Time,
	fields ...any,
) {
	log := logger.GetLogger()
	if log == nil {
		return
	}
	contextFields := make([]any, 0, len(fields)+8)
	contextFields = append(contextFields,
		"component", "safeheron_webhook",
		"result", result,
		"httpStatus", httpStatus,
		"elapsedMs", time.Since(startedAt).Milliseconds(),
	)
	contextFields = append(contextFields, fields...)
	switch level {
	case safeheronWebhookLogWarn:
		log.Warnw(message, contextFields...)
	case safeheronWebhookLogError:
		log.Errorw(message, contextFields...)
	default:
		log.Infow(message, contextFields...)
	}
}
