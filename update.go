package main

import (
	"github.com/datapointchris/goclikit"
	"github.com/datapointchris/goselfupdate"
)

// updateConfig describes where toolbox's releases come from. Shared by the
// `update` command and the daily check in main, so the two cannot point at
// different releases.
func updateConfig() goselfupdate.Config {
	return goselfupdate.Config{
		Owner:   "datapointchris",
		Repo:    "toolbox",
		Binary:  "toolbox",
		Version: getVersion(),
	}
}

func init() {
	rootCmd.AddCommand(goclikit.UpdateCommand(updateConfig()))
}
