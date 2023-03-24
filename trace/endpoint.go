package trace

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/opentracing/opentracing-go"

	"github.com/seniorGolang/gokit/types"
)

func TraceEndpoint(tags ...types.KeyValue) endpoint.Middleware {

	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {

			span := opentracing.SpanFromContext(ctx)

			for _, kv := range tags {
				if !kv.IsZero() {
					span.SetTag(kv.Key(), kv.Value())
				}
			}

			ctx = InjectContext(ctx, span.Context())

			return next(ctx, request)
		}
	}
}
