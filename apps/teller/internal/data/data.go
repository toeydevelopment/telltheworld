package data

import (
	"github.com/google/wire"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/biz"
	"github.com/toeydevelopment/telltheworld/pkg/healthcheck"
	"github.com/toeydevelopment/telltheworld/pkg/wgorm"
	"gorm.io/gorm"
)

var ProviderSet = wire.NewSet(
	NewData,
	NewGORM,
	NewGORMConfig,
	NewHealthZ,
	NewNotificationRepo,
	NewPubSub,
	wire.Bind(new(biz.NotificationRepo), new(*notificationRepo)),
)

type Data struct {
	db *gorm.DB
}

func NewData(db *gorm.DB) *Data {
	return &Data{db: db}
}

func NewHealthZ(db *gorm.DB) []healthcheck.HealthZ {
	return []healthcheck.HealthZ{
		wgorm.NewHealthZ(db),
	}
}
