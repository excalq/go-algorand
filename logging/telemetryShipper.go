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

// Telemetry Shipper, formerly known as telemetryHook
// Publishes telemetry events through the exporter/elastash HTTP client
package logging

import (
	"context"
	"fmt"

	"github.com/algorand/go-algorand/logging/exporters/elastash"
)

type telemetryShipper struct {
	client *elastash.Client
	ctx context.Context
	ctxCancel context.CancelFunc
	log Logger
}

// New shipper, using a stripped down HTTP client, which can write to a remote ingestion destination
func createTelemetryShipper(cfg TelemetryConfig) (shipper *telemetryShipper, err error) {
	// collectorURI := cfg.URI // ElasticSearch destination (TODO: Refactor)
	// esIndex := cfg.ChainID // e.g. beta-betanet-v1.0
	collectorURI := "https://logstash.deveks.algodev.network/"
	log := NewLogger()
	client, err := elastash.NewClient(
		elastash.SetURL(collectorURI),
		elastash.SetBasicAuth(cfg.UserName, cfg.Password),
		elastash.SetGzip(true),
		elastash.SetLogger(log), 
	)
	ctx, cancel := context.WithCancel(context.TODO())
	shipClient := &telemetryShipper{
		client: client,
		ctx: ctx,
		ctxCancel: cancel,
		log: log,
	}
	
	return shipClient, err
}

func (t *telemetryShipper) Publish(entry telEntry) (err error) {
	// Problem: zerolog doesn't expose internals of *Event. This is a problem for doing anything useful...
	res, err := t.client.Post(t.ctx, "some-path", entry.message)
	if err != nil {
		t.log.Errorf("Telemetry Publish - Error: %w", err)
		return err
	}
	t.log.Infof("Telemetry Publish! Status: %v, Result: %v", res.Status, res.Result)

	// TODO: Proper handling here...
	fmt.Printf("Tele Send: Status: %d\n", res.Status)
    fmt.Printf("Tele Send: Body: %s\n", string(res.Result))
	return nil
}
