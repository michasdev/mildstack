package http

import (
	"context"
	"errors"
	"fmt"
	"net"
	stdhttp "net/http"
	"time"
)

type PortRegistrar interface {
	Serve(context.Context, int) error
}

type PortReleaser interface {
	Release(context.Context, int) error
}

type Server struct {
	registrar PortRegistrar
	router    *Router
	port      int
	server    *stdhttp.Server
}

func NewServer(registrar PortRegistrar, router *Router, port int) *Server {
	return &Server{
		registrar: registrar,
		router:    router,
		port:      port,
	}
}

func (s *Server) Start(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	} else if err := ctx.Err(); err != nil {
		return err
	}

	listener, err := net.Listen("tcp", s.address())
	if err != nil {
		return err
	}

	actualPort, ok := tcpPort(listener.Addr())
	if !ok {
		_ = listener.Close()
		return fmt.Errorf("http: unexpected listener address %T", listener.Addr())
	}

	if err := s.registrar.Serve(ctx, actualPort); err != nil {
		_ = listener.Close()
		return err
	}
	if releaser, ok := s.registrar.(PortReleaser); ok {
		defer func() {
			_ = releaser.Release(ctx, actualPort)
		}()
	}

	s.server = &stdhttp.Server{
		Handler: s.router.Engine(),
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.server.Serve(listener)
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, stdhttp.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		return s.Shutdown(context.Background())
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return s.server.Shutdown(shutdownCtx)
}

func (s *Server) address() string {
	return fmt.Sprintf(":%d", s.port)
}

func tcpPort(addr net.Addr) (int, bool) {
	tcpAddr, ok := addr.(*net.TCPAddr)
	if !ok {
		return 0, false
	}
	return tcpAddr.Port, true
}
