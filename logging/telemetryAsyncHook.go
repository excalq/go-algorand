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

	"github.com/algorand/go-algorand/util/metrics"
)

var telemetryDrops = metrics.MakeCounter(metrics.MetricName{Name: "algod_telemetry_drops_total", Description: "telemetry messages not sent to server"})

// The outermost of three "Hook" layers. This runs event queueing and sending in a goroutine.
// wrappedHook: 
// channelDepth:
// maxQueueDepth:
func asyncTelemetryPublisher(teleDec telemetryDecorator, channelDepth uint, maxQueueDepth int) *asyncTelemetryHook {
	
	td, ok := teleDec.(*telemetryDecorator)
	ready := ok && td.shipper != nil

	hook := &asyncTelemetryHook{
		teleDecorator: teleDec,
		entries:       make(chan *Entry, channelDepth),
		quit:          make(chan struct{}),
		maxQueueDepth: maxQueueDepth,
		ready:         ready,
		urlUpdate:     make(chan bool),
	}

	// Telemetry Publishing Goroutine. Exists for lifetime of service.
	go func() {
		defer telemetryAsyncShutdown(hook)

		exit := false
		for !exit {
			exit = !hook.waitForEventAndReady()

			hasEvents := true
			for hasEvents {
				select {
				case entry := <-hook.entries:
					hook.appendEntry(entry)
				default:
					// Send entries queued in hook.pending
					hook.Lock()
					var entry *Entry
					if len(hook.pending) > 0 && hook.ready {
						entry = hook.pending[0]
						hook.pending = hook.pending[1:]
					}
					hook.Unlock()
					if entry != nil {
						// Where should level, message come from?
						err := hook.teleDecorator.Enrich(entry, level, message)
						if err != nil {
							Base().Warnf("Unable to write event %#v to telemetry : %v", entry, err)
						}
						hook.wg.Done()
					} else {
						hasEvents = false
					}
				}
			}
		}
	}()
    
	return hook
}

func telemetryAsyncLoop(hook Hook)

func telemetryAsyncShutdown(hook *asyncTelemetryHook) {
	moreEntries := true
	for moreEntries {
		select {
		case entry := <-hook.entries:
			hook.appendEntry(entry)
		default:
			moreEntries = false
		}
	}
	for range hook.pending {
		// The telemetry service is exiting. 
		// Un-wait to allow flushing of remaining messages.
		hook.wg.Done()
	}
	hook.wg.Done()
}


// appendEntry adds the given entry to the pending slice and returns whether the hook is ready or not.
func (hook *asyncTelemetryHook) appendEntry(entry *Entry) bool {
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
func (hook *asyncTelemetryHook) Run(entry *Entry, level Level, message string)  {
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

// Note: This will be removed with the externalized telemetry project. Return whether or not the URI was successfully updated.
func (hook *asyncTelemetryHook) UpdateHookURI(uri string) (err error) {
	
	updated := false
	if hook.teleDecorator == nil {
		return fmt.Errorf("asyncTelemetryHook.wrappedHook is nil")
	}

	tfh, ok := hook.teleDecorator.(*telemetryDecorator)
	if ok {
		hook.Lock()

		copy := tfh.telemetryConfig
		copy.URI = uri
		var newHook Hook
		newHook, err = tfh.factory(copy)

		if err == nil && newHook != nil {
			tfh.wrappedHook = newHook
			tfh.telemetryConfig.URI = uri
			hook.ready = true
			updated = true
		}

		// Need to unlock before sending event to hook.urlUpdate
		hook.Unlock()

		// Notify event listener if the hook was created.
		if updated {
			hook.urlUpdate <- true
		}
	} else {
		return fmt.Errorf("asyncTelemetryHook.wrappedHook does not implement telemetryFilteredHook")
	}
	return
}
