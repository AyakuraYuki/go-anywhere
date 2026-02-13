package config

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"

	"github.com/AyakuraYuki/go-anywhere/internal/log"
)

type Config struct {
	Host      string // server host ip or hostname
	Port      int    // server port
	Dir       string // the root directory for static files
	Silent    bool   // won't open browser automatically if enabled
	EnableLog bool   // print access log
	Fallback  string // enable history fallback
	Proxy     string // proxy URL
	Help      bool   // print help information
	Version   bool   // print version
}

func (cfg *Config) PortTLS() int { return cfg.Port + 1 }

func Parse() *Config {
	cfg := &Config{}

	pflag.StringVarP(&cfg.Host, "host", "h", "0.0.0.0", "server hostname")
	pflag.IntVarP(&cfg.Port, "port", "p", 8000, "server port")
	pflag.StringVarP(&cfg.Dir, "dir", "d", "./", "static file root directory")
	pflag.BoolVarP(&cfg.Silent, "silent", "s", false, "don't open browser automatically")
	pflag.BoolVarP(&cfg.EnableLog, "enable-log", "l", false, "print access log")
	pflag.StringVarP(&cfg.Fallback, "fallback", "f", "", "enable html5 history mode (eg: /index.html)")
	pflag.StringVar(&cfg.Proxy, "proxy", "", "proxy url (eg: http://localhost:7000/api)")
	pflag.BoolVar(&cfg.Help, "help", false, "print help information")
	pflag.BoolVarP(&cfg.Version, "version", "v", false, "print version")

	pflag.Usage = PrintHelp

	pflag.Parse()

	// Resolve the absolute path
	cfg.resolveRoot()

	// Assure listening host
	if cfg.Host == "" {
		cfg.Host = "0.0.0.0"
	}

	// Support assign port directly, like `anywhere 8888`
	// (priority is higher than `-p, --port` option)
	if args := pflag.Args(); len(args) > 0 {
		var port int
		_, err := fmt.Sscanf(args[0], "%d", &port)
		if err == nil && port > 0 && port < 65536 {
			cfg.Port = port
		}
	}

	// Verify port
	if cfg.Port < 1 && cfg.Port > 65534 {
		log.Scope("config").Errorf("invalid port %d (allowed: [1-65534])", cfg.Port)
		os.Exit(1)
	}

	return cfg
}

// Resolve root directory
func (cfg *Config) resolveRoot() {
	if cfg.Dir == "" {

		cwd, err := os.Getwd()
		if err != nil {
			log.Scope("config").Errorf("cannot get working directory: %v\n", err)
			os.Exit(1)
		}
		cfg.Dir = cwd

	} else {

		// expand ${HOME}
		cfg.Dir = os.ExpandEnv(cfg.Dir)

		// expand tilde
		usr, err := user.Current()
		if err != nil {
			log.Scope("config").Errorf("cannot get current user: %v\n", err)
			os.Exit(1)
		}
		if cfg.Dir == "~" {
			cfg.Dir = usr.HomeDir
		} else if strings.HasPrefix(cfg.Dir, "~/") {
			cfg.Dir = filepath.Join(usr.HomeDir, cfg.Dir[2:])
		}

		if absDir, err := filepath.Abs(cfg.Dir); err == nil {
			cfg.Dir = absDir
		}

	}

	// Verify root directory exists
	stat, err := os.Stat(cfg.Dir)
	if err != nil || !stat.IsDir() {
		log.Scope("config").Errorf("'%s' is not a valid directory\n", cfg.Dir)
		os.Exit(1)
	}
}

func PrintHelp() {
	fmt.Println(`anywhere - Run static file server anywhere

Usage:
  anywhere [options] [port]

Options:
  -p, --port <port>       Port number (default: 8000)
  -h, --host <hostname>   Hostname to bind (default: 0.0.0.0)
  -d, --dir <dir>         Root directory (default: current directory)
  -s, --silent            Silent mode, don't open browser
  -l, --enable-log        Enable access logging
  -f, --fallback <file>   Enable HTML5 history fallback (e.g. -f /index.html)
  --help                  Show this help message
  -v, --version           Show version

Examples:
  anywhere                    # Serve current dir on port 8000
  anywhere 8888               # Serve current dir on port 8888
  anywhere -p 8989            # Same as above
  anywhere -d /home/www       # Serve /home/www
  anywhere -s -l              # Silent + access logging
  anywhere -f /index.html     # SPA with HTML5 history fallback`)
}
