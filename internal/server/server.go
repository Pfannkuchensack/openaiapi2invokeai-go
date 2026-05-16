package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/Pfannkuchensack/openaiapi2invokeai-go/internal/admin"
	"github.com/Pfannkuchensack/openaiapi2invokeai-go/internal/config"
	"github.com/Pfannkuchensack/openaiapi2invokeai-go/internal/invoke"
	"github.com/Pfannkuchensack/openaiapi2invokeai-go/internal/workflow"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	cfg      *config.Config
	router   chi.Router
	http     *http.Server
	log      *slog.Logger
	invoke   *invoke.Client
	registry *workflow.Registry
}

func New(cfg *config.Config, log *slog.Logger, invokeClient *invoke.Client, registry *workflow.Registry) *Server {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	s := &Server{
		cfg:      cfg,
		router:   r,
		log:      log,
		invoke:   invokeClient,
		registry: registry,
		http: &http.Server{
			Addr:         cfg.Addr(),
			Handler:      r,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: cfg.Timeout + 5*time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}

	s.routes()
	return s
}

func (s *Server) routes() {
	s.router.Get("/healthz", s.handleHealthz)

	// OpenAI-compatible API
	s.router.Group(func(r chi.Router) {
		if s.cfg.APIKey != "" {
			r.Use(s.bearerAuth)
		}
		r.Get("/v1/models", s.handleModels)
		r.Post("/v1/images/generations", s.handleImageGenerations)
		r.Post("/v1/images/edits", s.handleImageEdits)
		r.Post("/v1/images/variations", s.handleImageVariations)
	})

	// Admin UI
	adminHandler := admin.NewHandler(s.cfg, s.registry, s.invoke, s.log)
	s.router.Group(func(r chi.Router) {
		if s.cfg.AdminUser != "" {
			r.Use(s.basicAuth)
		}
		r.Mount("/admin", adminHandler.Routes())
		r.Handle("/admin/static/*", http.StripPrefix("/admin/static/", admin.StaticHandler()))
	})
}

func (s *Server) Start() error {
	s.log.Info("starting server", "addr", s.http.Addr)
	return s.http.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}
