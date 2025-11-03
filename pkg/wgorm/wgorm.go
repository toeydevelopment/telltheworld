package wgorm

import (
	"context"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/plugin/opentelemetry/tracing"
)

func setDBLog() logger.Interface {
	if os.Getenv("SERVICE_ENV") == "production" {
		return logger.Default.LogMode(logger.Silent)
	}
	return nil
}

// / dsn := "host=localhost user=gorm password=gorm dbname=gorm port=9920".
func New(cfg *Config) (*gorm.DB, func(), error) {
	db, err := gorm.Open(postgres.Open(cfg.GetConnectionString()), &gorm.Config{
		Logger: setDBLog(),
	})

	if err != nil {
		return nil, nil, err
	}

	sdb, err := db.DB()

	if err != nil {
		return nil, nil, err
	}

	sdb.SetMaxIdleConns(cfg.MinPool)
	sdb.SetMaxOpenConns(cfg.MaxPool)

	if cfg.EnableTracing {
		if err = db.Use(tracing.NewPlugin(
			tracing.WithDBSystem(cfg.DBName),
		)); err != nil {
			return nil, nil, err
		}
	}

	if err = sdb.PingContext(context.TODO()); err != nil {
		return nil, nil, err
	}

	return db, func() {
		sdb.Close()
	}, nil
}

type HealthZ struct {
	db *gorm.DB
}

func NewHealthZ(db *gorm.DB) *HealthZ {
	return &HealthZ{
		db: db,
	}
}

func (d *HealthZ) Ping(ctx context.Context) error {
	db, err := d.db.DB()

	if err != nil {
		return err
	}

	return db.PingContext(ctx)
}
