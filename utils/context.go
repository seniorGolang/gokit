package utils

import (
	"context"
	"net/http"
	"strings"

	"github.com/openzipkin/zipkin-go/propagation/b3"

	"github.com/seniorGolang/gokit/types/uuid"
)

type contextKey string

const (
	ownerKey   = "x-user-id"
	headerKeys = contextKey("headerKeys")
)

func GetOwnerId(ctx context.Context) (id string) {
	id, _ = ctx.Value(ownerKey).(string)
	return
}

func GetOwnerUUID(ctx context.Context) (id uuid.UUID, err error) {
	return uuid.FromString(GetOwnerId(ctx))
}

func AddHeadersToContext(ctx context.Context, headers map[string]interface{}) context.Context {

	for key, value := range headers {
		ctx = AddHeaderToContext(ctx, key, value)
	}
	return ctx
}

func AddHeaderToContext(ctx context.Context, key string, value interface{}) context.Context {

	if value == nil {
		return ctx
	}

	var keys []string

	if ctxKeys := ctx.Value(headerKeys); ctxKeys != nil {
		keys = ctxKeys.([]string)
	}

	var found bool
	for _, k := range keys {
		if found = k == key; found {
			break
		}
	}

	if !found {
		keys = append(keys, key)
	}

	ctx = context.WithValue(ctx, key, value)
	ctx = context.WithValue(ctx, headerKeys, keys)
	return ctx
}

func HeadersFromContext(ctx context.Context) (headers map[string]interface{}) {

	headers = make(map[string]interface{})

	keys := headersKeys

	if ctxKeys := ctx.Value(headerKeys); ctxKeys != nil {
		keys = ctxKeys.([]string)
	}

	for _, key := range keys {
		if ctx.Value(key) != nil {
			headers[key] = ctx.Value(key)
		}
	}
	return
}

func HeadersToContext(ctx context.Context, headers map[string]interface{}) context.Context {

	var keys []string

	for key, value := range headers {

		if value != nil {
			keys = append(keys, key)
			ctx = context.WithValue(ctx, key, value)
		}
	}
	return context.WithValue(ctx, headerKeys, keys)
}

func HttpToContext(ctx context.Context, r *http.Request) context.Context {

	var keys []string

	for key := range r.Header {

		value := r.Header.Get(key)

		if value != "" {
			key = strings.ToLower(key)
			keys = append(keys, key)
			ctx = context.WithValue(ctx, key, r.Header.Get(key))
		}
	}
	return context.WithValue(ctx, headerKeys, keys)
}

var headersKeys = []string{
	"x-user-id",
	"x-trace-id",
	"user-agent",
	"x-session-id",
	"authorization",
	"x-requested-with",
	"x-authorization-provider",
	b3.TraceID,
	b3.SpanID,
	b3.ParentSpanID,
	b3.Sampled,
	b3.Flags,
	b3.Context,
}
