package core

import (
	"os"
	"os/user"
	"path/filepath"

	"github.com/AyakuraYuki/go-anywhere/internal/log"
)

func anywhereConfigDir() string {
	usr, err := user.Current()
	if err == nil {
		return filepath.Join(usr.HomeDir, ".local", "go-anywhere")
	}

	home, err := os.UserHomeDir()
	if err == nil {
		return filepath.Join(home, ".local", "go-anywhere")
	}

	log.Error().Str("scope", "core").Msg("Could not determine home directory")
	os.Exit(1)
	return "" // actually cannot reach here
}

func caDir() string {
	return filepath.Join(anywhereConfigDir(), "ca")
}

func caCertPath() string {
	return filepath.Join(caDir(), "rootCA.pem")
}

func caKeyPath() string {
	return filepath.Join(caDir(), "rootCA.key")
}
