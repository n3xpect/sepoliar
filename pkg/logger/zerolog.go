package logger

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

type zerologLogger struct {
	logger zerolog.Logger
}

type levelAwareWriter struct {
	stdout io.Writer
	stderr io.Writer
}

func (w *levelAwareWriter) WriteLevel(l zerolog.Level, p []byte) (int, error) {
	if l >= zerolog.ErrorLevel {
		return w.stderr.Write(p)
	}
	return w.stdout.Write(p)
}
func (w *levelAwareWriter) Write(p []byte) (int, error) {
	return w.stdout.Write(p)
}
func newLevelAwareConsoleWriter() zerolog.ConsoleWriter {
	return zerolog.ConsoleWriter{
		Out:        &levelAwareWriter{stdout: os.Stdout, stderr: os.Stderr},
		TimeFormat: time.RFC3339,
	}
}

func New() Logger {
	zl := zerolog.New(newLevelAwareConsoleWriter()).With().Timestamp().Logger()
	return &zerologLogger{logger: zl}
}

func NewLog(level string) Logger {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}

	zl := zerolog.New(newLevelAwareConsoleWriter()).Level(lvl).With().Timestamp().Logger()
	return &zerologLogger{logger: zl}
}

func NewJSON() Logger {
	writer := &levelAwareWriter{stdout: os.Stdout, stderr: os.Stderr}
	zl := zerolog.New(writer).With().Timestamp().Logger()
	return &zerologLogger{logger: zl}
}

func (l *zerologLogger) Trace(ctx context.Context, msg string, fields ...Field) {
	event := l.logger.Trace()
	l.addFields(event, fields...)
	l.addContextFields(ctx, event)
	event.Msg(msg)
}
func (l *zerologLogger) Debug(ctx context.Context, msg string, fields ...Field) {
	event := l.logger.Debug()
	l.addFields(event, fields...)
	l.addContextFields(ctx, event)
	event.Msg(msg)
}
func (l *zerologLogger) Info(ctx context.Context, msg string, fields ...Field) {
	event := l.logger.Info()
	l.addFields(event, fields...)
	l.addContextFields(ctx, event)
	event.Msg(msg)
}
func (l *zerologLogger) Warn(ctx context.Context, msg string, fields ...Field) {
	event := l.logger.Warn()
	l.addFields(event, fields...)
	l.addContextFields(ctx, event)
	event.Msg(msg)
}
func (l *zerologLogger) Error(ctx context.Context, msg string, fields ...Field) {
	event := l.logger.Error()
	l.addFields(event, fields...)
	l.addContextFields(ctx, event)
	event.Msg(msg)
}
func (l *zerologLogger) Fatal(ctx context.Context, msg string, fields ...Field) {
	event := l.logger.Fatal()
	l.addFields(event, fields...)
	l.addContextFields(ctx, event)
	event.Msg(msg)
}
func (l *zerologLogger) Panic(ctx context.Context, msg string, fields ...Field) {
	event := l.logger.Panic()
	l.addFields(event, fields...)
	l.addContextFields(ctx, event)
	event.Msg(msg)
}

func (l *zerologLogger) With(fields ...Field) Logger {
	ctx := l.logger.With()
	for _, f := range fields {
		ctx = ctx.Interface(f.Key, f.Value)
	}
	return &zerologLogger{logger: ctx.Logger()}
}

func (l *zerologLogger) addFields(event *zerolog.Event, fields ...Field) {
	for _, f := range fields {
		switch v := f.Value.(type) {
		case string:
			event.Str(f.Key, v)
		case int:
			event.Int(f.Key, v)
		case int64:
			event.Int64(f.Key, v)
		case float64:
			event.Float64(f.Key, v)
		case bool:
			event.Bool(f.Key, v)
		case error:
			event.Err(v)
		case time.Duration:
			event.Dur(f.Key, v)
		case time.Time:
			event.Time(f.Key, v)
		default:
			event.Interface(f.Key, v)
		}
	}
}

func (l *zerologLogger) addContextFields(ctx context.Context, event *zerolog.Event) {
	if ctx == nil {
		return
	}

	if requestID := ctx.Value("request_id"); requestID != nil {
		event.Interface("request_id", requestID)
	}
	if userID := ctx.Value("user_id"); userID != nil {
		event.Interface("user_id", userID)
	}
	if traceID := ctx.Value("trace_id"); traceID != nil {
		event.Interface("trace_id", traceID)
	}
}
