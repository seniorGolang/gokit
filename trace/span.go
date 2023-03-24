package trace

import (
	"net/http"

	"github.com/opentracing/opentracing-go"
)

func MakeSpan(methodName string, headers map[string]string) (span opentracing.Span) {

	spanContext, err := opentracing.GlobalTracer().Extract(opentracing.TextMap, opentracing.TextMapCarrier(headers))

	if err == nil {
		span = opentracing.StartSpan(methodName, opentracing.ChildOf(spanContext))
	} else {
		span = opentracing.GlobalTracer().StartSpan(methodName)
	}
	return
}

func SpanFromHttp(methodName string, r *http.Request) (span opentracing.Span) {

	spanContext, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))

	if err == nil {
		span = opentracing.StartSpan(methodName, opentracing.ChildOf(spanContext))
	} else {
		span = opentracing.GlobalTracer().StartSpan(methodName)
	}
	return
}
