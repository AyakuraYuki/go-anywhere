package handler

import (
	"context"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/hertz-contrib/reverseproxy"
)

func Proxy(proxyURL string) app.HandlerFunc {
	isURL := strings.HasPrefix(proxyURL, "https://") || strings.HasPrefix(proxyURL, "http://")
	if !isURL {
		return func(c context.Context, ctx *app.RequestContext) {
			ctx.Next(c)
		}
	}

	proxy, err := reverseproxy.NewSingleHostReverseProxy(proxyURL)
	if err != nil {
		return func(c context.Context, ctx *app.RequestContext) {
			ctx.Next(c)
		}
	}

	return func(c context.Context, ctx *app.RequestContext) {
		proxy.ServeHTTP(c, ctx)
		ctx.Abort()
	}
}
