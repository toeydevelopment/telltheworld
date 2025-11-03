package server

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	v1 "github.com/toeydevelopment/telltheworld/apis/teller/v1"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/conf"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/service"
	"github.com/toeydevelopment/telltheworld/internal/wkratos"
	"github.com/toeydevelopment/telltheworld/pkg/healthcheck"
)

func NewGRPCServer(
	c *conf.Server,
	ns *service.Notification,
	hs *healthcheck.Service,
	logger log.Logger,
) *grpc.Server {
	return wkratos.NewGRPCServer(wkratos.GRPCOption{
		Network: c.Grpc.Network,
		Addr:    c.Grpc.Addr,
		Timeout: c.Grpc.Timeout.AsDuration(),
	}, nil, logger, func(srv *grpc.Server) {
		v1.RegisterNotificationServiceServer(srv, ns)
	})
}
