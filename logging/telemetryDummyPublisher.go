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

// A dummy noop type to get rid of checks like telemetry.hook != nil
package logging

func (hook *dummyHook) NotifyURIUpdated(uri string) (err error) {
	return
}

func (hook *dummyHook) Run(event *Event, level Level, message string) {
}

func (hook *dummyHook) Enqueue(entry *telEntry) {
}

func (hook *dummyHook) Close() {}

func (hook *dummyHook) Flush() {}

func (hook *dummyHook) appendEntry(entry *telEntry) bool {
	return true
}

func (hook *dummyHook) waitForEventAndReady() bool {
	return true
}
