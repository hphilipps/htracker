package endpoint

import (
	"context"
	"fmt"
	"net/http"

	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/service"
	"golang.org/x/exp/slog"
)

type SubscriptionEndpoints struct {
	AddSubscriber                Endpoint[AddSubscriberReq, AddSubscriberResp]
	Subscribe                    Endpoint[SubscribeReq, SubscribeResp]
	GetSubscriptionsBySubscriber Endpoint[GetSubscriptionsBySubscriberReq, GetSubscriptionsBySubscriberResp]
	GetSubscribersBySubscription Endpoint[GetSubscribersBySubscriptionReq, GetSubscribersBySubscriptionResp]
	GetSubscribers               Endpoint[GetSubscribersReq, GetSubscribersResp]
	Unsubscribe                  Endpoint[UnsubscribeReq, UnsubscribeResp]
	DeleteSubscriber             Endpoint[DeleteSubscriberReq, DeleteSubscriberResp]
}

func MakeSubscriptionEndpoints(svc service.SubscriptionSvc, logger *slog.Logger) SubscriptionEndpoints {
	addSubscriberEP := MakeAddSubscriberEndpoint(svc)
	addSubscriberEP = LoggingMiddleware[AddSubscriberReq, AddSubscriberResp](logger)(addSubscriberEP)

	subscribeEP := MakeSubscribeEndpoint(svc)
	subscribeEP = LoggingMiddleware[SubscribeReq, SubscribeResp](logger)(subscribeEP)

	getSubscriptionsBySubscriberEP := MakeGetSubscriptionsBySubscriberEndpoint(svc)
	getSubscriptionsBySubscriberEP = LoggingMiddleware[GetSubscriptionsBySubscriberReq, GetSubscriptionsBySubscriberResp](logger)(getSubscriptionsBySubscriberEP)

	getSubscribersBySubscriptionEP := MakeGetSubscribersBySubscriptionEndpoint(svc)
	getSubscribersBySubscriptionEP = LoggingMiddleware[GetSubscribersBySubscriptionReq, GetSubscribersBySubscriptionResp](logger)(getSubscribersBySubscriptionEP)

	getSubscibersEP := MakeGetSubscribersEndpoint(svc)
	getSubscibersEP = LoggingMiddleware[GetSubscribersReq, GetSubscribersResp](logger)(getSubscibersEP)

	unsubscribeEP := MakeUnsubscribeEndpoint(svc)
	unsubscribeEP = LoggingMiddleware[UnsubscribeReq, UnsubscribeResp](logger)(unsubscribeEP)

	deleteEP := MakeDeleteSubscriberEndpoint(svc)
	deleteEP = LoggingMiddleware[DeleteSubscriberReq, DeleteSubscriberResp](logger)(deleteEP)

	return SubscriptionEndpoints{
		AddSubscriber:                addSubscriberEP,
		Subscribe:                    subscribeEP,
		GetSubscriptionsBySubscriber: getSubscriptionsBySubscriberEP,
		GetSubscribersBySubscription: getSubscribersBySubscriptionEP,
		GetSubscribers:               getSubscibersEP,
		Unsubscribe:                  unsubscribeEP,
		DeleteSubscriber:             deleteEP,
	}
}

type AddSubscriberReq struct {
	Subscriber *service.Subscriber
}

func (req AddSubscriberReq) Name() string {
	return "AddSubscriber"
}

type AddSubscriberResp struct {
	err error
}

func (resp AddSubscriberResp) Failed() error {
	return resp.err
}

func (resp AddSubscriberResp) StatusCode() int {
	return http.StatusNoContent
}

func MakeAddSubscriberEndpoint(svc service.SubscriptionSvc) Endpoint[AddSubscriberReq, AddSubscriberResp] {
	return func(ctx context.Context, req AddSubscriberReq) (AddSubscriberResp, error) {
		if req.Subscriber == nil {
			return AddSubscriberResp{}, fmt.Errorf("could not find valid subscriber in request")
		}
		err := svc.AddSubscriber(ctx, req.Subscriber)
		return AddSubscriberResp{err: err}, nil
	}
}

type SubscribeReq struct {
	Email        string
	Subscription *htracker.Subscription
}

func (req SubscribeReq) Name() string {
	return "Subscribe"
}

type SubscribeResp struct {
	err error
}

func (resp SubscribeResp) Failed() error {
	return resp.err
}

func (resp SubscribeResp) StatusCode() int {
	return http.StatusNoContent
}

func MakeSubscribeEndpoint(svc service.SubscriptionSvc) Endpoint[SubscribeReq, SubscribeResp] {
	return func(ctx context.Context, req SubscribeReq) (SubscribeResp, error) {
		if req.Subscription == nil {
			return SubscribeResp{}, fmt.Errorf("could not find valid subscription in request")
		}
		err := svc.Subscribe(ctx, req.Email, req.Subscription)
		return SubscribeResp{err: err}, nil
	}
}

type GetSubscriptionsBySubscriberReq struct {
	Email string
}

func (req GetSubscriptionsBySubscriberReq) Name() string {
	return "GetSubscriptionsBySubscriber"
}

type GetSubscriptionsBySubscriberResp struct {
	Subscriptions []*htracker.Subscription
	err           error
}

func (resp GetSubscriptionsBySubscriberResp) Failed() error {
	return resp.err
}

func (resp GetSubscriptionsBySubscriberResp) StatusCode() int {
	return http.StatusOK
}

func MakeGetSubscriptionsBySubscriberEndpoint(svc service.SubscriptionSvc) Endpoint[GetSubscriptionsBySubscriberReq, GetSubscriptionsBySubscriberResp] {
	return func(ctx context.Context, req GetSubscriptionsBySubscriberReq) (GetSubscriptionsBySubscriberResp, error) {
		subscriptions, err := svc.GetSubscriptionsBySubscriber(ctx, req.Email)
		return GetSubscriptionsBySubscriberResp{Subscriptions: subscriptions, err: err}, nil
	}
}

type GetSubscribersBySubscriptionReq struct {
	Subscription *htracker.Subscription
}

func (req GetSubscribersBySubscriptionReq) Name() string {
	return "GetSubscribersBySubscription"
}

type GetSubscribersBySubscriptionResp struct {
	Subscribers []*service.Subscriber
	err         error
}

func (resp GetSubscribersBySubscriptionResp) Failed() error {
	return resp.err
}

func (resp GetSubscribersBySubscriptionResp) StatusCode() int {
	return http.StatusOK
}

func MakeGetSubscribersBySubscriptionEndpoint(svc service.SubscriptionSvc) Endpoint[GetSubscribersBySubscriptionReq, GetSubscribersBySubscriptionResp] {
	return func(ctx context.Context, req GetSubscribersBySubscriptionReq) (GetSubscribersBySubscriptionResp, error) {
		if req.Subscription == nil {
			return GetSubscribersBySubscriptionResp{}, fmt.Errorf("could not find valid subscription in request")
		}
		subscribers, err := svc.GetSubscribersBySubscription(ctx, req.Subscription)
		return GetSubscribersBySubscriptionResp{Subscribers: subscribers, err: err}, nil
	}
}

type GetSubscribersReq struct{}

func (req GetSubscribersReq) Name() string {
	return "GetSubscribers"
}

type GetSubscribersResp struct {
	Subscribers []*service.Subscriber
	err         error
}

func (resp GetSubscribersResp) Failed() error {
	return resp.err
}

func (resp GetSubscribersResp) StatusCode() int {
	return http.StatusOK
}

func MakeGetSubscribersEndpoint(svc service.SubscriptionSvc) Endpoint[GetSubscribersReq, GetSubscribersResp] {
	return func(ctx context.Context, req GetSubscribersReq) (GetSubscribersResp, error) {
		subscribers, err := svc.GetSubscribers(ctx)
		return GetSubscribersResp{Subscribers: subscribers, err: err}, nil
	}
}

type UnsubscribeReq struct {
	Email        string
	Subscription *htracker.Subscription
}

func (req UnsubscribeReq) Name() string {
	return "Unsubscribe"
}

type UnsubscribeResp struct {
	err error
}

func (resp UnsubscribeResp) Failed() error {
	return resp.err
}

func (resp UnsubscribeResp) StatusCode() int {
	return http.StatusNoContent
}

func MakeUnsubscribeEndpoint(svc service.SubscriptionSvc) Endpoint[UnsubscribeReq, UnsubscribeResp] {
	return func(ctx context.Context, req UnsubscribeReq) (UnsubscribeResp, error) {
		if req.Subscription == nil {
			return UnsubscribeResp{}, fmt.Errorf("could not find valid subscription in request")
		}
		err := svc.Unsubscribe(ctx, req.Email, req.Subscription)
		return UnsubscribeResp{err: err}, nil
	}
}

type DeleteSubscriberReq struct {
	Email string
}

func (req DeleteSubscriberReq) Name() string {
	return "DeleteSubscriber"
}

type DeleteSubscriberResp struct {
	err error
}

func (resp DeleteSubscriberResp) Failed() error {
	return resp.err
}

func (resp DeleteSubscriberResp) StatusCode() int {
	return http.StatusNoContent
}

func MakeDeleteSubscriberEndpoint(svc service.SubscriptionSvc) Endpoint[DeleteSubscriberReq, DeleteSubscriberResp] {
	return func(ctx context.Context, req DeleteSubscriberReq) (DeleteSubscriberResp, error) {
		err := svc.DeleteSubscriber(ctx, req.Email)
		return DeleteSubscriberResp{err: err}, nil
	}
}
