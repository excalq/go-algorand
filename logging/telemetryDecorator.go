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

// EnrichedHook: Decorates the Shipping hook with metadata fields such as version and log context
// Filters out stack trace messages

package logging

import (
	"errors"
	"fmt"
	"strings"
)

type telemetryDecorator struct {
	telemetryConfig TelemetryConfig
	shipper 		*telemetryShipper
	reportLogLevel  Level
	history         *logBuffer
	sessionGUID     string
	factory         shipperFactory
}

// createTelemetryDecorator creates the Telemetry hook decorator, or returns nil if remote logging is not enabled
func createTelemetryDecorator(cfg TelemetryConfig, history *logBuffer, shipperFactory shipperFactory) (teleDec *telemetryDecorator, err error) {
	if !cfg.Enable {
		return nil, fmt.Errorf("createTelemetryHook called when telemetry not enabled")
	}

	shipper, err := shipperFactory(cfg)

	if err != nil {
		return nil, err
	}

	tDec, err := newTelemetryDecorator(cfg, shipper, cfg.ReportHistoryLevel, history, cfg.SessionGUID, shipperFactory)

	return tDec, err
}

// newTelemetryDecorator creates a hook filter for ensuring telemetry events are
// always included by the wrapped log hook.
func newTelemetryDecorator(cfg TelemetryConfig, shipper *telemetryShipper, reportLogLevel Level, history *logBuffer, sessionGUID string, factory shipperFactory) (*telemetryDecorator, error) {
	tDec := &telemetryDecorator{
		cfg,
		shipper,
		reportLogLevel,
		history,
		sessionGUID,
		factory,
	}
	return tDec, nil
}

// Enriches telemetry events with recent log context, and service metadata
func (td *telemetryDecorator) Enrich(entry *Entry, level Level, message string) (err error) {

	// NOTE(@excalq): Zerolog.Hook's interface does not include an error return, so disabling errors. Not awesome. :(
	// Just in case
	if td.shipper == nil {
		return errors.New("the wrapped hook has not been initialized")
	}

	// Don't include log history when logging debug.Stack() - just pass it through.
	if level == Error && strings.HasPrefix(message, stackPrefix) {
		return td.shipper.Publish(entry)
	}

	// @excalq: Zerolog.Event gives no external access to needed fields. Can we do this differently,
	// or is that a dealbreaker for using Zerolog? Could history telemetry happen in an 
	// external goroutine, rather than in hooks?
	// See https://github.com/rs/zerolog/pull/395

	if level <= td.reportLogLevel {
		// Logging entry at a level which should include log history
		// Create a new entry augmented with the history field.
		newEntry := entry.Fields(Fields{
			"log": td.history.string(), 
			"session": td.sessionGUID, 
			"v": td.telemetryConfig.Version,
		})
		newEntry.Time = entry.Time
		newEntry.Level = entry.Level
		newEntry.Message = entry.Message

		td.history.trim() // trim history log so we don't keep sending a lot of redundant logs

		return td.shipper.Publish(newEntry)
	}

	// If we're not including log history and session GUID, create a new
	// entry that includes the session GUID, unless it is already present
	// (which it will be for regular telemetry events)
	var newEntry *Entry
	if _, has := entry.Data["session"]; has {
		newEntry = entry
	} else {
		newEntry = entry.WithField("session", td.sessionGUID)
	}

	// Also add version field, if not already present.
	if _, has := entry.Data["v"]; !has {
		newEntry = newEntry.WithField("v", td.telemetryConfig.Version)
	}
	return td.shipper.Publish(newEntry)
}

