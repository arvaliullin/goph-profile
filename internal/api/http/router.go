package httpserver

import (
	"log/slog"
	"net/http"

	"github.com/arvaliullin/goph-profile/internal/api/http/handlers"
	"github.com/arvaliullin/goph-profile/internal/api/http/middleware"
	"github.com/arvaliullin/goph-profile/internal/observability"
	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// @title GophProfile API
// @version 1.0
// @description REST API сервиса профилей и аватаров (загрузка, выдача изображений и метаданных).
// @host localhost:8080
// @BasePath /
// @securityDefinitions.apikey UserIDAuth
// @in header
// @name X-User-ID

// Deps зависимости HTTP API.
type Deps struct {
	Log     *slog.Logger
	Service string
	Avatar  *handlers.AvatarHTTP
	Health  *handlers.Health
}

// NewRouter собирает chi mux.
func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Handle("/metrics", observability.MetricsHandler())
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequestLogger(d.Log, d.Service))
		r.Get("/health", d.Health.Handle)
		r.Get("/swagger/*", httpSwagger.Handler(
			httpSwagger.URL("/swagger/doc.json"),
		))
		r.Route("/api/v1", func(r chi.Router) {
			r.Use(middleware.UserIDFromHeader)
			r.Post("/avatars", d.Avatar.Upload)
			r.Get("/avatars/{avatarID}", d.Avatar.GetImage)
			r.Get("/avatars/{avatarID}/metadata", d.Avatar.Metadata)
			r.Delete("/avatars/{avatarID}", d.Avatar.DeleteAvatar)
			r.Get("/users/{userID}/avatar", d.Avatar.UserAvatar)
			r.Get("/users/{userID}/avatars", d.Avatar.UserAvatars)
			r.Delete("/users/{userID}/avatar", d.Avatar.DeleteUserAvatar)
		})
	})

	return r
}
