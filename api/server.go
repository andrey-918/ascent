package api

import (
	"ascent/config"
	"context"
	"net/http"
)

type ApiServer struct {
	Config *config.Config
}

func New(config *config.Config) *ApiServer {
	return &ApiServer{Config: config}
}

func (s *ApiServer) ping(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ping"))
}

func (s *ApiServer) Start(ctx context.Context) error {
	mux := 
}