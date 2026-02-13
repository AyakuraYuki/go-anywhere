package core

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"

	"github.com/AyakuraYuki/go-anywhere/internal/config"
	"github.com/AyakuraYuki/go-anywhere/internal/handler"
)

func Server(cfg *config.Config) *server.Hertz {
	h := server.Default(
		server.WithHostPorts(fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)),
		server.WithDisablePrintRoute(true),
	)

	registerMiddlewaresAndRoutes(h, cfg)

	return h
}

func ServerTLS(cfg *config.Config, ips []string) (*server.Hertz, error) {
	crt, key, err := GenSelfSignedCert(ips)
	if err != nil {
		return nil, err
	}

	cert, err := tls.X509KeyPair(crt, key)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	h := server.Default(
		server.WithHostPorts(fmt.Sprintf("%s:%d", cfg.Host, cfg.PortTLS())),
		server.WithTLS(tlsConfig),
		server.WithDisablePrintRoute(true),
	)

	registerMiddlewaresAndRoutes(h, cfg)

	return h, nil
}

func registerMiddlewaresAndRoutes(h *server.Hertz, cfg *config.Config) {
	h.Use(handler.CORS())
	h.Use(handler.BrotliMiddleware())
	h.Use(handler.LogMiddleware(cfg.EnableLog))

	// HTML5 history fallback (if enabled)
	if cfg.Fallback != "" {
		h.Use(handler.HistoryFallbackMiddleware(cfg.Dir, handler.FallbackOptions{
			Index:   cfg.Fallback,
			Verbose: cfg.EnableLog,
		}))
	}

	// proxy
	if cfg.Proxy != "" {
		h.Use(handler.Proxy(cfg.Proxy))
	}

	handler.RegisterTemplate(h)

	// Catch-all route for static files and directory listing
	h.GET("/*filepath", handler.StaticFileHandler(cfg))
	h.HEAD("/*filepath", handler.StaticFileHandler(cfg))
	// Root path
	h.GET("/", func(c context.Context, ctx *app.RequestContext) {
		handler.StaticFileHandler(cfg)(c, ctx)
	})
}
