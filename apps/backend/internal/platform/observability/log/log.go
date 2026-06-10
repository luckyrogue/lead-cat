package log

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(level, format string) (*zap.Logger, error) {
	var cfg zap.Config
	if strings.EqualFold(format, "console") {
		cfg = zap.NewDevelopmentConfig()
	} else {
		cfg = zap.NewProductionConfig()
	}
	lvl := zapcore.InfoLevel
	_ = lvl.UnmarshalText([]byte(level))
	cfg.Level = zap.NewAtomicLevelAt(lvl)
	cfg.EncoderConfig.TimeKey = "ts"
	return cfg.Build()
}

func MustNew(level, format string) *zap.Logger {
	l, err := New(level, format)
	if err != nil {
		_, _ = os.Stderr.WriteString("logger: " + err.Error() + "\n")
		os.Exit(1)
	}
	return l
}
