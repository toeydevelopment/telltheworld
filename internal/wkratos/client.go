package wkratos

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	kgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/toeydevelopment/telltheworld/pkg/healthcheck"
	"go.opentelemetry.io/otel/sdk/trace"
	grpcx "google.golang.org/grpc"
)

const (
	GRPC_PREFIX     = "grpc://"
	GRPC_TLS_PREFIX = "grpcs://"
)

type ClientOption struct {
	Addr    string
	Timeout time.Duration
	Opts    []kgrpc.ClientOption
}

func NewClient(option ClientOption, tp *trace.TracerProvider) (*grpcx.ClientConn, error) {
	opts := []kgrpc.ClientOption{
		kgrpc.WithTimeout(option.Timeout),
		kgrpc.WithOptions(
			grpcx.WithStatsHandler(&tracing.ClientHandler{}),
			grpcx.WithDisableHealthCheck(),
		),
	}
	addr := option.Addr
	if strings.HasPrefix(addr, GRPC_PREFIX) {
		addr = addr[len(GRPC_PREFIX):]
	} else if strings.HasPrefix(addr, GRPC_TLS_PREFIX) {
		addr = addr[len(GRPC_TLS_PREFIX):]
		certPool, err := x509.SystemCertPool()
		if err != nil {
			return nil, err
		}
		opts = append(opts, kgrpc.WithTLSConfig(&tls.Config{
			RootCAs:    certPool,
			MinVersion: tls.VersionTLS12,
		}))
	}
	opts = append(opts, kgrpc.WithEndpoint(addr))
	if tp != nil {
		opts = append(opts, kgrpc.WithMiddleware(ClientTracing(tp)))
	}

	if option.Opts != nil {
		opts = append(opts, option.Opts...)
	}

	conn, err := kgrpc.DialInsecure(context.TODO(), opts...)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

type HTTPClientOption struct {
	Addr    string
	Timeout time.Duration
	Opts    []khttp.ClientOption
}

func NewHTTPClient(option HTTPClientOption, tp *trace.TracerProvider) (*khttp.Client, error) {
	opts := []khttp.ClientOption{
		khttp.WithEndpoint(option.Addr),
		khttp.WithTimeout(option.Timeout),
	}
	if tp != nil {
		opts = append(opts, khttp.WithMiddleware(ClientTracing(tp)))
	}
	if option.Opts != nil {
		opts = append(opts, option.Opts...)
	}

	client, err := khttp.NewClient(context.TODO(), opts...)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func ClientTracing(tp *trace.TracerProvider) middleware.Middleware {
	return selector.Client(tracing.Client(tracing.WithTracerProvider(tp))).
		Match(NewOperationExclude(
			healthcheck.OperationHealthCheckServiceHealthzCheck,
			healthcheck.OperationHealthCheckServicePingCheck,
		)).Build()
}
