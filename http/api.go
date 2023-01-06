package http

import (
	"github.com/go-chi/chi"
	"gitlab.com/henri.philipps/htracker/endpoint"
	"gitlab.com/henri.philipps/htracker/service"
	"golang.org/x/exp/slog"
)

func MakeAPIHandler(archivesvc service.SiteArchive, subcriptionsvc service.SubscriptionSvc, logger *slog.Logger) *chi.Mux {
	archiveEndpoints := endpoint.MakeArchiveEndpoints(archivesvc, logger)
	subscriptionEndpoints := endpoint.MakeSubscriptionEndpoints(subcriptionsvc, logger)

	router := chi.NewRouter()
	router.Get("/api/get", createJSONHandler(archiveEndpoints.Get))
	router.Post("/api/subscriber", createJSONHandler(subscriptionEndpoints.AddSubscriber))
	router.Post("/api/subscribe", createJSONHandler(subscriptionEndpoints.Subscribe))

	return router
}
