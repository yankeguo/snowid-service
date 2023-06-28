package ufx

import (
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"net/http"
)

func instrumentHTTPHandler(pattern string, h http.Handler) http.Handler {
	return otelhttp.NewHandler(otelhttp.WithRouteTag(pattern, h), pattern)
}
