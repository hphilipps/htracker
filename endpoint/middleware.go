package endpoint

import (
	"context"
	"time"

	"golang.org/x/exp/slog"
)

// Middleware is a chainable behavior modifier for generic endpoints.
type Middleware[Req Requester, Resp Responder] func(Endpoint[Req, Resp]) Endpoint[Req, Resp]

// LoggingMiddleware is creating a endpoint Middleware for logging the execution duration and success status.
func LoggingMiddleware[Req Requester, Resp Responder](logger *slog.Logger) Middleware[Req, Resp] {
	return func(next Endpoint[Req, Resp]) Endpoint[Req, Resp] {
		return func(ctx context.Context, request Req) (response Resp, err error) {
			defer func(begin time.Time) {
				logger.Debug("called endpoint", slog.String("method", request.Name()),
					slog.Bool("success", response.Failed() == nil), slog.Duration("duration", time.Since(begin)))
			}(time.Now())

			return next(ctx, request)
		}
	}
}
