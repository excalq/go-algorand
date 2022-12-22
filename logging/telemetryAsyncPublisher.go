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
	"fmt"
	"time"

	"github.com/algorand/go-algorand/util/metrics"
)

var telemetryDrops = metrics.MakeCounter(metrics.MetricName{Name: "algod_telemetry_drops_total", Description: "telemetry messages not sent to server"})

// The outermost of three "Hook" layers. This runs event queueing and sending in a goroutine.
// wrappedHook: 
// channelDepth:
// maxQueueDepth:
func asyncTelemetryPublisher(teleDec *telemetryDecorator, channelDepth uint, maxQueueDepth int) *asyncTelemetryHook {
	
	td := teleDec
	ready := td.shipper != nil

	processor := &asyncTelemetryHook{
		teleDecorator: teleDec,
		entries:       make(chan *telEntry, channelDepth),
		quit:          make(chan struct{}),
		maxQueueDepth: maxQueueDepth,
		ready:         ready,
		urlUpdate:     make(chan bool),
	}

	go asyncTelemetryProcessorLoop(processor)
    
	return processor
}

// Telemetry Publishing Goroutine. Exists for lifetime of service.
func asyncTelemetryProcessorLoop(processor *asyncTelemetryHook) {
	defer telemetryAsyncShutdown(processor)

	exit := false
	for !exit {
		exit = !processor.waitForEventAndReady()

		hasEvents := true
		for hasEvents {
			select {
			case entry := <-processor.entries:
				 processor.appendEntry(entry)
			default:
				// Send entries queued in hook.pending
				processor.Lock()
				var entry *telEntry
				if len(processor.pending) > 0 && processor.ready {
					entry = processor.pending[0]
					processor.pending = processor.pending[1:]
				}
				processor.Unlock()
				if entry != nil {
					entry, err := processor.teleDecorator.Enrich(entry)
					if err != nil {
						Base().Warnf("Unable to decorate entry %#v : %v", entry, err)
					}
					// TODO: Refactor Ship() up to asyncTelemetryHook. Decorator ain't got time for that. Also nil ptr chk?.
					err = processor.teleDecorator.shipper.Publish(*entry)
					if err != nil {
						Base().Warnf("Unable to write entry %#v to telemetry : %v", entry, err)
					}
					processor.wg.Done()
				} else {
					hasEvents = false
				}
			}
		}
	}
}

func telemetryAsyncShutdown(processor *asyncTelemetryHook) {
	moreEntries := true
	for moreEntries {
		select {
		case entry := <-processor.entries:
			processor.appendEntry(entry)
		default:
			moreEntries = false
		}
	}
	for range processor.pending {
		// The telemetry service is exiting. 
		// Un-wait to allow flushing of remaining messages.
		processor.wg.Done()
	}
	// ************ TODO: Actually log this! ******************************
	fmt.Println("Telemetry Processor is being shut down. This should only happen on service termination.")
	processor.wg.Done()
}


// appendEntry adds the given entry to the pending slice and returns whether the hook is ready or not.
func (hook *asyncTelemetryHook) appendEntry(entry *telEntry) bool {
	hook.Lock()
	defer hook.Unlock()
	// TODO: If there are errors at startup, before the telemetry URI is set, this can fill up. Should we prioritize
	//       startup / heartbeat events?
	if len(hook.pending) >= hook.maxQueueDepth {
		hook.pending = hook.pending[1:]
		hook.wg.Done()
		telemetryDrops.Inc(nil)
	}
	hook.pending = append(hook.pending, entry)

	// Return ready here to avoid taking the lock again.
	return hook.ready
}

// Blocks until one of: service shutdown, entry (and hook ready), hook URI has been updated (or set by config).
// Return values:
// - false: publish events immediately, we're exiting
// - true: Queue events until hook is ready
func (hook *asyncTelemetryHook) waitForEventAndReady() bool {
	for {
		select {
		case <-hook.quit:
			return false
		case entry := <-hook.entries:
			ready := hook.appendEntry(entry)

			// Otherwise keep waiting for the URL to update.
			if ready {
				return true
			}
		case <-hook.urlUpdate:
			hook.Lock()
			hasEvents := len(hook.pending) > 0
			hook.Unlock()

			// Otherwise keep waiting for an entry.
			if hasEvents {
				return true
			}
		}
	}
}

// This is run directly by the logging library on each log event. It triggers inner Hooks via goroutines
func (hook *asyncTelemetryHook) Run(event *Event, level Level, message string)  {
	// Since event is "write-only", create a usable equivalent for publishing as telemetry
	telEntry := &telEntry{
		time: time.Now(),
		level: level,
		message: message,
		rawLogEvent: event,
	}

	hook.Enqueue(telEntry)
}

// Adds telemetry events to the async processor, via channels
func (hook *asyncTelemetryHook) Enqueue(entry *telEntry) {
	hook.wg.Add(1)
	select {
	case <-hook.quit:
		// telemetry quit
		hook.wg.Done()
	case hook.entries <- entry:
	default:
		hook.wg.Done()
		// queue is full, don't block, drop message.

		// metrics is a different mechanism that will never block
		telemetryDrops.Inc(nil)
	}
}

func (hook *asyncTelemetryHook) Close() {
	hook.wg.Add(1)
	close(hook.quit)
	hook.wg.Wait()
}

func (hook *asyncTelemetryHook) Flush() {
	hook.wg.Wait()
}

// This informs the async processor that the URI has been updated, unblocking publishing
func (hook *asyncTelemetryHook) NotifyURIUpdated(uri string) (err error) {
	hook.urlUpdate <- true
	return
}
