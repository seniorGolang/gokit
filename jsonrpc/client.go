package jsonrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/go-kit/kit/endpoint"
	httpTransport "github.com/go-kit/kit/transport/http"

	"github.com/seniorGolang/gokit/types/uuid"
)

// Client wraps a JSON RPC method and provides a method that implements endpoint.Endpoint.
type Client struct {
	client httpTransport.HTTPClient

	// JSON RPC endpoint URL
	tgt *url.URL

	// JSON RPC method name.
	method string

	enc            EncodeRequestFunc
	dec            DecodeResponseFunc
	before         []httpTransport.RequestFunc
	after          []httpTransport.ClientResponseFunc
	finalizer      httpTransport.ClientFinalizerFunc
	requestID      RequestIDGenerator
	bufferedStream bool
}

// NewClient constructs a usable Client for a single remote method.
func NewClient(
	tgt *url.URL,
	method string,
	options ...ClientOption,
) *Client {
	c := &Client{
		client:         http.DefaultClient,
		method:         method,
		tgt:            tgt,
		enc:            DefaultRequestEncoder,
		dec:            DefaultResponseDecoder,
		before:         []httpTransport.RequestFunc{},
		after:          []httpTransport.ClientResponseFunc{},
		requestID:      NewUUIDGenerator(),
		bufferedStream: false,
	}
	for _, option := range options {
		option(c)
	}
	return c
}

// DefaultRequestEncoder marshals the given request to JSON.
func DefaultRequestEncoder(_ context.Context, req interface{}) (json.RawMessage, error) {
	return json.Marshal(req)
}

// DefaultResponseDecoder unmarshals the result to interface{}, or returns an
// error, if found.
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

// ClientOption sets an optional parameter for clients.
type ClientOption func(*Client)

// SetClient sets the underlying HTTP client used for requests.
// By default, http.DefaultClient is used.
func SetClient(client httpTransport.HTTPClient) ClientOption {
	return func(c *Client) { c.client = client }
}

// ClientBefore sets the RequestFuncs that are applied to the outgoing HTTP
// request before it's invoked.
func ClientBefore(before ...httpTransport.RequestFunc) ClientOption {
	return func(c *Client) { c.before = append(c.before, before...) }
}

// ClientAfter sets the ClientResponseFuncs applied to the server's HTTP
// response prior to it being decoded. This is useful for obtaining anything
// from the response and adding onto the context prior to decoding.
func ClientAfter(after ...httpTransport.ClientResponseFunc) ClientOption {
	return func(c *Client) { c.after = append(c.after, after...) }
}

// ClientFinalizer is executed at the end of every HTTP request.
// By default, no finalizer is registered.
func ClientFinalizer(f httpTransport.ClientFinalizerFunc) ClientOption {
	return func(c *Client) { c.finalizer = f }
}

// ClientRequestEncoder sets the func used to encode the request params to JSON.
// If not set, DefaultRequestEncoder is used.
func ClientRequestEncoder(enc EncodeRequestFunc) ClientOption {
	return func(c *Client) { c.enc = enc }
}

// ClientResponseDecoder sets the func used to decode the response params from
// JSON. If not set, DefaultResponseDecoder is used.
func ClientResponseDecoder(dec DecodeResponseFunc) ClientOption {
	return func(c *Client) { c.dec = dec }
}

// RequestIDGenerator returns an ID for the request.
type RequestIDGenerator interface {
	Generate() *RequestID
}

// ClientRequestIDGenerator is executed before each request to generate an ID
// for the request.
// By default, AutoIncrementRequestID is used.
func ClientRequestIDGenerator(g RequestIDGenerator) ClientOption {
	return func(c *Client) { c.requestID = g }
}

// BufferedStream sets whether the Response.Body is left open, allowing it
// to be read from later. Useful for transporting a file as a buffered stream.
func BufferedStream(buffered bool) ClientOption {
	return func(c *Client) { c.bufferedStream = buffered }
}

// Endpoint returns a usable endpoint that invokes the remote endpoint.
func (c Client) Endpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		var (
			resp *http.Response
			err  error
		)
		if c.finalizer != nil {
			defer func() {
				if resp != nil {
					ctx = context.WithValue(ctx, httpTransport.ContextKeyResponseHeaders, resp.Header)
					ctx = context.WithValue(ctx, httpTransport.ContextKeyResponseSize, resp.ContentLength)
				}
				c.finalizer(ctx, err)
			}()
		}

		var params json.RawMessage
		if params, err = c.enc(ctx, request); err != nil {
			return nil, err
		}
		rpcReq := Request{
			JSONRPC: "",
			Method:  c.method,
			Params:  params,
			ID:      c.requestID.Generate(),
		}

		req, err := http.NewRequest("POST", c.tgt.String()+"/"+c.method, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		var b bytes.Buffer
		req.Body = ioutil.NopCloser(&b)
		err = json.NewEncoder(&b).Encode(rpcReq)
		if err != nil {
			return nil, err
		}

		for _, f := range c.before {
			ctx = f(ctx, req)
		}

		resp, err = c.client.Do(req.WithContext(ctx))
		if err != nil {
			return nil, err
		}

		if !c.bufferedStream {
			defer resp.Body.Close()
		}

		// Decode the body into an object
		var rpcRes Response
		err = json.NewDecoder(resp.Body).Decode(&rpcRes)
		if err != nil {
			return nil, err
		}

		for _, f := range c.after {
			ctx = f(ctx, resp)
		}

		return c.dec(ctx, rpcRes)
	}
}

// SetAuthorizaedHeader returns a RequestFunc that set Authorization header
func SetAuthorizationHeader(token string) httpTransport.RequestFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		var bearer = "Bearer " + token
		r.Header.Set(http.CanonicalHeaderKey("Authorization"), bearer)
		return ctx
	}
}

// SetUserIDHeader returns a RequestFunc that set X-User-Id header
func SetUserIDHeader(userID uuid.UUID) httpTransport.RequestFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		r.Header.Set(http.CanonicalHeaderKey("X-User-Id"), userID.String())
		return ctx
	}
}

// SetAuthorizedHeader returns a RequestFunc that sets Authorization header
func SetSessionIDHeader(sessionID uuid.UUID) httpTransport.RequestFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		r.Header.Set(http.CanonicalHeaderKey("X-Session-Id"), sessionID.String())
		return ctx
	}
}

// autoUUIDGeneratorID is a RequestIDGenerator that generates
// new uuid v4
type autoUUIDGeneratorID struct{}

// NewUUIDGenerator returns an auto-incrementing request ID generator,
// initialised with the given value.
func NewUUIDGenerator() RequestIDGenerator {
	return &autoUUIDGeneratorID{}
}

// Generate satisfies RequestIDGenerator
func (i *autoUUIDGeneratorID) Generate() *RequestID {
	return &RequestID{
		stringValue: uuid.NewV4().String(),
	}
}
