package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Init(level zapcore.Level) {
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(level)
	logger, err := cfg.Build()
	if err != nil {
		return
	}
	zap.ReplaceGlobals(logger)
}
