package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/rs/zerolog"
)

type zerologLogger struct {
	logger zerolog.Logger
}

// prettyFileWriter re-indents each JSON log line before writing it so that
// clean-logs/app.log stays human-readable for local debugging, while
// logs/app.log (written separately, undecorated) keeps the compact,
// machine-parseable original -- satisfying the dual-write requirement.
type prettyFileWriter struct {
	file *os.File
}

func (w *prettyFileWriter) Write(p []byte) (int, error) {
	var raw json.RawMessage
	if err := json.Unmarshal(p, &raw); err != nil {
		// Defensive fallback: should not happen since zerolog always
		// emits JSON, but never silently drop a log line.
		if _, werr := w.file.Write(p); werr != nil {
			return 0, werr
		}
		return len(p), nil
	}

	indented, err := json.MarshalIndent(&raw, "", "  ")
	if err != nil {
		if _, werr := w.file.Write(p); werr != nil {
			return 0, werr
		}
		return len(p), nil
	}

	if _, err := w.file.Write(indented); err != nil {
		return 0, err
	}
	if _, err := w.file.Write([]byte("\n")); err != nil {
		return 0, err
	}
	return len(p), nil
}

// New builds a Logger that dual-writes structured JSON logs to
// <logsDir>/app.log (raw, compact) and <cleanLogsDir>/app.log (indented),
// plus a console stream. When consolePretty is true the console stream is
// colorized/human-formatted (nice for `go run` during local development);
// otherwise it emits the same compact JSON as the file writer (nicer for
// container log aggregation).
func New(logsDir, cleanLogsDir string, consolePretty bool) (Logger, error) {
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cleanLogsDir, 0o755); err != nil {
		return nil, err
	}

	timeStamp := time.Now().Format("2006-01-02_15-04-05")

	rawFile, err := os.OpenFile(filepath.Join(logsDir, fmt.Sprintf("%s-%s.log", timeStamp, uuid.New().String())), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	cleanFile, err := os.OpenFile(filepath.Join(cleanLogsDir, fmt.Sprintf("%s-%s-clean.log", timeStamp, uuid.New().String())), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}

	writers := []io.Writer{rawFile, &prettyFileWriter{file: cleanFile}}
	if consolePretty {
		writers = append(writers, zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	} else {
		writers = append(writers, os.Stdout)
	}

	multi := zerolog.MultiLevelWriter(writers...)
	zl := zerolog.New(multi).With().Timestamp().Logger()

	return &zerologLogger{logger: zl}, nil
}

func applyFields(event *zerolog.Event, fields []Field) *zerolog.Event {
	for _, f := range fields {
		event = event.Interface(f.Key, f.Value)
	}
	return event
}

func (l *zerologLogger) Debug(msg string, fields ...Field) {
	applyFields(l.logger.Debug(), fields).Msg(msg)
}

func (l *zerologLogger) Info(msg string, fields ...Field) {
	applyFields(l.logger.Info(), fields).Msg(msg)
}

func (l *zerologLogger) Warn(msg string, fields ...Field) {
	applyFields(l.logger.Warn(), fields).Msg(msg)
}

func (l *zerologLogger) Error(err error, msg string, fields ...Field) {
	applyFields(l.logger.Error().Err(err), fields).Msg(msg)
}

func (l *zerologLogger) Fatal(err error, msg string, fields ...Field) {
	applyFields(l.logger.Fatal().Err(err), fields).Msg(msg)
}

func (l *zerologLogger) With(fields ...Field) Logger {
	ctx := l.logger.With()
	for _, f := range fields {
		ctx = ctx.Interface(f.Key, f.Value)
	}
	return &zerologLogger{logger: ctx.Logger()}
}
