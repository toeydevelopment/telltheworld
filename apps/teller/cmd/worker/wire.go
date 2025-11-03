//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/biz"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/conf"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/data"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/worker"
)

// wireWorker init worker application.
func wireWorker(*conf.Data, *conf.PubSubConfig, log.Logger) (*worker.Worker, func(), error) {
	panic(wire.Build(
		data.ProviderSet,
		biz.ProviderSet,
		worker.NewWorker,
	))
}