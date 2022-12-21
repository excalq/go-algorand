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
	"strings"

	"github.com/algorand/go-algorand/logging/telemetryspec"
	"github.com/algorand/go-algorand/util/uuid"
)

// Telemetry Events are formatted as "/Network/ConnectPeer"
const telemetryPrefix = "/"
const telemetrySeparator = "/"
const logBufferDepth = 2
const channelDepth = 32 // Entries channel to telemetry loop goroutine
const maxQueueDepth = 100 // Size of log history queue

// EnableTelemetry configures and enables telemetry based on the config provided
func EnableTelemetry(cfg TelemetryConfig, l *logFacade) (err error) {
	telemetry, err := makeTelemetryState(cfg, createTelemetryShipper)
	if err != nil {
		return
	};
	enableTelemetryState(telemetry, l)
	return
}

func enableTelemetryState(telemetry *telemetryState, l *logFacade) {
	l.telemetry = telemetry
	// Hook our normal logging to send desired types to telemetry (Events)
	l.AddHook(telemetry.hook)

	// Wrap current logger Output writer to capture history
	// (Tees log events >= cfg.ReportHistoryLevel to Telemetry)
	l.SetOutput(telemetry.wrapOutput(l.getOutput()))
}

// Sets telemetry buffer and outermost hook
func makeTelemetryState(cfg TelemetryConfig, shipperFactory shipperFactory) (*telemetryState, error) {
	telemetry := &telemetryState{}
	telemetry.history = createLogBuffer(logBufferDepth)
	// If telemetry is enabled, set up the remote forwarding
	if cfg.Enable {
		if cfg.SessionGUID == "" {
			cfg.SessionGUID = uuid.New()
		}
		decoratedShipper, err := createTelemetryDecorator(cfg, telemetry.history, shipperFactory)
		if err != nil {
			return nil, err
		}
		// Creates the outermost Hook layer
		telemetry.hook = asyncTelemetryPublisher(decoratedShipper, channelDepth, maxQueueDepth)
	} else {
		telemetry.hook = new(dummyHook)
	}
	telemetry.telemetryConfig = cfg
	return telemetry, nil
}

// ==== TelemetryState Methods ====

// wrapOutput wraps the log writer so we can keep a history of
// the tail of the file to send with critical telemetry events when logged.
func (t *telemetryState) wrapOutput(out io.Writer) io.Writer {
	return t.history.wrapOutput(out)
}

func (t *telemetryState) logMetrics(l logFacade, category telemetryspec.Category, metrics telemetryspec.MetricDetails, details interface{}) {
	if metrics == nil {
		return
	}
	l.log = l.log.With().Fields(Fields{
		"metrics": metrics,
	}).Logger()

	t.logTelemetry(l, buildMessage(string(category), string(metrics.Identifier())), details)
}

func (t *telemetryState) logEvent(l logFacade, category telemetryspec.Category, identifier telemetryspec.Event, details interface{}) {
	t.logTelemetry(l, buildMessage(string(category), string(identifier)), details)
}

func (t *telemetryState) logStartOperation(l logFacade, category telemetryspec.Category, identifier telemetryspec.Operation) TelemetryOperation {
	op := makeTelemetryOperation(t, category, identifier)
	t.logTelemetry(l, buildMessage(string(category), string(identifier), "Start"), nil)
	return op
}

func buildMessage(args ...string) string {
	message := telemetryPrefix + strings.Join(args, telemetrySeparator)
	return message
}

// logTelemetry explicitly only sends telemetry events to the cloud.
func (t *telemetryState) logTelemetry(l logFacade, message string, details interface{}) {
	logFields := Fields{
		"session":      l.GetTelemetrySession(),
		"instanceName": l.GetInstanceName(),
		"v":            l.GetTelemetryVersion(),
	}
	if details != nil {
		logFields["details"] = details
	}

	// @excalq: (perf) Info() can only be called on a pointer
	logger := l.log.With().Fields(logFields).Logger()
	// @excalq: Are all telemetry events sent here at INFO level?
	// entry.Level = logrus.InfoLevel // (pre-refactored) 
	event := logger.Info()

	if t.telemetryConfig.SendToLog {
		l.Info(message)
	}
	if t.hook != nil {
		t.hook.Run(event, Info, message)
	}
}

func (t *telemetryState) Close() {
	if t.hook != nil {
		t.hook.Close()
	}
}

func (t *telemetryState) Flush() {
	t.hook.Flush()
}
