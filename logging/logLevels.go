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

// Logging Levels
// Various logging libraries have inconsistent level values. This preserves
// the orignal mapping (based on Logrus), for continuity in configs.
// However, for performance log calls use the active library's level values.
// This creates constants matching the active library (zerolog).

package logging

import (
	"errors"
	"fmt"

	"github.com/rs/zerolog"
)

type Level = zerolog.Level // int8

// Constants must match the active logging library
// for performance (no runtime translation)
const (
	Trace = zerolog.TraceLevel  // -1
	Debug = zerolog.DebugLevel  // 0
	Info = zerolog.InfoLevel    // 1
	Warn = zerolog.WarnLevel    // 2
	Error = zerolog.ErrorLevel  // 3
	Fatal = zerolog.FatalLevel  // 4
	Panic = zerolog.PanicLevel  // 5
	NoLevel = zerolog.NoLevel   // 6
	Disabled = zerolog.Disabled // 7
)

// For continutiy, logging.config and config.json still use 
// Logrus's level conventions (0=debug, 5=panic)
// We'll translate when loading and saving configs
const (
	PanicConfig  = iota // zerolog.PanicLevel=5
	FatalConfig         // zerolog.FatalLevel=4
	ErrorConfig         // zerolog.ErrorLevel=3
	WarnConfig          // zerolog.WarnLevel=2
	InfoConfig          // zerolog.InfoLevel=1
	DebugConfig         // zerolog.DebugLevel=0
)

var libraryToConfigValues = map[zerolog.Level]Level{
	Debug: DebugConfig,
	Info: InfoConfig,
	Warn: WarnConfig,
	Error: ErrorConfig,
	Fatal: FatalConfig,
	Panic: PanicConfig,
}

func LevelToZerologl(l Level) zerolog.Level {
	return zerolog.Level(l)
}

func LevelToConfigValue(l Level) uint32 {
	if val, ok := libraryToConfigValues[l]; ok {
		return uint32(val)
	} else {
		// Default config level - update if that changes
		return DebugConfig
	}
}

func LevelFromConfigValue(cl int32) (Level, error) {
	for k, v := range libraryToConfigValues {
		if cl == int32(v) {
			return Level(k), nil
		}
	}
	return Level(Debug), errors.New(fmt.Sprintf("Logging Config level %d does not correspond to a logging level.", cl))
}

func LevelName(l Level) string {
	return l.String()
}