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
	Health        *handler.HealthHandler
	Organization  *handler.OrganizationHandler
	User          *handler.UserHandler
	OrgInvite     *handler.OrgInviteHandler
	PublisherApp  *handler.PublisherAppHandler
	APIKey        *handler.APIKeyHandler
	PublisherRule *handler.PublisherRuleHandler
}

func NewRouter(h *Handlers, auth *middleware.Auth) http.Handler {
	r := chi.NewRouter()
	r.Use(otelchi.Middleware("api-dashboard"))
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)

	r.Get("/healthz", h.Health.Check)

	r.Route("/v1", func(r chi.Router) {
		r.Use(auth.ValidateJWT)

		r.Get("/me", h.User.GetMe)

		r.Route("/organizations", func(r chi.Router) {
			r.Use(auth.RequireRole("viewer", "editor", "admin", "owner"))
			r.Post("/", h.Organization.Create)
			r.Get("/", h.Organization.List)
			r.Get("/{id}", h.Organization.GetByID)
		})

		r.Route("/orgs/{id}/invites", func(r chi.Router) {
			r.Use(auth.RequireRole("admin", "owner"))
			r.Post("/", h.OrgInvite.Create)
		})

		r.Route("/publisher-apps", func(r chi.Router) {
			// Viewer+ can list and read.
			r.With(auth.RequireRole("viewer", "editor", "admin", "owner")).Get("/", h.PublisherApp.List)
			// Editor+ can create.
			r.With(auth.RequireRole("editor", "admin", "owner")).Post("/", h.PublisherApp.Create)

			r.Route("/{id}", func(r chi.Router) {
				r.With(auth.RequireRole("viewer", "editor", "admin", "owner")).Get("/", h.PublisherApp.GetByID)
				r.With(auth.RequireRole("editor", "admin", "owner")).Put("/", h.PublisherApp.Update)

				// API Keys sub-resource.
				r.Route("/api-keys", func(r chi.Router) {
					r.With(auth.RequireRole("viewer", "editor", "admin", "owner")).Get("/", h.APIKey.List)
					r.With(auth.RequireRole("editor", "admin", "owner")).Post("/", h.APIKey.Create)
					r.With(auth.RequireRole("admin", "owner")).Delete("/{keyId}", h.APIKey.Revoke)
				})

				// Rules sub-resource.
				r.Route("/rules", func(r chi.Router) {
					r.With(auth.RequireRole("viewer", "editor", "admin", "owner")).Get("/", h.PublisherRule.List)
					r.With(auth.RequireRole("editor", "admin", "owner")).Post("/", h.PublisherRule.Create)

					r.Route("/{ruleId}", func(r chi.Router) {
						r.With(auth.RequireRole("viewer", "editor", "admin", "owner")).Get("/", h.PublisherRule.GetByID)
						r.With(auth.RequireRole("editor", "admin", "owner")).Put("/", h.PublisherRule.Update)
						r.With(auth.RequireRole("admin", "owner")).Delete("/", h.PublisherRule.Delete)
					})
				})
			})
		})
	})

	return r
}
