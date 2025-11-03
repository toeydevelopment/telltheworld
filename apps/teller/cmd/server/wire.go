//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/biz"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/conf"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/data"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/server"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/service"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// wireApp init kratos application.
func wireApp(*conf.Server, *conf.Data, *conf.PubSubConfig, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp))
}
