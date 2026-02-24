package handler

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"github.com/AyakuraYuki/go-anywhere/internal/config"
	"github.com/AyakuraYuki/go-anywhere/internal/log"
)

var (
	//go:embed templates
	templateFS embed.FS

	nonCacheRootFS = &app.FS{
		Root:               "/",
		GenerateIndexPages: true,
		Compress:           false,
		AcceptByteRange:    true,
	}
)

type Config struct {
	Dir      string // root directory to serve
	Log      bool   // enable access logging
	Fallback string // HTML5 history fallback index file (empty as disabled)
}

type FallbackOptions struct {
	// Fallback file path (default: "/index.html")
	Index string

	// Custom rewrite rules, evaluated in order
	Rewrites []RewriteRule

	// If true, paths with dots can also be rewritten
	DisableDotRule bool

	// Accept values that qualify as HTML (default: ["text/html", "*/*"])
	HTMLAcceptHeaders []string

	// Enable debug logging
	Verbose bool
}

type RewriteContext struct {
	ParsedURL *url.URL // The parsed request URL
	Match     []string // Regex submatch results from the From pattern
}

// RewriteFunc is a function-based rewrite target.
type RewriteFunc func(ctx RewriteContext) string

// RewriteRule defines a URL rewrite mapping.
// To can be either a string (returned as-is) or a RewriteFunc (called with
// context).
type RewriteRule struct {
	From *regexp.Regexp // Pattern to match against the request path
	To   any            // string or RewriteFunc
}

func CORS() app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		ctx.Header("Access-Control-Allow-Origin", "*")
		ctx.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		ctx.Header("Access-Control-Allow-Headers", "*")

		if string(ctx.Method()) == "OPTIONS" {
			ctx.AbortWithStatus(consts.StatusNoContent)
			return
		}

		ctx.Next(c)
	}
}

func LogMiddleware(enabled bool) app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		if enabled {
			log.Info().Str("scope", "access-log").
				Str("method", string(ctx.Method())).
				Str("path", string(ctx.Path()))
		}
		ctx.Next(c)
	}
}

func BrotliMiddleware() app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		urlPath := string(ctx.Path())

		if strings.HasSuffix(strings.ToLower(urlPath), ".br") {
			// get the real file extension before .br
			realName := urlPath[:len(urlPath)-3]
			realExt := filepath.Ext(realName)
			contentType := mime.TypeByExtension(realExt)
			if contentType == "" {
				contentType = "application/octet-stream"
			}
			ctx.Header("Content-Type", contentType)
			ctx.Header("Content-Encoding", "br")
		}

		ctx.Next(c)
	}
}

// HistoryFallbackMiddleware rewrites requests to the fallback index file
// following these rules:
//
//  1. Only GET/HEAD requests
//  2. Client must send an Accept header
//  3. Skip if Accept prefers application/json
//  4. Only rewrite if Accept includes text/html
//  5. Apply custom rewrite rules first (if any)
//  6. Dot Rule: if the path's last segment contains a dot, treat it as a
//     file → skip
//  7. Otherwise, rewrite to the fallback index
func HistoryFallbackMiddleware(dir string, opts FallbackOptions) app.HandlerFunc {
	if opts.Index == "" {
		opts.Index = "/index.html"
	}

	logger := func(format string, args ...any) {
		if opts.Verbose {
			log.Trace().Str("scope", "history-fallback").Msgf(format, args...)
		}
	}

	return func(c context.Context, ctx *app.RequestContext) {
		method := string(ctx.Method())
		reqURL := string(ctx.Request.URI().RequestURI())

		// 1. Only GET / HEAD
		if method != "GET" && method != "HEAD" {
			logger("Not rewriting %s %s — method is not GET or HEAD.", method, reqURL)
			ctx.Next(c)
			return
		}

		// 2. Must have an Accept header
		accept := string(ctx.GetHeader("Accept"))
		if accept == "" {
			logger("Not rewriting %s %s — no Accept header.", method, reqURL)
			ctx.Next(c)
			return
		}

		// 3. Skip if client prefers JSON (Accept starts with application/json)
		if strings.HasPrefix(accept, "application/json") {
			logger("Not rewriting %s %s — client prefers JSON.", method, reqURL)
			ctx.Next(c)
			return
		}

		// 4. Must accept text/html
		if !acceptsHTML(accept, opts.HTMLAcceptHeaders) {
			logger("Not rewriting %s %s — client does not accept HTML.", method, reqURL)
			ctx.Next(c)
			return
		}

		// Parse URL to get pathname
		parsedURL, err := url.Parse(reqURL)
		if err != nil {
			ctx.Next(c)
			return
		}
		pathname := parsedURL.Path

		// 5. Check custom rewrite rules
		for _, rule := range opts.Rewrites {
			matches := rule.From.FindStringSubmatch(pathname)
			if matches != nil {
				rewriteTarget := evaluateRewriteRule(parsedURL, matches, rule.To)
				if len(rewriteTarget) > 0 && rewriteTarget[0] != '/' {
					logger("Warning: non-absolute rewrite target %q for URL %s", rewriteTarget, reqURL)
				}
				logger("Rewriting %s %s to %s (matched rule)", method, reqURL, rewriteTarget)
				ctx.Request.SetRequestURI(rewriteTarget)
				ctx.Next(c)
				return
			}
		}

		// 6. Dot Rule: if the last segment of the path contains a dot,
		//    assume it's a file
		if !opts.DisableDotRule {
			lastSlash := strings.LastIndex(pathname, "/")
			lastDot := strings.LastIndex(pathname, ".")
			if lastDot > lastSlash {
				logger("Not rewriting %s %s — path includes a dot (.) character.", method, reqURL)
				ctx.Next(c)
				return
			}
		}

		// 7. Rewrite to fallback index
		rewriteTarget := opts.Index
		logger("Rewriting %s %s to %s", method, reqURL, rewriteTarget)

		// Serve the fallback file directly
		fallbackPath := filepath.Join(dir, rewriteTarget)
		if _, err := os.Stat(fallbackPath); err == nil {
			ctx.File(fallbackPath)
			ctx.Abort()
			return
		}

		// Fallback file not found, continue to next handler
		ctx.Next(c)
	}
}

// acceptsHTML checks if the Accept header includes any of the HTML accept
// values.
func acceptsHTML(accept string, htmlAcceptHeaders []string) bool {
	if len(htmlAcceptHeaders) == 0 {
		htmlAcceptHeaders = []string{"text/html", "*/*"}
	}
	for _, header := range htmlAcceptHeaders {
		if strings.Contains(accept, header) {
			return true
		}
	}
	return false
}

// evaluateRewriteRule resolves the rewrite target.
//   - string  → returned as-is
//   - RewriteFunc → called with {parsedURL, match} context
func evaluateRewriteRule(parsedURL *url.URL, matches []string, to any) string {
	switch target := to.(type) {
	case string:
		return target
	case RewriteFunc:
		return target(RewriteContext{
			ParsedURL: parsedURL,
			Match:     matches,
		})
	default:
		panic(fmt.Sprintf("rewrite rule To must be string or RewriteFunc, got %T", to))
	}
}

// StaticFileHandler serves static files with directory listing fallback
func StaticFileHandler(cfg *config.Config) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		urlPath := string(c.Path())

		// Decode URL path
		decoded, err := url.PathUnescape(urlPath)
		if err != nil {
			decoded = urlPath
		}

		// Security: prevent path traversal
		cleanPath := filepath.Clean(decoded)
		absPath := filepath.Join(cfg.Dir, cleanPath)

		// Ensure we don't escape the root directory
		absDir, _ := filepath.Abs(cfg.Dir)
		absReq, _ := filepath.Abs(absPath)
		if !strings.HasPrefix(absReq, absDir) {
			c.String(consts.StatusForbidden, "403 Forbidden")
			return
		}

		info, err := os.Stat(absPath)
		if err != nil {
			c.String(consts.StatusNotFound, "404 Not Found")
			return
		}

		// If it's a directory
		if info.IsDir() {
			// Ensure trailing slash for directories
			if !strings.HasSuffix(urlPath, "/") {
				c.Redirect(consts.StatusMovedPermanently, []byte(urlPath+"/"))
				return
			}

			// Try to serve index.html
			indexPath := filepath.Join(absPath, "index.html")
			if _, err := os.Stat(indexPath); err == nil {
				c.File(indexPath)
				return
			}

			// Generate directory listing
			data, err := BuildDirListData(absPath, urlPath)
			if err != nil {
				c.String(consts.StatusInternalServerError, "Error listing directory: %v", err)
				return
			}

			c.HTML(consts.StatusOK, "index.gohtml", data)
			return
		}

		// Serve the file
		c.FileFromFS(absPath, nonCacheRootFS)
	}
}

func RegisterTemplate(h *server.Hertz) {
	tmpl, err := template.New("hertz-html-engine").ParseFS(templateFS, "templates/*")
	if err != nil {
		log.Error().Err(err).Msg("cannot load html templates")
		os.Exit(1)
	}

	h.SetHTMLTemplate(tmpl)
}
