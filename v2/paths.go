package main

import (
	"path/filepath"

	"github.com/xyproto/env/v2"
)

var (
	userConfigDir = env.Dir("XDG_CONFIG_HOME", "~/.config")
	userCacheDir  = env.Dir("XDG_CACHE_HOME", "~/.cache")

	locationHistoryFilename = filepath.Join(userCacheDir, "o", "locations.txt")
	quickHelpToggleFilename = filepath.Join(userCacheDir, "o", "quickhelp.txt")

	vimLocationHistoryFilename   = env.ExpandUser("~/.viminfo")
	nvimLocationHistoryFilename  = filepath.Join(env.Dir("XDG_DATA_HOME", "~/.local/share"), "nvim", "shada", "main.shada")
	emacsLocationHistoryFilename = env.ExpandUser("~/.emacs.d/places")
)
