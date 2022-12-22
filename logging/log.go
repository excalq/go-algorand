// Copyright (C) 2019-2022 Algorand, Inc.
// This file is part of go-algorand
//
// go-algorand is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// go-algorand is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with go-algorand.  If not, see <https://www.gnu.org/licenses/>.

package logging

import (
	"io"
	"os"

	"github.com/algorand/go-algorand/logging/telemetryspec"
	"github.com/rs/zerolog"
)

// BaseLogger is a facade to Zerolog
var baseLogger Logger
var defaultOutput io.Writer = zerolog.ConsoleWriter{Out: os.Stderr} // Zerolog, Logrus' default

const stackPrefix = "[Stack]" // For filtering stack traces
const TimeFormatRFC3339Micro = "2006-01-02T15:04:05.999999Z07:00"

// Allows other packages to use zerolog's types (and levels)
type Fields map[string]interface{}

type logFacade struct {
	log  zerolog.Logger
	output io.Writer
	telemetry *telemetryState
}

// @excalq: Removed Init() with sync.once wrap, as that's probably 
// less performant and unneccessary. O_APPEND is already atomic

// Base returns the default Logger
func Base() Logger {
	if baseLogger == nil {
		baseLogger = NewLogger()
		SetLoggerDefaults()
	}
	return baseLogger
}

// Returns a new instace of loggerFacade
func NewLogger() Logger {
	return &logFacade{
		zerolog.New(defaultOutput).With().Timestamp().Logger(),
		defaultOutput,
		&telemetryState{},
	}
}

// Global Zerolog Config

// Set global logger defaults
func SetLoggerDefaults() {
	zerolog.TimeFieldFormat = TimeFormatRFC3339Micro
}

// Affects all instances of Zerolog loggers, process-wide
func SetGlobalLevel(lvl Level) {
	zerolog.SetGlobalLevel(zerolog.Level(lvl))
}

// LogFacade Interfaces

type Logger interface {
	LoggerConfig
	LoggerInfo
	ChainableLogger
	PublishingLogger
	// TODO: refactor further...
	TelemetryManager
}

// These mutate the existing logFacade.log instance
type LoggerConfig interface {
	// Set the logging version (Info by default)
	SetLevel(Level) *logFacade
	
	// Sets the output target
	SetOutput(io.Writer)

	// Human Readable k=v ANSI colored. Less performant
	UsePrettyOutput(io.Writer)

	// Sets the logger to performant JSON Format
	SetJSONFormatter()
	
	// Adds a hook to the logger
	AddHook(hook Hook)
}

// These do not mutate, only return values
type LoggerInfo interface {

	// Direct access to Zerolog. Lacks context and logging pkg types
	Log() zerolog.Logger

	IsLevelEnabled(Level) bool
	
	getOutput() io.Writer

}

// These methods can be chanined to one another
// logFacade.log gets replaced with a new instance
// due to zerolog's returning a new context
type ChainableLogger interface {
	// Add one key-value to log
	With(key string, value interface{}) Logger
	
	// WithFields logs a message with specific fields
	WithFields(Fields) Logger

	// source adds file, line and function fields to the context
	WithCaller() Logger
}

// These methods write log entries and return nothing
// They can only end a chain of Logger methods.
type PublishingLogger interface {

	// Debug logs a message at level Debug.
	Debug(...interface{})
	Debugln(...interface{})
	Debugf(string, ...interface{})

	// Info logs a message at level Info.
	Info(...interface{})
	Infoln(...interface{})
	Infof(string, ...interface{})

	// Warn logs a message at level Warn.
	Warn(...interface{})
	Warnln(...interface{})
	Warnf(string, ...interface{})

	// Error logs a message at level Error.
	Error(...interface{})
	Errorln(...interface{})
	Errorf(string, ...interface{})

	// Fatal logs a message at level Fatal.
	Fatal(...interface{})
	Fatalln(...interface{})
	Fatalf(string, ...interface{})

	// Panic logs a message at level Panic.
	Panic(...interface{})
	Panicln(...interface{})
	Panicf(string, ...interface{})
}

type TelemetryManager interface {
	EnableTelemetry(cfg TelemetryConfig) error
	UpdateTelemetryURI(uri string) error
	GetTelemetryEnabled() bool
	GetTelemetryUploadingEnabled() bool
	Metrics(category telemetryspec.Category, metrics telemetryspec.MetricDetails, details interface{})
	Event(category telemetryspec.Category, identifier telemetryspec.Event)
	EventWithDetails(category telemetryspec.Category, identifier telemetryspec.Event, details interface{})
	StartOperation(category telemetryspec.Category, identifier telemetryspec.Operation) TelemetryOperation
	GetTelemetrySession() string
	GetTelemetryVersion() string
	GetTelemetryGUID() string
	GetInstanceName() string
	GetTelemetryURI() string
	CloseTelemetry()
	FlushTelemetry()
}