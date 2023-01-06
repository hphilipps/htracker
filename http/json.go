package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/endpoint"
)

type StatusCoder interface {
	StatusCode() int
}

func createJSONHandler[Req endpoint.Requester, Resp endpoint.Responder](ep endpoint.Endpoint[Req, Resp]) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		request, err := decodeHTTPJSONRequest[Req](ctx, req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			errResponse := struct{ Error string }{Error: fmt.Sprintf("request decoder: %s", err.Error())}
			if err := json.NewEncoder(w).Encode(errResponse); err != nil {
				panic(err)
			}
			return
		}

		response, err := ep(ctx, request)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			errResponse := struct{ Error string }{Error: err.Error()}
			if err := json.NewEncoder(w).Encode(errResponse); err != nil {
				panic(err)
			}
			return
		}

		if err := encodeHTTPJSONResponse(ctx, w, response); err != nil {
			panic(err)
		}
	}
}

func decodeHTTPJSONRequest[Req endpoint.Requester](_ context.Context, r *http.Request) (Req, error) {
	var req Req
	err := json.NewDecoder(r.Body).Decode(&req)
	return req, err
}

func encodeHTTPJSONResponse[Resp endpoint.Responder](ctx context.Context, w http.ResponseWriter, response Resp) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := response.Failed(); err != nil {
		switch {
		case errors.Is(err, htracker.ErrNotExist):
			w.WriteHeader(http.StatusNotFound)
		case errors.Is(err, htracker.ErrAlreadyExists):
			w.WriteHeader(http.StatusConflict)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		errResponse := struct{ Error string }{Error: err.Error()}
		return json.NewEncoder(w).Encode(errResponse)
	}

	w.WriteHeader(response.StatusCode())
	if response.StatusCode() != http.StatusNoContent {
		return json.NewEncoder(w).Encode(response)
	}
	return nil
}
