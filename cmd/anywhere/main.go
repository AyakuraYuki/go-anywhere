package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/jedib0t/go-pretty/v6/table"

	"github.com/AyakuraYuki/go-anywhere/internal/config"
	"github.com/AyakuraYuki/go-anywhere/internal/core"
	"github.com/AyakuraYuki/go-anywhere/internal/log"
	"github.com/AyakuraYuki/go-anywhere/internal/signals"
)

var version string

func main() {
	cfg := config.Parse()

	if cfg.Help {
		config.PrintHelp()
		os.Exit(0)
	}

	if cfg.Version {
		fmt.Println(version)
		os.Exit(0)
	}

	if cfg.UninstallCA {
		err := core.UninstallCA()
		if err != nil {
			log.Error().Err(err).Msg("Uninstall root CA failed")
			os.Exit(1)
		}
		os.Exit(0)
	}

	// --- Resolve ip addresses
	allIPs, err := core.AllIPAddresses()
	if err != nil {
		log.Error().Err(err).Msg("Cannot load net interfaces")
		os.Exit(1)
	}

	// --- Suppress Hertz default logs for cleaner output
	hlog.SetLevel(hlog.LevelWarn)

	// --- Build server host ports
	h := core.Server(cfg)
	hs, err := core.ServerTLS(cfg, allIPs)
	if err != nil {
		log.Warn().Err(err).Msg("An issue occurred when preparing tls server, skipped")
	}

	// --- Start Hertz server
	tlsStarted := false
	go func() { h.Spin() }()
	if hs != nil {
		go func() { hs.Spin() }()
		tlsStarted = true
	}

	// --- Print startup message
	printStartup(cfg, allIPs, tlsStarted)
	// --- Open the system default browser
	openBrowser(cfg, allIPs)

	// --- Hung the program
	signals.GraceStop(func() {
		_ = h.Shutdown(context.Background())
		_ = h.Close()
		h = nil

		if hs != nil {
			_ = hs.Shutdown(context.Background())
			_ = hs.Close()
			hs = nil
		}
	})
}

func printStartup(cfg *config.Config, allIPs []string, tlsStarted bool) {
	var (
		portString    string
		portStringTLS string
	)

	if cfg.Port != 80 {
		portString = fmt.Sprintf(":%d", cfg.Port)
	}
	if cfg.PortTLS() != 443 {
		portStringTLS = fmt.Sprintf(":%d", cfg.PortTLS())
	}

	t := table.NewWriter()
	t.SetStyle(table.StyleLight)
	t.SetOutputMirror(os.Stdout)

	rows := []table.Row{
		{fmt.Sprintf("anywhere v%s", version)},
		{""},
		{fmt.Sprintf("Serving: %-30s", cfg.Dir)},
		{""},
		{"HTTP running at:"},
	}
	for _, ip := range allIPs {
		u := fmt.Sprintf("http://%s%s", ip, portString)
		rows = append(rows, table.Row{
			fmt.Sprintf("  * %-33s", u),
		})
	}
	rows = append(rows, table.Row{
		fmt.Sprintf("  * %-33s", fmt.Sprintf("http://127.0.0.1%s", portString)),
	})

	if tlsStarted {
		rows = append(rows, table.Row{""})
		rows = append(rows, table.Row{"Also running at:"})
		for _, ip := range allIPs {
			u := fmt.Sprintf("https://%s%s", ip, portStringTLS)
			rows = append(rows, table.Row{
				fmt.Sprintf("  * %-33s", u),
			})
		}
		rows = append(rows, table.Row{
			fmt.Sprintf("  * %-33s", fmt.Sprintf("https://127.0.0.1%s", portStringTLS)),
		})
	}

	t.AppendRows(rows)

	t.Render()
}

func openBrowser(cfg *config.Config, allIPs []string) {
	if cfg.Silent {
		return
	}

	var displayHost string

	if len(allIPs) > 0 {
		displayHost = allIPs[0]
	} else {
		displayHost = "127.0.0.1"
	}

	openURL := fmt.Sprintf("https://%s:%d/", displayHost, cfg.PortTLS())
	err := core.OpenBrowser(openURL)
	if err != nil {
		log.Error().Err(err).Msg("cannot open browser")
	}
}
