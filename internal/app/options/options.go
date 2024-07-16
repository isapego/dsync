/*
 * Copyright (C) 2024 Adiom, Inc.
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */
package options

import (
	"github.com/urfave/cli/v2"
)

type Options struct {
	Verbosity string

	SrcConnString        string
	DstConnString        string
	StateStoreConnString string

	NamespaceFrom []string

	Verify  bool
	Cleanup bool
}

func NewFromCLIContext(c *cli.Context) Options {
	o := Options{}

	o.Verbosity = c.String("verbosity")
	o.SrcConnString = c.String("source")
	o.DstConnString = c.String("destination")
	o.StateStoreConnString = c.String("metadata")
	o.NamespaceFrom = c.Generic("namespace").(*ListFlag).Values
	o.Verify = c.Bool("verify")
	o.Cleanup = c.Bool("cleanup")

	return o
}
