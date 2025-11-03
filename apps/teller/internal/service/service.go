package service

import (
	"github.com/google/wire"
	"github.com/toeydevelopment/telltheworld/pkg/healthcheck"
)

var ProviderSet = wire.NewSet(NewNotification, healthcheck.NewService)
