package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/riandyrn/otelchi"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/handler"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/middleware"
)

type Handlers struct {
	Health       *handler.HealthHandler
	Organization *handler.OrganizationHandler
}

func NewRouter(h *Handlers, auth *middleware.Auth) http.Handler {
	r := chi.NewRouter()
	r.Use(otelchi.Middleware("api-dashboard"))
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)

	r.Get("/healthz", h.Health.Check)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(auth.ValidateJWT)

		r.Route("/organizations", func(r chi.Router) {
			r.Use(auth.RequireRole("viewer", "editor", "admin", "owner"))
			r.Post("/", h.Organization.Create)
			r.Get("/", h.Organization.List)
			r.Get("/{id}", h.Organization.GetByID)
		})
	})

	return r
}
