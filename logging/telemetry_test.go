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
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/algorand/go-deadlock"

	"github.com/algorand/go-algorand/config"
	"github.com/algorand/go-algorand/logging/telemetryspec"
	"github.com/algorand/go-algorand/test/partitiontest"
)

type mockTelemetryHook struct {
	mu       *deadlock.Mutex
	// levels   []Level
	_entries []string
	_data    []Fields
	cb       func(entry *zerolog.Event)
}

func makeMockTelemetryHook(level Level) mockTelemetryHook {
	// levels := make([]logging.Level, 0)
	// for _, l := range []logging.Level{
	// 	logging.PanicLevel,
	// 	logging.FatalLevel,
	// 	logging.ErrorLevel,
	// 	logging.WarnLevel,
	// 	logging.InfoLevel,
	// 	logging.DebugLevel,
	// } {
	// 	if l <= level {
	// 		levels = append(levels, l)
	// 	}
	// }
	h := mockTelemetryHook{
		// levels: levels,
		mu:     &deadlock.Mutex{},
	}
	return h
}

type telemetryTestFixture struct {
	hook  mockTelemetryHook
	telem *telemetryState
	l     Logger
}

func makeTelemetryTestFixture(minLevel Level) *telemetryTestFixture {
	return makeTelemetryTestFixtureWithConfig(minLevel, nil)
}

func makeTelemetryTestFixtureWithConfig(minLevel Level, cfg *TelemetryConfig) *telemetryTestFixture {
	f := &telemetryTestFixture{}
	var lcfg TelemetryConfig
	if cfg == nil {
		lcfg = createTelemetryConfig()
	} else {
		lcfg = *cfg
	}
	lcfg.Enable = true
	lcfg.MinLogLevel = minLevel
	f.hook = makeMockTelemetryHook(minLevel)
	f.l = Base().(Logger)
	f.l.SetLevel(Debug) // Ensure logging doesn't filter anything out

	// f.telem, _ = makeTelemetryState(lcfg, func(cfg TelemetryConfig) (hook Hook, err error) {
	// 	return &f.hook, nil
	// })
	f.telem = f.telem
	return f
}

func (f *telemetryTestFixture) Flush() {
	f.telem.hook.Flush()
}

func (f *telemetryTestFixture) hookData() []Fields {
	f.Flush()
	return f.hook.data()
}

func (f *telemetryTestFixture) hookEntries() []string {
	f.Flush()
	return f.hook.entries()
}

func (h *mockTelemetryHook) Run(entry *zerolog.Event, level zerolog.Level, message string) {
	// h.mu.Lock()
	// defer h.mu.Unlock()
	// h._entries = append(h._entries, entry.Message)
	// h._data = append(h._data, entry.Data)
	// if h.cb != nil {
	// 	h.cb(entry)
	// }
	// return
}

func (h *mockTelemetryHook) data() []Fields {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h._data
}

func (h *mockTelemetryHook) entries() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h._entries
}

func TestCreateHookError(t *testing.T) {
	partitiontest.PartitionTest(t)
	// a := require.New(t)

	cfg := createTelemetryConfig()
	cfg.Enable = true
	// telem, err := makeTelemetryState(cfg, func(cfg TelemetryConfig) (hook Hook, err error) {
	// 	return nil, fmt.Errorf("failed")
	// })

	// a.Nil(telem)
	// a.NotNil(err)
	// a.Equal(err.Error(), "failed")
}

func TestTelemetryHook(t *testing.T) {
	// partitiontest.PartitionTest(t)
	// a := require.New(t)
	// f := makeTelemetryTestFixture(logging.InfoLevel)

	// a.NotNil(f.telem)
	// a.Zero(len(f.hookEntries()))

	// f.telem.logMetrics(f.l, testString1, testMetrics{}, nil)
	// f.telem.logEvent(f.l, testString1, testString2, nil)
	// op := f.telem.logStartOperation(f.l, testString1, testString2)
	// time.Sleep(1 * time.Millisecond)
	// op.Stop(f.l, nil)

	// entries := f.hookEntries()
	// a.Equal(4, len(entries))
	// a.Equal(buildMessage(testString1, testString2), entries[0])
	// a.Equal(buildMessage(testString1, testString2), entries[1])
	// a.Equal(buildMessage(testString1, testString2, "Start"), entries[2])
	// a.Equal(buildMessage(testString1, testString2, "Stop"), entries[3])
	// a.NotZero(f.hookData()[3]["duration"])
}

func TestNilMetrics(t *testing.T) {
	// partitiontest.PartitionTest(t)
	// a := require.New(t)
	// f := makeTelemetryTestFixture(logging.InfoLevel)

	// f.telem.logMetrics(f.l, testString1, nil, nil)

	// a.Zero(len(f.hookEntries()))
}

func TestMultipleOperationStop(t *testing.T) {
	// partitiontest.PartitionTest(t)
	// a := require.New(t)
	// f := makeTelemetryTestFixture(logging.InfoLevel)

	// op := f.telem.logStartOperation(f.l, testString1, testString2)
	// op.Stop(f.l, nil)

	// // Start and stop should result in 2 entries
	// a.Equal(2, len(f.hookEntries()))

	// op.Stop(f.l, nil)

	// // Calling stop again should not result in another entry
	// a.Equal(2, len(f.hookEntries()))
}

func TestDetails(t *testing.T) {
	// partitiontest.PartitionTest(t)
	// a := require.New(t)
	// f := makeTelemetryTestFixture(logging.InfoLevel)

	// details := testMetrics{
	// 	val: "value",
	// }
	// f.telem.logEvent(f.l, testString1, testString2, details)

	// data := f.hookData()
	// a.NotNil(data)
	// a.Equal(details, data[0]["details"])
}

func TestHeartbeatDetails(t *testing.T) {
	// partitiontest.PartitionTest(t)
	// a := require.New(t)
	// f := makeTelemetryTestFixture(logging.InfoLevel)

	// var hb telemetryspec.HeartbeatEventDetails
	// hb.Info.Version = "v2"
	// hb.Info.VersionNum = "1234"
	// hb.Info.Channel = "alpha"
	// hb.Info.Branch = "br0"
	// hb.Info.CommitHash = "abcd"
	// hb.Metrics = map[string]float64{
	// 	"Hello": 38.8,
	// }
	// f.telem.logEvent(f.l, telemetryspec.ApplicationState, telemetryspec.HeartbeatEvent, hb)

	// data := f.hookData()
	// a.NotNil(data)
	// a.Len(data, 1)
	// a.Equal(hb, data[0]["details"])

	// // assert JSON serialization is backwards compatible
	// js, err := json.Marshal(data[0])
	// a.NoError(err)
	// var unjs map[string]interface{}
	// a.NoError(json.Unmarshal(js, &unjs))
	// a.Contains(unjs, "details")
	// ev := unjs["details"].(map[string]interface{})
	// Metrics := ev["Metrics"].(map[string]interface{})
	// m := ev["m"].(map[string]interface{})
	// a.Equal("v2", Metrics["version"].(string))
	// a.Equal("1234", Metrics["version-num"].(string))
	// a.Equal("alpha", Metrics["channel"].(string))
	// a.Equal("br0", Metrics["branch"].(string))
	// a.Equal("abcd", Metrics["commit-hash"].(string))
	// a.InDelta(38.8, m["Hello"].(float64), 0.01)
}

type testMetrics struct {
	val string
}

func (m testMetrics) Identifier() telemetryspec.Metric {
	return testString2
}

func TestMetrics(t *testing.T) {
	// partitiontest.PartitionTest(t)
	// a := require.New(t)
	// f := makeTelemetryTestFixture(logging.InfoLevel)

	// metrics := testMetrics{
	// 	val: "value",
	// }

	// f.telem.logMetrics(f.l, testString1, metrics, nil)

	// data := f.hookData()
	// a.NotNil(data)
	// a.Equal(metrics, data[0]["metrics"])
}

func TestLogHook(t *testing.T) {
	// partitiontest.PartitionTest(t)
	// a := require.New(t)
	// f := makeTelemetryTestFixture(logging.InfoLevel)

	// // Wire up our telemetry hook directly
	// enableTelemetryState(f.telem, &f.l)
	// a.True(f.l.GetTelemetryEnabled())

	// // When we enable telemetry, we no longer send an event.
	// a.Equal(0, len(f.hookEntries()))

	// f.l.Warn("some error")

	// // Now that we're hooked, we should see the log entry in telemetry too
	// a.Equal(1, len(f.hookEntries()))
}

func TestLogLevels(t *testing.T) {
	// partitiontest.PartitionTest(t)
	// runLogLevelsTest(t, logging.DebugLevel, 7)
	// runLogLevelsTest(t, logging.InfoLevel, 6)
	// runLogLevelsTest(t, logging.WarnLevel, 5)
	// runLogLevelsTest(t, logging.ErrorLevel, 4)
	// runLogLevelsTest(t, logging.FatalLevel, 1)
	// runLogLevelsTest(t, logging.PanicLevel, 1)
}

func runLogLevelsTest(t *testing.T, minLevel Level, expected int) {
	// a := require.New(t)
	// f := makeTelemetryTestFixture(minLevel)
	// enableTelemetryState(f.telem, &f.l)

	// f.l.Debug("debug")
	// f.l.Info("info")
	// f.l.Warn("warn")
	// f.l.Error("error")
	// // f.l.Fatal("fatal") - can't call this - it will os.Exit()

	// // Protect the call to log.Panic as we don't really want to crash
	// func() {
	// 	defer func() {
	// 		if r := recover(); r != nil {
	// 		}
	// 	}()
	// 	f.l.Panic("panic")
	// }()

	// // See if we got the expected number of entries
	// a.Equal(expected, len(f.hookEntries()))
}

func TestLogHistoryLevels(t *testing.T) {
	// partitiontest.PartitionTest(t)
	// a := require.New(t)
	// cfg := createTelemetryConfig()
	// cfg.MinLogLevel = logging.DebugLevel
	// cfg.ReportHistoryLevel = logging.ErrorLevel

	// f := makeTelemetryTestFixtureWithConfig(logging.DebugLevel, &cfg)
	// enableTelemetryState(f.telem, &f.l)

	// f.l.Debug("debug")
	// f.l.Info("info")
	// f.l.Warn("warn")
	// f.l.Error("error")
	// // f.l.Fatal("fatal") - can't call this - it will os.Exit()
	// // Protect the call to log.Panic as we don't really want to crash
	// func() {
	// 	defer func() {
	// 		if r := recover(); r != nil {
	// 		}
	// 	}()
	// 	f.l.Panic("panic")
	// }()

	// data := f.hookData()
	// a.Nil(data[0]["log"]) // Debug
	// a.Nil(data[1]["log"]) // Info
	// a.Nil(data[2]["log"]) // Warn

	// // Starting with Error level, we include log history.
	// // Error also emits a debug.stack() log error, so each Error/Panic also create
	// // a log entry.
	// // We do not include log history with stack trace events as they're redundant

	// a.Nil(data[3]["log"])    // Error - we start including log history (this is stack trace)
	// a.NotNil(data[4]["log"]) // Error
	// a.Nil(data[5]["log"])    // Panic - this is stack trace
	// a.NotNil(data[6]["log"]) // Panic
}

func TestReadTelemetryConfigOrDefaultNoDataDir(t *testing.T) {
	partitiontest.PartitionTest(t)
	a := require.New(t)
	tempDir := os.TempDir()
	originalGlobalConfigFileRoot, _ := config.GetGlobalConfigFileRoot()
	config.SetGlobalConfigFileRoot(tempDir)

	cfg, err := ReadTelemetryConfigOrDefault("", "")
	defaultCfgSettings := createTelemetryConfig()
	config.SetGlobalConfigFileRoot(originalGlobalConfigFileRoot)

	a.Nil(err)
	a.NotNil(cfg)
	a.NotEqual(TelemetryConfig{}, cfg)
	a.Equal(defaultCfgSettings.UserName, cfg.UserName)
	a.Equal(defaultCfgSettings.Password, cfg.Password)
	a.Equal(len(defaultCfgSettings.GUID), len(cfg.GUID))
}
