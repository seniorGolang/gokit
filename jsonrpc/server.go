package jsonrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"

	httpTransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"

	"github.com/seniorGolang/gokit/logger"
)

const reqID = "requestID"
var log = logger.Log.WithField("module", "httpServer")

// Server wraps an endpoint and implements http.Handler.
type Server struct {
	ecm          EndpointCodecMap
	errorEncoder httpTransport.ErrorEncoder
	before       []httpTransport.RequestFunc
	finalizer    httpTransport.ServerFinalizerFunc
	after        []httpTransport.ServerResponseFunc
}

// NewServer constructs a new server, which implements http.Server.
func NewServer(ecm EndpointCodecMap, options ...ServerOption) *Server {
	s := &Server{
		ecm:          ecm,
		errorEncoder: DefaultErrorEncoder,
	}
	for _, option := range options {
		option(s)
	}
	return s
}

// ServerOption sets an optional parameter for servers.
type ServerOption func(*Server)

// ServerBefore functions are executed on the HTTP request object before the
// request is decoded.
func ServerBefore(before ...httpTransport.RequestFunc) ServerOption {
	return func(s *Server) { s.before = append(s.before, before...) }
}

// ServerAfter functions are executed on the HTTP response writer after the
// endpoint is invoked, but before anything is written to the client.
func ServerAfter(after ...httpTransport.ServerResponseFunc) ServerOption {
	return func(s *Server) { s.after = append(s.after, after...) }
}

// ServerErrorEncoder is used to encode errors to the http.ResponseWriter
// whenever they're encountered in the processing of a request. Clients can
// use this to provide custom error formatting and response codes. By default,
// errors will be written with the DefaultErrorEncoder.
func ServerErrorEncoder(ee httpTransport.ErrorEncoder) ServerOption {
	return func(s *Server) { s.errorEncoder = ee }
}

// ServeHTTP implements http.Handler.
func (s Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = io.WriteString(w, "405 must POST\n")
		return
	}

	ctx := r.Context()

	for _, f := range s.before {
		ctx = f(ctx, r)
	}

	bodyData, err := ioutil.ReadAll(r.Body)

	if err != nil {
		rpcErr := parseError("read body error: " + err.Error())
		s.errorEncoder(ctx, rpcErr, w)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var reqList []Request

	if err = json.Unmarshal(bodyData, &reqList); err != nil {

		var req Request
		if err = json.Unmarshal(bodyData, &req); err != nil {
			rpcErr := parseError("request body could not be decoded: " + err.Error())
			s.errorEncoder(ctx, rpcErr, w)
			w.WriteHeader(http.StatusBadRequest)
			return
		} else {
			reqList = append(reqList, req)
		}
	}

	var wg sync.WaitGroup
	var respList []Response

	urlMethod, _ := mux.Vars(r)["method"]

	for _, req := range reqList {

		if urlMethod != "" && req.Method != "" && req.Method != urlMethod {

			if req.ID != nil {
				respList = append(respList, Response{
					ID:      req.ID,
					JSONRPC: Version,
					Error: &Error{
						Code:    MethodNotFoundError,
						Message: fmt.Sprintf("incorrect method: %s != %s", urlMethod, req.Method),
					},
				})
			}
			continue
		}

		if urlMethod != "" {
			req.Method = urlMethod
		}
		ecm, ok := s.ecm[req.Method]

		if ! ok {
			if req.ID != nil {
				respList = append(respList, Response{
					ID:      req.ID,
					JSONRPC: Version,
					Error: &Error{
						Code:    MethodNotFoundError,
						Message: fmt.Sprintf("method %s not found", req.Method),
					},
				})
			}
			continue
		}

		ctx = context.WithValue(ctx, reqID, req.ID)
		reqParams, err := ecm.Decode(ctx, req.Params)

		if err != nil {
			if req.ID != nil {
				respList = append(respList, Response{
					ID:      req.ID,
					JSONRPC: Version,
					Error: &Error{
						Code:    InvalidParamsError,
						Message: fmt.Sprintf("decode params error: %s", err.Error()),
					},
				})
			}
			continue
		}

		wg.Add(1)

		go func(req Request) {

			defer wg.Done()

			response, err := ecm.Endpoint(ctx, reqParams)

			if err != nil {

				if req.ID != nil {
					respList = append(respList, Response{
						ID:      req.ID,
						JSONRPC: Version,
						Error: &Error{
							Data:    err,
							Message: err.Error(),
							Code:    InternalError,
						},
					})
				}
				return
			}

			if req.ID != nil {

				result, err := ecm.Encode(ctx, response)

				if err != nil {
					if req.ID != nil {
						respList = append(respList, Response{
							ID:      req.ID,
							JSONRPC: Version,
							Error: &Error{
								Code:    InternalError,
								Message: fmt.Sprintf("response encode error: %s", err.Error()),
							},
						})
					}
					return
				}

				respList = append(respList, Response{
					ID:      req.ID,
					JSONRPC: Version,
					Result:  result,
				})
			}
		}(req)
	}

	wg.Wait()

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", ContentType)

	if len(respList) == 1 {
		if err := json.NewEncoder(w).Encode(respList[0]); err != nil {
			log.WithError(err).Error("encode error")
			return
		}
	} else if len(respList) > 1 {
		if err := json.NewEncoder(w).Encode(respList); err != nil {
			log.WithError(err).Error("encode error")
			return
		}
	}

	for _, f := range s.after {
		ctx = f(ctx, w)
	}
}

// DefaultErrorEncoder writes the error to the ResponseWriter,
// as a json-rpc error response, with an InternalError status code.
// The Error() string of the error will be used as the response error message.
// If the error implements ErrorCoder, the provided code will be set on the
// response error.
// If the error implements Headerer, the given headers will be set.
func DefaultErrorEncoder(ctx context.Context, err error, w http.ResponseWriter) {

	w.Header().Set("Content-Type", ContentType)

	if headerer, ok := err.(httpTransport.Headerer); ok {
		for k := range headerer.Headers() {
			w.Header().Set(k, headerer.Headers().Get(k))
		}
	}

	e := Error{
		Code:    InternalError,
		Message: err.Error(),
	}

	if sc, ok := err.(ErrorCoder); ok {
		e.Code = sc.ErrorCode()
	}

	reqID, _ := ctx.Value(reqID).(*RequestID)

	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(Response{
		JSONRPC: Version,
		Error:   &e,
		ID:      reqID,
	})
}

// ErrorCoder is checked by DefaultErrorEncoder. If an error value implements
// ErrorCoder, the integer result of ErrorCode() will be used as the JSONRPC
// error code when encoding the error.
//
// By default, InternalError (-32603) is used.
type ErrorCoder interface {
	ErrorCode() int
}

// interceptingWriter intercepts calls to WriteHeader, so that a finalizer
// can be given the correct status code.
type interceptingWriter struct {
	http.ResponseWriter
	code int
}

// WriteHeader may not be explicitly called, so care must be taken to
// initialize w.code to its default value of http.StatusOK.
func (w *interceptingWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}
