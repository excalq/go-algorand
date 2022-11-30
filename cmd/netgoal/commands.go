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

package main

import (
	"fmt"
	"os"

	"github.com/algorand/go-algorand/logging"
	"github.com/algorand/go-deadlock"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/spf13/cobra"
)

// var log *logrus.Logger

func init() {

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: logging.TimeFormatRFC3339Micro})
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	// disable the deadlock detection for this tool.
	deadlock.Opts.Disable = true
}

var rootCmd = &cobra.Command{
	Use:   "netgoal",
	Short: "CLI for building and deploying algorand networks",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		// If no arguments passed, we should fallback to help

		cmd.HelpFunc()(cmd, args)
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Error().Err(err)
		os.Exit(1)
	}
}

func reportInfoln(args ...interface{}) {
	log.Info().Msg(fmt.Sprint(args...))
}

func reportInfof(format string, args ...interface{}) {
	log.Info().Msgf(format+"\n", args...)
}

func reportError(msg string) {
	log.Error().Msg(msg)
}

func reportErrorf(format string, args ...interface{}) {
	log.Error().Msgf(format+"\n", args...)
}
