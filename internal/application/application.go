package application

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/psds-microservice/helpy/db"
	"github.com/psds-microservice/session-manager-service/internal/config"
	"github.com/psds-microservice/session-manager-service/internal/database"
	"github.com/psds-microservice/session-manager-service/internal/handler"
	"github.com/psds-microservice/session-manager-service/internal/router"
	"github.com/psds-microservice/session-manager-service/internal/service"
)

type API struct {
	cfg *config.Config
	srv *http.Server
}

func NewAPI(cfg *config.Config) (*API, error) {
	if err := database.MigrateUp(cfg.DatabaseURL()); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	conn, err := db.Open(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}

	sessionSvc := service.NewSessionService(conn)
	sessionHandler := handler.NewSessionHandler(sessionSvc)
	r := router.New(sessionHandler)

	addr := cfg.AppHost + ":" + cfg.HTTPPort
	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return &API{cfg: cfg, srv: srv}, nil
}

func (a *API) Run(ctx context.Context) error {
	host := a.cfg.AppHost
	if host == "0.0.0.0" {
		host = "localhost"
	}
	base := "http://" + host + ":" + a.cfg.HTTPPort
	log.Printf("HTTP server listening on %s", a.srv.Addr)
	log.Printf("  Swagger UI:    %s/swagger", base)
	log.Printf("  Swagger spec:  %s/swagger/openapi.json", base)
	log.Printf("  Health:        %s/health", base)
	log.Printf("  Ready:         %s/ready", base)
	log.Printf("  Session:       %s/session/", base)

	go func() {
		if err := a.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("http: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := a.srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("http shutdown: %w", err)
	}
	return nil
}
