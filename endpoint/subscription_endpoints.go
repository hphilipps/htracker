package endpoint

import (
	"context"

	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/service"
)

type AddSubscriberReq struct {
	subscriber *service.Subscriber
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

func MakeAddSubscriberEndpoint(svc service.SubscriptionSvc) Endpoint[AddSubscriberReq, AddSubscriberResp] {
	return func(ctx context.Context, req AddSubscriberReq) (AddSubscriberResp, error) {
		err := svc.AddSubscriber(ctx, req.subscriber)
		return AddSubscriberResp{err: err}, nil
	}
}

type SubscribeReq struct {
	email        string
	subscription *htracker.Subscription
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

func MakeSubscribeEndpoint(svc service.SubscriptionSvc) Endpoint[SubscribeReq, SubscribeResp] {
	return func(ctx context.Context, req SubscribeReq) (SubscribeResp, error) {
		err := svc.Subscribe(ctx, req.email, req.subscription)
		return SubscribeResp{err: err}, nil
	}
}

type GetSubscriptionsBySubscriberReq struct {
	email string
}

func (req GetSubscriptionsBySubscriberReq) Name() string {
	return "GetSubscriptionsBySubscriber"
}

type GetSubscriptionsBySubscriberResp struct {
	subscriptions []*htracker.Subscription
	err           error
}

func (resp GetSubscriptionsBySubscriberResp) Failed() error {
	return resp.err
}

func MakeGetSubscriptionsBySubscriberEndpoint(svc service.SubscriptionSvc) Endpoint[GetSubscriptionsBySubscriberReq, GetSubscriptionsBySubscriberResp] {
	return func(ctx context.Context, req GetSubscriptionsBySubscriberReq) (GetSubscriptionsBySubscriberResp, error) {
		subscriptions, err := svc.GetSubscriptionsBySubscriber(ctx, req.email)
		return GetSubscriptionsBySubscriberResp{subscriptions: subscriptions, err: err}, nil
	}
}

type GetSubscribersBySubscriptionReq struct {
	subscription *htracker.Subscription
}

func (req GetSubscribersBySubscriptionReq) Name() string {
	return "GetSubscribersBySubscription"
}

type GetSubscribersBySubscriptionResp struct {
	subscribers []*service.Subscriber
	err         error
}

func (resp GetSubscribersBySubscriptionResp) Failed() error {
	return resp.err
}

func MakeGetSubscribersBySubscriptionEndpoint(svc service.SubscriptionSvc) Endpoint[GetSubscribersBySubscriptionReq, GetSubscribersBySubscriptionResp] {
	return func(ctx context.Context, req GetSubscribersBySubscriptionReq) (GetSubscribersBySubscriptionResp, error) {
		subscribers, err := svc.GetSubscribersBySubscription(ctx, req.subscription)
		return GetSubscribersBySubscriptionResp{subscribers: subscribers, err: err}, nil
	}
}

type GetSubscribersReq struct{}

func (req GetSubscribersReq) Name() string {
	return "GetSubscribers"
}

type GetSubscribersResp struct {
	subscribers []*service.Subscriber
	err         error
}

func (resp GetSubscribersResp) Failed() error {
	return resp.err
}

func MakeGetSubscribersEndpoint(svc service.SubscriptionSvc) Endpoint[GetSubscribersReq, GetSubscribersResp] {
	return func(ctx context.Context, req GetSubscribersReq) (GetSubscribersResp, error) {
		subscribers, err := svc.GetSubscribers(ctx)
		return GetSubscribersResp{subscribers: subscribers, err: err}, nil
	}
}

type UnsubscribeReq struct {
	email        string
	subscription *htracker.Subscription
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

func MakeUnsubscribeEndpoint(svc service.SubscriptionSvc) Endpoint[UnsubscribeReq, UnsubscribeResp] {
	return func(ctx context.Context, req UnsubscribeReq) (UnsubscribeResp, error) {
		err := svc.Unsubscribe(ctx, req.email, req.subscription)
		return UnsubscribeResp{err: err}, nil
	}
}

type DeleteSubscriberReq struct {
	email string
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

func MakeDeleteSubscriberEndpoint(svc service.SubscriptionSvc) Endpoint[DeleteSubscriberReq, DeleteSubscriberResp] {
	return func(ctx context.Context, req DeleteSubscriberReq) (DeleteSubscriberResp, error) {
		err := svc.DeleteSubscriber(ctx, req.email)
		return DeleteSubscriberResp{err: err}, nil
	}
}
