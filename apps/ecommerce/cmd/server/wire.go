//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"

	"github.com/toeydevelopment/telltheworld/apps/ecommerce/internal/biz"
	"github.com/toeydevelopment/telltheworld/apps/ecommerce/internal/conf"
	"github.com/toeydevelopment/telltheworld/apps/ecommerce/internal/data"
	"github.com/toeydevelopment/telltheworld/apps/ecommerce/internal/server"
	"github.com/toeydevelopment/telltheworld/apps/ecommerce/internal/service"
)

// wireApp init kratos application.
func wireApp(*conf.Server, *conf.Data, *conf.TellerClient, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp))
}
