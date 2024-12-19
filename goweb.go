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

type Server struct {
	Handler     http.Handler
	Logger      *slog.Logger
	ListenAddr  string
	ProfileAddr string
}

func (s *Server) Start(ctx context.Context) error {
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

	g.Go(func() error {
		s.Logger.InfoContext(ctx, "starting server", "addr", s.ListenAddr)
		return serve(ctx, s.ListenAddr, &requestLogger{ctx, s.Handler, s.Logger})
	})

	// Wait for goroutines
	err := g.Wait()

	switch err {
	case nil, context.Canceled, http.ErrServerClosed:
		fmt.Print("\r") // nuke the CTRL-C character
		s.Logger.InfoContext(ctx, "shutdown gracefully")
		return nil
	}
	return err
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
