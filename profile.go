package goweb

import (
	"context"
	"net/http"
	_ "net/http/pprof"
)

func newProfiler(ctx context.Context) http.Handler {
	// The pprof import adds to DefaultServeMux
	return http.DefaultServeMux
}
