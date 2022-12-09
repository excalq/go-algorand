package logging

import (
	"io"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/algorand/go-algorand/logging/telemetryspec"
	"github.com/rs/zerolog"
)

// ==== LoggerConfig Methods ====

// Replaces logger with a new, re-leveled instance
func (l *logFacade) SetLevel(lvl Level) *logFacade {
	l.log = l.log.Level(zerolog.Level(lvl)) // zerolog.Level() returns a new copy
	return l
}

func (l *logFacade) SetOutput(w io.Writer) {
	l.log = l.log.Output(w)
}

// Logs colored, human readable text. Use in development only as it's not perforant.
func (l logFacade) UsePrettyOutput(w io.Writer) {
	l.SetOutput(zerolog.ConsoleWriter{Out: w})
	l.log.Info().Msg("Enabling ConsoleWriter logging output. Warning: This has slower perforance than JSON.")
}

// Deprecated: JSON is the default when any io output is set
func (l logFacade) SetJSONFormatter() {
	l.log.Warn().Str("DEPRECATED", "logger.SetJSONFormatter()").Msg("logger.SetJSONFormatter() as JSON is default in Zerolog.")
}

// Register hooks onto Zerolog Events
func (l logFacade) AddHook(hook telemetryHook) {
	// FIXME(@excalq) l.log.Hook(hook)
	l.log.Info().Msg("New logging hook registered.")
}

// ==== LoggerInfo Methods ====

// Direct access to Zerolog. Lacks context and logging pkg types
func (l logFacade) Log() zerolog.Logger {
	return l.log
}

func (l logFacade) IsLevelEnabled(level Level) bool {
	return Level(l.log.GetLevel()) >= level
}

// Returns the io.writer output used by this logger instance
func (l logFacade) getOutput() io.Writer {
	return l.output
}


// ==== ChainableLogger Methods ====

// Adds single field to structured log
func (l logFacade) With(key string, value interface{}) Logger {
	return &logFacade{
		l.log.With().Interface(key, value).Logger(),
		l.output,
		l.telemetry,
	}
}

// Adds fields from a map of key-values
func (l logFacade) WithFields(fields Fields) Logger {
	return &logFacade{
		l.log.With().Fields(fields).Logger(),
		l.output,
		l.telemetry,
	}
}

// Generate fields for file, line of caller
func (l logFacade) WithCaller() Logger {
	var lctx zerolog.Context

	pc, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "<???>"
		line = 1
		lctx = l.log.With().Fields(Fields{
			"file": file,
			"line": line,
		})
	} else {
		// Add file name and number
		slash := strings.LastIndex(file, "/")
		file = file[slash+1:]
		lctx = l.log.With().Fields(Fields{
			"file": file,
			"line": line,
		})

		// Add function name if possible
		if function := runtime.FuncForPC(pc); function != nil {
			lctx = lctx.Str("function", function.Name())
		}
	}
	return &logFacade{
		lctx.Logger(),
		l.output,
		l.telemetry,
	}
}

// ==== PublishingLogger Methods ===
// TODO: Refactor logging callers to use typed methods, not these catchalls


// Note: These call Msg() which sends the event.
// To chain a .Debug().other() call, use l.log.Debug().other()
func (l logFacade) Debug(args ...interface{}) {
	// Using .source() requires allocation (.Debug() et. al need addressable *logger)
	log := l.WithCaller().Log()
	log.Debug().Caller().Msg(serializeArguments(args...))
}

// Cargo culting these into the age of stuctured logging...
func (l logFacade) Debugln(args ...interface{}) {
	l.Debug(args)
}

func (l logFacade) Debugf(format string, args ...interface{}) {
	log := l.WithCaller().Log()
	log.Debug().Msgf(format, args...)
}

func (l logFacade) Info(args ...interface{}) {
	log := l.WithCaller().Log()
	log.Info().Msg(serializeArguments(args...))
}

func (l logFacade) Infoln(args ...interface{}) {
	l.Info(args)
}

func (l logFacade) Infof(format string, args ...interface{}) {
	log := l.WithCaller().Log()
	log.Info().Msgf(format, args...)
}

func (l logFacade) Warn(args ...interface{}) {
	log := l.WithCaller().Log()
	log.Warn().Msg(serializeArguments(args...))
}

func (l logFacade) Warnln(args ...interface{}) {
	l.Warn(args)
}

func (l logFacade) Warnf(format string, args ...interface{}) {
	log := l.WithCaller().Log()
	log.Warn().Msgf(format, args...)
}

func (l logFacade) Error(args ...interface{}) {
	log := l.WithCaller().Log()
	log.Error().Msg(serializeArguments(args...))
}

func (l logFacade) Errorln(args ...interface{}) {
	l.Error(args)
}

func (l logFacade) Errorf(format string, args ...interface{}) {
	log := l.WithCaller().Log()
	log.Error().Msgf(format, args...)
}

func (l logFacade) Fatal(args ...interface{}) {
	log := l.WithCaller().Log()
	log.Fatal().Msg(serializeArguments(args...))
}

func (l logFacade) Fatalln(args ...interface{}) {
	l.Fatal(args)
}

func (l logFacade) Fatalf(format string, args ...interface{}) {
	log := l.WithCaller().Log()
	log.Fatal().Msgf(format, args...)
}

func (l logFacade) Panic(args ...interface{}) {
	defer func() {
		if r := recover(); r != nil {
			l.FlushTelemetry()
			panic(r)
		}
	}()
	log := l.WithCaller().Log()
	log.Panic().Msg(serializeArguments(args...))
}

func (l logFacade) Panicln(args ...interface{}) {
	log := l.WithCaller().Log()
	log.Panic().Msg(serializeArguments(args...))
}

func (l logFacade) Panicf(format string, args ...interface{}) {
	defer func() {
		if r := recover(); r != nil {
			l.FlushTelemetry()
			panic(r)
		}
	}()
	// Preceed with stack trace at error level
	l.Error(stackPrefix, string(debug.Stack()))
	log := l.WithCaller().Log()
	log.Panic().Msgf(format, args...)
}

// === Telemetry Methods ====

// Telemetry Methods (@TODO Move to own location)
// (Refactor into Telemetry Facade?)

// Enables telemetry: local logged and remote (if cfg.Enable)
func (l logFacade) EnableTelemetry(cfg TelemetryConfig) (err error) {
	if l.telemetry != nil || (!cfg.Enable && !cfg.SendToLog) {
		return nil
	}
	return EnableTelemetry(cfg, &l)
}

func (l logFacade) UpdateTelemetryURI(uri string) (err error) {
	if l.telemetry.hook != nil {
		err = l.telemetry.hook.UpdateHookURI(uri)
		if err == nil {
			l.telemetry.telemetryConfig.URI = uri
		}
	}
	return err
}

// GetTelemetryEnabled returns true when any of:
// - TelemetryConfig.SendToLog is true
// - config.json TelemetryToLog is true
// - algod/algoh `-t` flag is true or 1
// Note `remoteTelemetryEnabled` may be still false
func (l logFacade) GetTelemetryEnabled() bool {
	return l.telemetry != nil
}

// GetTelemetryUploadingEnabled returns true 
// if remote telemetry uploading logging is enabled.
// This is set logging.config's Enable parameter
func (l logFacade) GetTelemetryUploadingEnabled() bool {
	return l.GetTelemetryEnabled() &&
		l.telemetry.telemetryConfig.Enable
}

func (l logFacade) Metrics(category telemetryspec.Category, metrics telemetryspec.MetricDetails, details interface{}) {
	if l.telemetry != nil {
		l.telemetry.logMetrics(l, category, metrics, details)
	}
}

func (l logFacade) Event(category telemetryspec.Category, identifier telemetryspec.Event) {
	l.EventWithDetails(category, identifier, nil)
}

func (l logFacade) EventWithDetails(category telemetryspec.Category, identifier telemetryspec.Event, details interface{}) {
	if l.telemetry != nil {
		l.telemetry.logEvent(l, category, identifier, details)
	}
}

func (l logFacade) StartOperation(category telemetryspec.Category, identifier telemetryspec.Operation) TelemetryOperation {
	if l.telemetry != nil {
		return l.telemetry.logStartOperation(l, category, identifier)
	}
	return TelemetryOperation{}
}

func (l logFacade) GetTelemetrySession() string {
	if !l.GetTelemetryEnabled() {
		return ""
	}
	return l.telemetry.telemetryConfig.SessionGUID
}

func (l logFacade) GetTelemetryVersion() string {
	if !l.GetTelemetryEnabled() {
		return ""
	}
	return l.telemetry.telemetryConfig.Version
}

func (l logFacade) GetTelemetryGUID() string {
	if !l.GetTelemetryEnabled() {
		return ""
	}
	return l.telemetry.telemetryConfig.getHostGUID()
}

func (l logFacade) GetInstanceName() string {
	if !l.GetTelemetryEnabled() {
		return ""
	}
	return l.telemetry.telemetryConfig.getInstanceName()
}

func (l logFacade) GetTelemetryURI() string {
	if !l.GetTelemetryEnabled() {
		return ""
	}
	return l.telemetry.telemetryConfig.URI
}

func (l logFacade) CloseTelemetry() {
	if l.telemetry != nil {
		l.telemetry.Close()
	}
}

func (l logFacade) FlushTelemetry() {
	if l.telemetry != nil {
		l.telemetry.Flush()
	}
}

// ==== Helper / Migration methods ====

// Turns all varargs into a sprintf string
func serializeArguments(args ...interface{}) string {
	var str string
	for _, s := range args {
		s, _ := s.(string)
		str += s
	}
	return str
}