package wkratos

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/metadata"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/middleware/validate"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/toeydevelopment/telltheworld/pkg/healthcheck"
	"go.opentelemetry.io/otel/sdk/trace"
	otrace "go.opentelemetry.io/otel/trace"
)

type HTTPOption struct {
	Network    string
	Addr       string
	Timeout    time.Duration
	Middleware []middleware.Middleware
}

func NewHTTPServer(
	option HTTPOption,
	tp *trace.TracerProvider,
	logger log.Logger,
	fn func(srv *http.Server),
) *http.Server {
	mid := []middleware.Middleware{
		recovery.Recovery(),
		validate.Validator(),
		metadata.Server(metadata.WithPropagatedPrefix("")),
	}

	if option.Middleware != nil {
		mid = append(mid, option.Middleware...)
	}

	if tp != nil {
		mid = append(mid, Tracing(tp))
	}
	// mid = append(mid, ErrorMiddleware(logger))
	var opts = []http.ServerOption{
		http.Middleware(mid...),
	}

	if option.Network != "" {
		opts = append(opts, http.Network(option.Network))
	}

	if option.Addr != "" {
		opts = append(opts, http.Address(option.Addr))
	}

	if option.Timeout > 0 {
		opts = append(opts, http.Timeout(option.Timeout))
	}

	srv := http.NewServer(opts...)

	if fn != nil {
		fn(srv)
	}

	return srv
}

type GRPCOption struct {
	Network    string
	Addr       string
	Timeout    time.Duration
	Middleware []middleware.Middleware
}

func NewGRPCServer(
	option GRPCOption,
	tp *trace.TracerProvider,
	logger log.Logger,
	fn func(srv *grpc.Server),
) *grpc.Server {
	mid := []middleware.Middleware{
		recovery.Recovery(),
		validate.Validator(),
		metadata.Server(metadata.WithPropagatedPrefix("")),
	}

	if option.Middleware != nil {
		mid = append(mid, option.Middleware...)
	}

	if tp != nil {
		mid = append(mid, Tracing(tp))
	}

	var opts = []grpc.ServerOption{
		grpc.Middleware(mid...),
	}

	if option.Network != "" {
		opts = append(opts, grpc.Network(option.Network))
	}

	if option.Addr != "" {
		opts = append(opts, grpc.Address(option.Addr))
	}

	if option.Timeout > 0 {
		opts = append(opts, grpc.Timeout(option.Timeout))
	}

	srv := grpc.NewServer(opts...)

	if fn != nil {
		fn(srv)
	}

	return srv
}

func Logging(logger log.Logger) middleware.Middleware {
	return MLogging(logger)
}

func Tracing(tp *trace.TracerProvider) middleware.Middleware {
	return selector.Server(
		tracing.Server(tracing.WithTracerProvider(tp)),
		func(h middleware.Handler) middleware.Handler {
			return func(ctx context.Context, req interface{}) (any, error) {
				tr, ok := transport.FromServerContext(ctx)
				if !ok {
					return nil, errors.Unauthorized("unauthorized", "something went wrong")
				}
				if span := otrace.SpanContextFromContext(ctx); span.HasTraceID() {
					tr.ReplyHeader().Set("X-Trace-Id", span.TraceID().String())
				}
				return h(ctx, req)
			}
		},
	).Match(NewOperationExclude(
		healthcheck.OperationHealthCheckServiceHealthzCheck,
		healthcheck.OperationHealthCheckServicePingCheck,
	)).Build()
}
