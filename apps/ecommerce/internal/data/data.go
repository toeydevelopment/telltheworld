package data

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"

	"github.com/toeydevelopment/telltheworld/apps/ecommerce/internal/biz"
	"github.com/toeydevelopment/telltheworld/apps/ecommerce/internal/conf"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewCommerceRepo, wire.Bind(new(biz.EcommerceRepo), new(*commerceRepo)))

// Data contains all data sources
type Data struct {
	// Add database connections here if needed
	log *log.Helper
}

// NewData creates a new Data instance
func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	log := log.NewHelper(log.With(logger, "module", "data"))

	cleanup := func() {
		log.Info("closing data resources")
	}

	return &Data{
		log: log,
	}, cleanup, nil
}
