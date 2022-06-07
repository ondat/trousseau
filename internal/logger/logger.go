package logger

import (
	"os"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/klog/v2"
)

const (
	Info1 klog.Level = iota + 1
	Info2
	Info3
	Debug1
	Debug2
)

// NewLogger creates a new logger.
func NewLogger(logLevel klog.Level, logEncoder string) logr.Logger {
	encConfig := zap.NewProductionEncoderConfig()
	encConfig.TimeKey = "timestamp"
	encConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder

	encoder := zapcore.NewConsoleEncoder(encConfig)
	if logEncoder == "json" {
		encoder = zapcore.NewJSONEncoder(encConfig)
	}

	core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zap.NewAtomicLevelAt(zapcore.Level(-logLevel)))
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return zapr.NewLogger(logger)
}
