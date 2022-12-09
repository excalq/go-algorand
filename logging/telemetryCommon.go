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
	"sync"
	"time"

	"github.com/algorand/go-deadlock"
	"github.com/rs/zerolog"

	"github.com/algorand/go-algorand/logging/telemetryspec"
)

// TelemetryOperation wraps the context for an ongoing telemetry.StartOperation call
type TelemetryOperation struct {
	startTime      time.Time
	category       telemetryspec.Category
	identifier     telemetryspec.Operation
	telemetryState *telemetryState
	pending        int32
}

type telemetryHook interface {
	// Run(entry *zerolog.Event) error
	Run(e *zerolog.Event, level Level, message string)
	Close()
	Flush()
	UpdateHookURI(uri string) (err error)

	appendEntry(entry *zerolog.Event) bool
	waitForEventAndReady() bool
}

type telemetryState struct {
	history         *logBuffer
	hook            telemetryHook
	telemetryConfig TelemetryConfig
}

// TelemetryConfig represents the configuration of Telemetry logging
type TelemetryConfig struct {
	Enable             bool      // Enable remote telemetry
	SendToLog          bool      // Include telemetry events in local log
	URI                string
	Name               string
	GUID               string
	// !!! WARNING !!!
	// TODO(2022-12-01): Refactoring to zerolog/Zap would have a breaking change. Leveling in Logrus (Panic==0) is reverse of Zerolog & Zap (Debug==0)
	MinLogLevel        zerolog.Level `json:"-"` // these are the logrus.Level, but we can't use it directly since on logrus version 1.4.2 they added
	ReportHistoryLevel zerolog.Level `json:"-"` // text marshalers which breaks our backward compatibility.
	FilePath           string       // Path to file on disk, if any
	ChainID            string       `json:"-"`
	SessionGUID        string       `json:"-"`
	Version            string       `json:"-"`
	UserName           string
	Password           string
}

// MarshalingTelemetryConfig is used for json serialization of the TelemetryConfig
// so that we could replace the MinLogLevel/ReportHistoryLevel with our own types.
type MarshalingTelemetryConfig struct {
	TelemetryConfig

	MinLogLevel        uint32
	ReportHistoryLevel uint32
}

type asyncTelemetryHook struct {
	deadlock.Mutex
	wrappedHook   zerolog.Hook
	wg            sync.WaitGroup
	pending       []*zerolog.Event
	entries       chan *zerolog.Event
	quit          chan struct{}
	maxQueueDepth int
	ready         bool
	urlUpdate     chan bool
}

// A dummy noop type to get rid of checks like telemetry.hook != nil
type dummyHook struct{}

type hookFactory func(cfg TelemetryConfig) (Hook, error)
