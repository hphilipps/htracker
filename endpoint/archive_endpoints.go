package endpoint

import (
	"context"

	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/service"
)

type UpdateReq struct {
	site *htracker.Site
}

func (req UpdateReq) Name() string {
	return "sitearchive_Update"
}

type UpdateResp struct {
	diff string
	err  error
}

func (resp UpdateResp) Failed() error {
	return resp.err
}

func MakeUpdateEndpoint(svc service.SiteArchive) Endpoint[UpdateReq, UpdateResp] {
	return func(ctx context.Context, req UpdateReq) (UpdateResp, error) {
		diff, err := svc.Update(ctx, req.site)
		return UpdateResp{diff: diff, err: err}, nil
	}
}

type GetReq struct {
	subscription *htracker.Subscription
}

func (req GetReq) Name() string {
	return "sitearchive_Get"
}

type GetResp struct {
	site *htracker.Site
	err  error
}

func (resp GetResp) Failed() error {
	return resp.err
}

func MakeGetEndpoint(svc service.SiteArchive) Endpoint[GetReq, GetResp] {
	return func(ctx context.Context, req GetReq) (GetResp, error) {
		site, err := svc.Get(ctx, req.subscription)
		return GetResp{site: site, err: err}, nil
	}
}
