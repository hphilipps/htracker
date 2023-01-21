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
	router.Get("/api/site", createJSONHandler(archiveEndpoints.Get))
	router.Post("/api/subscriber", createJSONHandler(subscriptionEndpoints.AddSubscriber))
	router.Get("/api/subscriber", createJSONHandler(subscriptionEndpoints.GetSubscribers))
	router.Get("/api/subscriber/by_subscription", createJSONHandler(subscriptionEndpoints.GetSubscribersBySubscription))
	router.Delete("/api/subscriber", createJSONHandler(subscriptionEndpoints.DeleteSubscriber))
	router.Post("/api/subscription", createJSONHandler(subscriptionEndpoints.Subscribe))
	router.Get("/api/subscription/by_subscriber", createJSONHandler(subscriptionEndpoints.GetSubscriptionsBySubscriber))
	router.Delete("/api/subscription", createJSONHandler(subscriptionEndpoints.Unsubscribe))

	return router
}
