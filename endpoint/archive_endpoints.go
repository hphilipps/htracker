package endpoint

import (
	"context"
	"fmt"
	"net/http"

	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/service"
	"golang.org/x/exp/slog"
)

type ArchiveEndpoints struct {
	Update Endpoint[UpdateReq, UpdateResp]
	Get    Endpoint[GetReq, GetResp]
}

func MakeArchiveEndpoints(svc service.SiteArchive, logger *slog.Logger) ArchiveEndpoints {

	updateEP := MakeUpdateEndpoint(svc)
	updateEP = LoggingMiddleware[UpdateReq, UpdateResp](logger)(updateEP)

	getEP := MakeGetEndpoint(svc)
	getEP = LoggingMiddleware[GetReq, GetResp](logger)(getEP)

	return ArchiveEndpoints{
		Update: updateEP,
		Get:    getEP,
	}
}

type UpdateReq struct {
	Site *htracker.Site
}

func (req UpdateReq) Name() string {
	return "sitearchive_Update"
}

type UpdateResp struct {
	Diff string
	err  error
}

func (resp UpdateResp) Failed() error {
	return resp.err
}

func (resp UpdateResp) StatusCode() int {
	return http.StatusNoContent
}

func MakeUpdateEndpoint(svc service.SiteArchive) Endpoint[UpdateReq, UpdateResp] {
	return func(ctx context.Context, req UpdateReq) (UpdateResp, error) {
		if req.Site == nil {
			return UpdateResp{}, fmt.Errorf("could not find site in request")
		}
		diff, err := svc.Update(ctx, req.Site)
		return UpdateResp{Diff: diff, err: err}, nil
	}
}

type GetReq struct {
	Subscription *htracker.Subscription
}

func (req GetReq) Name() string {
	return "sitearchive_Get"
}

type GetResp struct {
	Site *htracker.Site
	err  error
}

func (resp GetResp) Failed() error {
	return resp.err
}

func (resp GetResp) StatusCode() int {
	return http.StatusOK
}

func MakeGetEndpoint(svc service.SiteArchive) Endpoint[GetReq, GetResp] {
	return func(ctx context.Context, req GetReq) (GetResp, error) {
		if req.Subscription == nil {
			return GetResp{}, fmt.Errorf("could not find subscription in request")
		}
		site, err := svc.Get(ctx, req.Subscription)
		return GetResp{Site: site, err: err}, nil
	}
}
