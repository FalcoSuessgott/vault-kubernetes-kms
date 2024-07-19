package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewStandardLogger creates a new zap.Logger based on common configuration
// https://github.com/kubernetes-sigs/aws-encryption-provider/blob/master/pkg/logging/zap.go
// nolint: mnd
func NewStandardLogger(logLevel zapcore.Level) (*zap.Logger, error) {
	return zap.Config{
		Level:       zap.NewAtomicLevelAt(logLevel),
		Development: false,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding: "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "timestamp",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}.Build()
}
