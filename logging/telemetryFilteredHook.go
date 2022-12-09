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
	"strings"

	"github.com/rs/zerolog"
)

type telemetryFilteredHook struct {
	telemetryConfig TelemetryConfig
	wrappedHook     Hook
	reportLogLevel  Level
	history         *logBuffer
	sessionGUID     string
	factory         hookFactory
	levels          []Level
}

// newFilteredTelemetryHook creates a hook filter for ensuring telemetry events are
// always included by the wrapped log hook.
func newTelemetryFilteredHook(cfg TelemetryConfig, hook Hook, reportLogLevel Level, history *logBuffer, sessionGUID string, factory hookFactory, levels []Level) (Hook, error) {
	filteredHook := &telemetryFilteredHook{
		cfg,
		hook,
		reportLogLevel,
		history,
		sessionGUID,
		factory,
		levels,
	}
	return filteredHook, nil
}

// Run is required to implement zerolog hook interface
func (hook *telemetryFilteredHook) Run(entry *zerolog.Event, level Level, message string) {

	// NOTE(@excalq): Zerolog.Hook's interface does not include an error return, so disabling errors. Not awesome. :(
	// Just in case
	if hook.wrappedHook == nil {
		return
		// return errors.New("the wrapped hook has not been initialized")
	}

	// Don't include log history when logging debug.Stack() - just pass it through.
	if level == Error && strings.HasPrefix(message, stackPrefix) {
		return
		// return hook.wrappedHook.Run(entry, level, message)
	}

	// @excalq: Zerolog.Event gives no external access to needed fields. Can we do this differently,
	// or is that a dealbreaker for using Zerolog? Could history telemetry happen in an 
	// external goroutine, rather than in hooks?
	// See https://github.com/rs/zerolog/pull/395

	// if level <= hook.reportLogLevel {
	// 	// Logging entry at a level which should include log history
	// 	// Create a new entry augmented with the history field.
	// 	newEntry := entry.Fields(Fields{
	// 		"log": hook.history.string(), 
	// 		"session": hook.sessionGUID, 
	// 		"v": hook.telemetryConfig.Version,
	// 	})
	// 	newEntry.Time = entry.Time
	// 	newEntry.Level = entry.Level
	// 	newEntry.Message = entry.Message

	// 	hook.history.trim() // trim history log so we don't keep sending a lot of redundant logs

	// 	return hook.wrappedHook.Run(newEntry)
	// }

	// If we're not including log history and session GUID, create a new
	// entry that includes the session GUID, unless it is already present
	// (which it will be for regular telemetry events)
	// var newEntry *zerolog.Event
	// if _, has := entry.Data["session"]; has {
	// 	newEntry = entry
	// } else {
	// 	newEntry = entry.WithField("session", hook.sessionGUID)
	// }

	// // Also add version field, if not already present.
	// if _, has := entry.Data["v"]; !has {
	// 	newEntry = newEntry.WithField("v", hook.telemetryConfig.Version)
	// }
	// return hook.wrappedHook.Run(newEntry)
}

