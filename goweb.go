package goweb

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"
)

// A wrapper around http.Server with some extra functionality.
// To shut it down gracefully cancel the context passed to Start().
type Server struct {
	Logger      *slog.Logger // defaults to the package level Logger if not given
	mux         *http.ServeMux
	ListenAddr  string // passed to http.ListenAndServe()
	ProfileAddr string // If not empty starts a net/http/pprof listening on this address
}

func (s *Server) Start(ctx context.Context) error {
	if s.mux != nil {
		return fmt.Errorf("already started")
	}

	if s.Logger == nil {
		if Logger != nil {
			s.Logger = Logger
		} else {
			s.Logger = slog.New(&discardLogHandler{})
		}
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

	s.mux = http.NewServeMux()

	g.Go(func() error {
		s.Logger.InfoContext(ctx, "starting server", "addr", s.ListenAddr)
		return serve(ctx, s.ListenAddr, s.mux)
	})

	err := g.Wait() // Wait for goroutines
	s.mux = nil

	switch err {
	case nil, context.Canceled, http.ErrServerClosed:
		fmt.Print("\r") // nuke the CTRL-C character
		s.Logger.InfoContext(ctx, "shutdown gracefully")
		return nil
	}

	return err
}

func (s *Server) Started() bool {
	return s.mux != nil
}

func (s *Server) Handle(pattern string, handler http.Handler) {
	s.mux.Handle(pattern, handler)
}

func (s *Server) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	s.mux.HandleFunc(pattern, handler)
}

func (s *Server) Handler(r *http.Request) (h http.Handler, pattern string) {
	return s.mux.Handler(r)
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
