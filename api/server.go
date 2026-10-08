package api

import (
	"ascent/config"
	"ascent/store"
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
	store  *store.Store
}

func New(cfg *config.Config, logger *slog.Logger, store *store.Store) *ApiServer {
	return &ApiServer{
		config: cfg,
		logger: logger,
		store:  store,
	}
}

func (s *ApiServer) ping(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("pong"))
}

func (s *ApiServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ping", s.ping)
	mux.HandleFunc("POST /auth/signup", s.signupHandler)

	middleware := NewLoggerMiddelware(s.logger)
	server := &http.Server{
		Addr:    net.JoinHostPort(s.config.ApiServerHost, s.config.ApiServerPort),
		Handler: middleware(mux),
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
