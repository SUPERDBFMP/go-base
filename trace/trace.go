package trace

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/sirupsen/logrus"
)

// TraceIdKey trace id key
const TraceIdKey = "traceId"

// GetOrGenerateTraceId get or generate trace id
func GetOrGenerateTraceId(ctx context.Context) string {
	traceId, ok := ctx.Value(TraceIdKey).(string)
	if !ok || traceId == "" {
		return GenerateTraceId()
	}
	return traceId
}

func GenerateTraceId() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func BuildTraceField(ctx context.Context) logrus.Fields {
	if traceId, ok := ctx.Value(TraceIdKey).(string); ok {
		return logrus.Fields{
			TraceIdKey: traceId,
		}
	}
	return nil
}
