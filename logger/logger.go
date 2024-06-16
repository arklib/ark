package logger

import (
	"io"
	"os"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/hertz-contrib/logger/zerolog"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Config struct {
	// Log
	File string `default:"private/logs/app.log"`
	// trace | debug | info | notice | warn | error | fatal
	Level string `default:"error"`
	// 30 day
	MaxAge int `default:"30"`
	// 128 MB
	MaxSize    int `default:"128"`
	MaxBackups int `default:"32"`
}

type Logger struct {
	*zerolog.Logger
	Writer io.Writer
}

func New(c *Config) *Logger {
	logger := zerolog.New(
		zerolog.WithFormattedTimestamp(time.DateTime),
	)

	// configure logger
	lumberjackLogger := lumberjack.Logger{
		Filename:   c.File,
		MaxSize:    c.MaxSize,
		MaxBackups: c.MaxBackups,
		MaxAge:     c.MaxAge,
	}

	// async writer
	logWriter := &zapcore.BufferedWriteSyncer{
		WS:            zapcore.AddSync(&lumberjackLogger),
		FlushInterval: time.Second,
	}

	logger.SetOutput(logWriter)
	logger.SetLevel(GetLogLevel(c.Level))

	return &Logger{Logger: logger, Writer: logWriter}
}

func NewConsole(c *Config) *Logger {
	logger := zerolog.New(
		zerolog.WithFormattedTimestamp(time.Kitchen),
	)

	logWriter := zerolog.ConsoleWriter{
		Out: os.Stdout,
	}

	logger.SetOutput(logWriter)
	logger.SetLevel(GetLogLevel(c.Level))

	return &Logger{Logger: logger, Writer: logWriter}
}

func GetLogLevel(level string) hlog.Level {
	switch level {
	case "trace":
		return hlog.LevelTrace
	case "debug":
		return hlog.LevelDebug
	case "info":
		return hlog.LevelInfo
	case "notice":
		return hlog.LevelNotice
	case "warn":
		return hlog.LevelWarn
	case "error":
		return hlog.LevelError
	case "fatal":
		return hlog.LevelFatal
	default:
		return hlog.LevelInfo
	}
}
