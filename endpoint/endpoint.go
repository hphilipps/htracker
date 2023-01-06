package endpoint

import (
	"context"
)

// Endpoint is the fundamental building block of servers and clients.
// It represents a single RPC method.
type Endpoint[Req Requester, Resp Responder] func(ctx context.Context, request Req) (response Resp, err error)

// Requester is an interface helpful to enrich endpoint middlewares with information.
type Requester interface {
	Name() string
}

// Responder is an interface helpful for encoders and to enrich endpoint middlewares with information.
type Responder interface {
	Failed() error
	StatusCode() int
}
