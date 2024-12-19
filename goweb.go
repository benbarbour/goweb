package goweb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"
)

// A wrapper around http.Server with some extra functionality.
type Server struct {
	ctx             context.Context
	errGrp          *errgroup.Group
	logger          *slog.Logger
	mux             *http.ServeMux
	profileFunc     func() error
	cancelCtx       context.CancelFunc
	disableSigCatch context.CancelFunc
}

var ErrServerAlreadyStarted = errors.New("server already started")

func NewServer(ctx context.Context, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.New(&discardLogHandler{})
	}

	s := &Server{
		logger: logger,
		mux:    http.NewServeMux(),
	}

	s.ctx, s.cancelCtx = context.WithCancel(ctx)

	return s
}

func (s *Server) Start(listenAddr string) error {
	if s.Started() {
		return ErrServerAlreadyStarted
	}

	s.errGrp, s.ctx = errgroup.WithContext(s.ctx)
	s.ctx, s.disableSigCatch = signal.NotifyContext(s.ctx, syscall.SIGINT, syscall.SIGTERM)

	if s.profileFunc != nil {
		s.errGrp.Go(s.profileFunc)
	}

	s.errGrp.Go(func() error {
		s.logger.InfoContext(s.ctx, "starting server", "addr", listenAddr)
		return serve(s.ctx, listenAddr, s.mux)
	})

	err := s.errGrp.Wait() // Wait for goroutines

	switch err {
	case nil, context.Canceled, http.ErrServerClosed:
		fmt.Print("\r") // nuke the CTRL-C character
		s.logger.InfoContext(s.ctx, "shutdown gracefully")
		err = nil
	}

	s.Stop()

	return err
}

func (s *Server) Stop() {
	if !s.Started() {
		return
	}

	s.disableSigCatch()
	s.cancelCtx()
	s.errGrp = nil
}

// EnableProfiler will cause Start() to also start a net/http/pprof listening at addr.
// Calling it after Start() returns a ServerAlreadyStartedError.
func (s *Server) EnableProfiler(addr string) error {
	if s.Started() {
		return ErrServerAlreadyStarted
	}

	s.profileFunc = func() error {
		cmd := fmt.Sprintf(
			"go tool pprof -http :6061 http://%s/debug/pprof/profile", addr,
		)
		s.logger.InfoContext(
			s.ctx,
			"starting profiler",
			"addr",
			addr,
			"example-cmd",
			cmd,
		)
		return serve(s.ctx, addr, newProfiler(s.ctx))
	}

	return nil
}

func (s *Server) Started() bool {
	return s.errGrp != nil
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
