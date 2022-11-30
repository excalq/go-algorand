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
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func init() {
	// ConsoleWriter is good for CLI usage, less performant for daemons
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: logging.TimeFormatRFC3339Micro})
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(getCmd)
}

var rootCmd = &cobra.Command{
	Use:   "nodecfg",
	Short: "CLI for applying Algorand node configuration to a host",
	Long:  `CLI for retrieving and applying Algorand node configuration to the current host machine`,
	Run: func(cmd *cobra.Command, args []string) {
		// If no arguments passed, we should fallback to help

		cmd.HelpFunc()(cmd, args)
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
