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

type Hook = zerolog.Hook
type Event = zerolog.Event
// Zerolog.Event is write-only, (values are pre-marshalled)
// telEntry allows passing hook metadata through decorators
type telEntry struct {
	time time.Time
	level Level
	message string
	fields Fields
	rawLogEvent *Event // Perf: Avoid use without good reason
}

// AsyncPublisher's Hook interface, used by logging library
type telemetryHook interface {
	// zerolog Hook function
	Run(e *Event, level Level, message string)
	// Direct sending, via Async Publisher
	Enqueue(e *telEntry)
	Close()
	Flush()
	NotifyURIUpdated(uri string) (err error)
	appendEntry(entry *telEntry) bool
	waitForEventAndReady() bool
}

type asyncTelemetryHook struct {
	deadlock.Mutex
	teleDecorator   *telemetryDecorator
	wg            sync.WaitGroup
	pending       []*telEntry
	entries       chan *telEntry
	quit          chan struct{}
	maxQueueDepth int
	ready         bool
	urlUpdate     chan bool
}

// A dummy noop type to get rid of checks like telemetry.hook != nil
type dummyHook struct{}

type telemetryDecorator struct {
	telemetryConfig  TelemetryConfig
	shipper 		 *telemetryShipper
	reportLogLevel   Level
	history          *logBuffer
	sessionGUID      string
	factory          shipperFactory
}
type shipperFactory func(cfg TelemetryConfig) (*telemetryShipper, error)

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
	// Note that levels are translated to/from config, and use zerolog's native numbering here. See logLevels.go
	MinLogLevel        Level `json:"-"` // Custom marshalled for backward compatibility
	ReportHistoryLevel Level `json:"-"` // Custom marshalled
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
