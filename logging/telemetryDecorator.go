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
	"fmt"
	"strings"
)

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
	teleDec := &telemetryDecorator{
		cfg,
		shipper,
		reportLogLevel,
		history,
		sessionGUID,
		factory,
	}
	return teleDec, nil
}

// Enriches telemetry events with recent log context, and service metadata
func (td *telemetryDecorator) Enrich(entry *telEntry) (decoratedEntry *telEntry, err error) {
	level := entry.level
	message := entry.message

	// Don't include log history when logging debug.Stack() - just pass it through.
	if level == Error && strings.HasPrefix(message, stackPrefix) {
		return entry, nil
	}

	// If the log event which triggered this hook was above the configured
	// reporting level, attach the last `logBufferDepth` lines of lines to telemetry.
	if level <= td.reportLogLevel {
		entry.fields["log"] = td.history.string()
		td.history.trim() // trim history log so we don't keep sending a lot of redundant logs
	}

	// Add certain metadata to the telemetry entry:
	
	// Algod version
	entry.fields["v"] = td.telemetryConfig.Version
	// Host session GUID
	entry.fields["session"] = td.sessionGUID

	return entry, nil
}
