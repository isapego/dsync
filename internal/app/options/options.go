/*
 * Copyright (C) 2024 Adiom, Inc.
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */
package options

import (
	"fmt"
	"time"

	connectorMongo "github.com/adiom-data/dsync/connectors/mongo"
	"github.com/urfave/cli/v2"
)

type Options struct {
	Verbosity  string
	Sourcetype string

	SrcConnString        string
	DstConnString        string
	StateStoreConnString string

	Logfile string

	NamespaceFrom []string

	Verify   bool
	Cleanup  bool
	Progress bool

	CosmosDeletesEmu bool

	Pprof bool

	PprofPort uint
	WebPort   uint

	LoadLevel                            string
	CosmosInitialSyncNumParallelCopiers  int
	CosmosNumParallelWriters             int
	CosmosNumParallelIntegrityCheckTasks int
	CosmosNumParallelPartitionWorkers    int
	CosmosMaxNumNamespaces               int
	CosmosServerConnectTimeout           time.Duration
	CosmosPingTimeout                    time.Duration
	CosmosCdcResumeTokenUpdateInterval   time.Duration
	CosmosWriterMaxBatchSize             int
	CosmosTargetDocCountPerPartition     int64
	CosmosDeletesCheckInterval           time.Duration
}

func NewFromCLIContext(c *cli.Context) (Options, error) {
	o := Options{}

	o.Verbosity = c.String("verbosity")
	o.Sourcetype = c.String("sourcetype")
	o.SrcConnString = c.String("source")
	o.DstConnString = c.String("destination")
	o.StateStoreConnString = c.String("metadata")
	o.Logfile = c.String("logfile")
	o.NamespaceFrom = c.Generic("namespace").(*ListFlag).Values
	o.Verify = c.Bool("verify")
	o.Cleanup = c.Bool("cleanup")
	o.Progress = c.Bool("progress")
	o.Pprof = c.Bool("pprof")
	o.LoadLevel = c.String("load-level")
	o.PprofPort = c.Uint("pprof-port")
	o.WebPort = c.Uint("web-port")

	o.CosmosInitialSyncNumParallelCopiers = c.Int("cosmos-parallel-copiers")
	o.CosmosNumParallelWriters = c.Int("cosmos-parallel-writers")
	o.CosmosNumParallelIntegrityCheckTasks = c.Int("cosmos-parallel-integrity-check")
	o.CosmosNumParallelPartitionWorkers = c.Int("cosmos-parallel-partition-workers")
	o.CosmosMaxNumNamespaces = c.Int("cosmos-max-namespaces")
	o.CosmosServerConnectTimeout = time.Duration(c.Int("cosmos-server-timeout")) * time.Second
	o.CosmosPingTimeout = time.Duration(c.Int("ping-timeout")) * time.Second
	o.CosmosCdcResumeTokenUpdateInterval = time.Duration(c.Int("cosmos-resume-token-interval")) * time.Second
	o.CosmosWriterMaxBatchSize = c.Int("cosmos-writer-batch-size")
	o.CosmosTargetDocCountPerPartition = c.Int64("cosmos-doc-partition")
	o.CosmosDeletesCheckInterval = time.Duration(c.Int("cosmos-delete-interval")) * time.Second

	// Infer source type if not provided
	if o.Sourcetype == "" && o.SrcConnString != "/dev/random" {
		mongoFlavor := connectorMongo.GetMongoFlavor(o.SrcConnString)
		if mongoFlavor == connectorMongo.FlavorCosmosDB {
			o.Sourcetype = "CosmosDB"
		} else {
			o.Sourcetype = "MongoDB"
		}
		fmt.Printf("Inferred source type: %v\n", o.Sourcetype)
	}

	o.CosmosDeletesEmu = c.Bool("cosmos-deletes-cdc")
	if o.Sourcetype != "CosmosDB" && o.CosmosDeletesEmu {
		return o, fmt.Errorf("cosmos-deletes-cdc flag is only valid for CosmosDB source")
	}
	if (o.DstConnString == "/dev/null") && o.CosmosDeletesEmu {
		// /dev/null doesn't offer a persistent index
		return o, fmt.Errorf("cosmos-deletes-cdc flag cannot be used with /dev/null destination")
	}

	return o, nil
}
