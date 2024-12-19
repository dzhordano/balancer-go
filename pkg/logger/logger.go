package logger

import (
	"fmt"
	"io"
	"log/slog"
)

func NewSlogLogger(out io.Writer, logLevel string) *slog.Logger {
	switch logLevel {
	case "debug":
		return slog.New(slog.NewJSONHandler(out, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case "info":
		return slog.New(slog.NewJSONHandler(out, &slog.HandlerOptions{Level: slog.LevelInfo}))
	case "warn":
		return slog.New(slog.NewJSONHandler(out, &slog.HandlerOptions{Level: slog.LevelWarn}))
	case "error":
		return slog.New(slog.NewJSONHandler(out, &slog.HandlerOptions{Level: slog.LevelError}))
	default:
		fmt.Println("Unknown log level:", logLevel, "defaulting to level Info")
		return slog.New(slog.NewJSONHandler(out, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
}
