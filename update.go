package main

import (
	"github.com/datapointchris/goselfupdate"
	"github.com/datapointchris/goselfupdate/cobracmd"
)

func init() {
	rootCmd.AddCommand(cobracmd.New(goselfupdate.Config{
		Owner:   "datapointchris",
		Repo:    "toolbox",
		Binary:  "toolbox",
		Version: getVersion(),
	}))
}
