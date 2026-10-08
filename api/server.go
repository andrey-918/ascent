package api

import (
	"ascent/config"
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"
)

type ApiServer struct {
	config *config.Config
	logger *slog.Logger
}

func New(cfg *config.Config, logger *slog.Logger) *ApiServer {
	return &ApiServer{
		config: cfg,
		logger: logger,
	}
}

func (s *ApiServer) ping(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("pong"))
}

func (s *ApiServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", s.ping)

	server := &http.Server{
		Addr:    net.JoinHostPort(s.config.ApiServerHost, s.config.ApiServerPort),
		Handler: mux,
	}

	go func() {
		s.logger.Info("starting server on", "port", s.config.ApiServerPort)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("apiserver faile to listen and server", "error", err)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("apiserver faile to shutdown", "error", err)
		}
	}()

	wg.Wait()
	return nil
}
