package wkratos

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/toeydevelopment/telltheworld/pkg/healthcheck"
)

func MLogging(logger log.Logger) middleware.Middleware {
	return selector.Server(logging.Server(logger)).Match(NewOperationExclude(
		healthcheck.OperationHealthCheckServiceHealthzCheck,
		healthcheck.OperationHealthCheckServicePingCheck,
	)).Build()
}
