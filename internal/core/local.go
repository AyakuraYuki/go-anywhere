package core

import (
	"os"
	"os/user"
	"path/filepath"

	"github.com/AyakuraYuki/go-anywhere/internal/log"
)

func AnywhereConfigDir() string {
	usr, err := user.Current()
	if err == nil {
		return anywhereConfigDir(usr.HomeDir)
	}

	home, err := os.UserHomeDir()
	if err == nil {
		return anywhereConfigDir(home)
	}

	log.Scope("core").Errorf("Could not determine home directory")
	os.Exit(1)
	return "" // actually cannot reach here
}

func anywhereConfigDir(prefix string) string {
	return filepath.Join(prefix, ".local", ".anywhere")
}
