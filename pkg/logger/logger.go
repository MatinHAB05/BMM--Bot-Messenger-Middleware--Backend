// Package logger provides an interface-based wrapper around rs/zerolog.
// The rest of the codebase depends only on the Logger interface defined
// here -- never on zerolog directly -- so the concrete logging backend
// stays swappable and unit tests can supply a no-op/mock implementation.
package logger

import "time"

// Field is a single structured key/value pair attached to a log line.
// Use the constructors below (String, Int, Err, ...) rather than building
// Field literals so call sites read naturally, e.g.:
//
//	log.Info("broadcast dispatched", logger.String("platform", "telegram"), logger.Int("targets", 3))
type Field struct {
	Key   string
	Value interface{}
}

func String(key, value string) Field           { return Field{Key: key, Value: value} }
func Int(key string, value int) Field          { return Field{Key: key, Value: value} }
func Int64(key string, value int64) Field      { return Field{Key: key, Value: value} }
func Uint(key string, value uint) Field        { return Field{Key: key, Value: value} }
func Bool(key string, value bool) Field        { return Field{Key: key, Value: value} }
func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value.String()}
}
func Any(key string, value interface{}) Field { return Field{Key: key, Value: value} }

// Err attaches an error under the conventional "error" key. Prefer passing
// the error to Error()/Fatal() directly; use this when logging a
// *secondary* error alongside the primary one passed to Error().
func Err(err error) Field {
	if err == nil {
		return Field{Key: "error", Value: nil}
	}
	return Field{Key: "error", Value: err.Error()}
}

// Logger is the logging contract every layer of the application (services,
// repositories, middleware, messenger adapters, bot listeners) depends on.
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	// Error logs at error level. err may be nil.
	Error(err error, msg string, fields ...Field)
	// Fatal logs at fatal level and then terminates the process (os.Exit(1)),
	// matching zerolog's own Fatal semantics. Reserve for unrecoverable
	// startup failures.
	Fatal(err error, msg string, fields ...Field)
	// With returns a child Logger that always includes the given fields,
	// useful for attaching a component name or request id once.
	With(fields ...Field) Logger
}
