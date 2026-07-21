package main

import (
	"os"
	"path/filepath"
)

// xdgDataHome returns $XDG_DATA_HOME, or its spec default (~/.local/share) when
// unset. Authored, versioned content (like the tool registry, owned by dotfiles
// and symlinked into place) lives here.
func xdgDataHome() string {
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		panic("cannot determine home directory: " + err.Error())
	}
	return filepath.Join(home, ".local", "share")
}

// xdgStateHome returns $XDG_STATE_HOME, or its spec default (~/.local/state)
// when unset. Mutable state the tool writes (like reminder dates) lives here;
// whether it replicates across machines is the sync layer's concern, not ours.
func xdgStateHome() string {
	if d := os.Getenv("XDG_STATE_HOME"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		panic("cannot determine home directory: " + err.Error())
	}
	return filepath.Join(home, ".local", "state")
}
