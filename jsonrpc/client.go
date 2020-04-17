package jsonrpc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/go-kit/kit/endpoint"
	"github.com/valyala/fasthttp"

	"github.com/seniorGolang/gokit/types/uuid"
)

type DecodeResponseError func(context.Context, json.RawMessage) (err error)
type ClientRequestFunc func(context.Context, *fasthttp.Request) context.Context
type ClientResponseFunc func(context.Context, *fasthttp.Response) context.Context

type Client struct {
	method string
	tgtURL *url.URL
	client fasthttp.Client

	enc        EncodeRequestFunc
	dec        DecodeResponseFunc
	before     []ClientRequestFunc
	after      []ClientResponseFunc
	errDecoder DecodeResponseError
	requestID  RequestIDGenerator
}

func NewClient(uri, method string, options ...ClientOption) *Client {

	tgtURL, _ := url.Parse(uri)

	c := &Client{
		method:    method,
		tgtURL:    tgtURL,
		client:    fasthttp.Client{},
		requestID: NewUUIDGenerator(),
		enc:       DefaultRequestEncoder,
		dec:       DefaultResponseDecoder,
	}

	for _, option := range options {
		option(c)
	}
	return c
}

func DefaultRequestEncoder(_ context.Context, req interface{}) (json.RawMessage, error) {
	return json.Marshal(req)
}

func DefaultResponseDecoder(_ context.Context, res Response) (interface{}, error) {
	if res.Error != nil {
		return nil, *res.Error
	}
	var result interface{}
	err := json.Unmarshal(res.Result, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type ClientOption func(*Client)

func (c Client) Endpoint() endpoint.Endpoint {

	return func(ctx context.Context, request interface{}) (result interface{}, err error) {

		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		defer cancel()

		req := fasthttp.AcquireRequest()
		resp := fasthttp.AcquireResponse()

		var params json.RawMessage
		if params, err = c.enc(ctx, request); err != nil {
			return nil, err
		}

		rpcReq := Request{
			JSONRPC: Version,
			Params:  params,
			Method:  c.method,
			ID:      c.requestID.Generate(),
		}

		req.Header.SetMethod("POST")
		req.SetRequestURI(c.tgtURL.String() + "/" + c.method)
		req.Header.Set("Content-Type", "application/json; charset=utf-8")

		if err = json.NewEncoder(req.BodyWriter()).Encode(rpcReq); err != nil {
			return
		}

		for _, f := range c.before {
			ctx = f(ctx, req)
		}

		if err = c.client.Do(req, resp); err != nil {
			return
		}

		// Decode the body into an object
		var rpcResRaw ResponseRaw
		if err = json.Unmarshal(resp.Body(), &rpcResRaw); err != nil {
			return
		}

		if rpcResRaw.Error != nil && c.errDecoder != nil {
			if err = c.errDecoder(ctx, rpcResRaw.Error); err != nil {
				return
			}
		}

		for _, f := range c.after {
			ctx = f(ctx, resp)
		}

		rpcRes := Response{
			ID:      rpcResRaw.ID,
			Result:  rpcResRaw.Result,
			JSONRPC: rpcResRaw.JSONRPC,
		}

		return c.dec(ctx, rpcRes)
	}
}

func SetAuthorizationHeader(token string) ClientRequestFunc {
	return func(ctx context.Context, r *fasthttp.Request) context.Context {
		var bearer = "Bearer " + token
		r.Header.Set(http.CanonicalHeaderKey("authorization"), bearer)
		return ctx
	}
}

func SetUserIDHeader(userID uuid.UUID) ClientRequestFunc {
	return func(ctx context.Context, r *fasthttp.Request) context.Context {
		r.Header.Set(http.CanonicalHeaderKey("x-user-id"), userID.String())
		return ctx
	}
}

func SetSessionIDHeader(sessionID uuid.UUID) ClientRequestFunc {
	return func(ctx context.Context, r *fasthttp.Request) context.Context {
		r.Header.Set(http.CanonicalHeaderKey("x-session-id"), sessionID.String())
		return ctx
	}
}

func SetClient(client fasthttp.Client) ClientOption {
	return func(c *Client) { c.client = client }
}

func ClientBefore(before ...ClientRequestFunc) ClientOption {
	return func(c *Client) { c.before = append(c.before, before...) }
}

func ClientAfter(after ...ClientResponseFunc) ClientOption {
	return func(c *Client) { c.after = append(c.after, after...) }
}

func ClientRequestEncoder(enc EncodeRequestFunc) ClientOption {
	return func(c *Client) { c.enc = enc }
}

func ClientResponseErrorDecoder(enc DecodeResponseError) ClientOption {
	return func(c *Client) { c.errDecoder = enc }
}

func ClientResponseDecoder(dec DecodeResponseFunc) ClientOption {
	return func(c *Client) { c.dec = dec }
}

type RequestIDGenerator interface {
	Generate() *RequestID
}

func ClientRequestIDGenerator(g RequestIDGenerator) ClientOption {
	return func(c *Client) { c.requestID = g }
}
