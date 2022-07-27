package logger

import (
	"flag"
	"os"
	"strconv"

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

func init() {
	klog.InitFlags(nil)
}

// InitializeLogging initializing global logging.
func InitializeLogging(logEncoder string) error {
	v := flag.CommandLine.Lookup("v").Value.String()

	logLevel, err := strconv.Atoi(v)
	if err != nil {
		klog.Error(err, "Invalid verbosity level", "level", v)
		return err
	} else if logLevel < 0 || logLevel > 5 {
		klog.Fatalln("Log level must be between 0 and 5", "level", v)
	}

	klog.SetLogger(newLogger(klog.Level(logLevel), logEncoder))

	return nil
}

func newLogger(logLevel klog.Level, logEncoder string) logr.Logger {
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
