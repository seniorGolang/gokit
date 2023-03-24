package trace

import (
	"context"

	"github.com/opentracing/opentracing-go"

	"github.com/seniorGolang/gokit/utils"
)

func InjectSpan(ctx context.Context) context.Context {

	span := opentracing.SpanFromContext(ctx)

	if span == nil {
		return ctx
	}

	headers := make(opentracing.TextMapCarrier)

	_ = opentracing.GlobalTracer().Inject(span.Context(), opentracing.TextMap, headers)

	for k, v := range headers {
		ctx = utils.AddHeaderToContext(ctx, k, v)
	}
	return ctx
}

func InjectContext(ctx context.Context, sc opentracing.SpanContext) context.Context {

	headers := make(opentracing.TextMapCarrier)

	_ = opentracing.GlobalTracer().Inject(sc, opentracing.TextMap, headers)

	for k, v := range headers {
		ctx = utils.AddHeaderToContext(ctx, k, v)
	}
	return ctx
}
