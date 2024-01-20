package goweb

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os/signal"
	"strings"
	"syscall"

	"golang.org/x/sync/errgroup"
)

type Server struct {
	ListenAddr  string
	ProfileAddr string
	Mux         *http.ServeMux
	Logger      *slog.Logger // may be omitted safely
}

func (s *Server) Start(ctx context.Context) error {
	if s.Logger == nil {
		s.Logger = slog.New(&discardHandler{})
	}

	// Graceful shutdown on signal reception
	ctx, done := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer done()

	g, ctx := errgroup.WithContext(ctx)

	if s.ProfileAddr != "" {
		g.Go(func() error {
			cmd := fmt.Sprintf(
				"go tool pprof -http :6061 http://%s/debug/pprof/profile", s.ProfileAddr,
			)
			s.Logger.InfoContext(
				ctx,
				"starting profiler",
				"addr",
				s.ProfileAddr,
				"example-cmd",
				cmd,
			)
			return serve(ctx, s.ProfileAddr, newProfiler(ctx))
		})
	}

	g.Go(func() error {
		s.Logger.InfoContext(ctx, "starting server", "addr", s.ListenAddr)
		return serve(ctx, s.ListenAddr, &requestLogger{ctx, s.Mux, s.Logger})
	})

	// Wait for goroutines
	err := g.Wait()

	switch err {
	case nil, context.Canceled, http.ErrServerClosed:
		fmt.Print("\r") // nuke the CTRL-C character
		s.Logger.InfoContext(ctx, "shutdown gracefully")
	}
	return err
}

func HttpXXX(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}

func Http400(w http.ResponseWriter) {
	HttpXXX(w, http.StatusBadRequest)
}

func Http405(w http.ResponseWriter, allowed ...string) {
	w.Header().Set("Allow", strings.Join(allowed, ", "))
	HttpXXX(w, http.StatusMethodNotAllowed)
}

func Http500(ctx context.Context, w http.ResponseWriter, err error, logMsg string) {
	slog.ErrorContext(ctx, logMsg, "err", err)
	HttpXXX(w, http.StatusInternalServerError)
}

func serve(ctx context.Context, addr string, handler http.Handler) error {
	s := http.Server{
		Addr:    addr,
		Handler: handler,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}

	go func() {
		<-ctx.Done() // wait for stop signal
		s.Shutdown(context.Background())
	}()

	return s.ListenAndServe()
}
