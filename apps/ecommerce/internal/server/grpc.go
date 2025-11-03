package server

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/validate"
	"github.com/go-kratos/kratos/v2/transport/grpc"

	"github.com/toeydevelopment/telltheworld/apps/ecommerce/internal/conf"
	"github.com/toeydevelopment/telltheworld/apps/ecommerce/internal/service"

	v1 "github.com/toeydevelopment/telltheworld/apis/ecommerce/v1"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(c *conf.Server, ecommerce *service.EcommerceService, logger log.Logger) *grpc.Server {
	var opts = []grpc.ServerOption{
		grpc.Middleware(
			recovery.Recovery(),
			validate.Validator(),
			logging.Server(logger),
		),
	}
	if c.Grpc.Network != "" {
		opts = append(opts, grpc.Network(c.Grpc.Network))
	}
	if c.Grpc.Addr != "" {
		opts = append(opts, grpc.Address(c.Grpc.Addr))
	}
	if c.Grpc.Timeout != nil {
		opts = append(opts, grpc.Timeout(c.Grpc.Timeout.AsDuration()))
	}
	srv := grpc.NewServer(opts...)
	v1.RegisterEcommerceServiceServer(srv, ecommerce)
	return srv
}
