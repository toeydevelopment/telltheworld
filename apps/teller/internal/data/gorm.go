package data

import (
	"os"
	"strconv"

	"github.com/toeydevelopment/telltheworld/apps/teller/internal/conf"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/data/entity"
	"github.com/toeydevelopment/telltheworld/pkg/wgorm"
	"gorm.io/gorm"
)

func NewGORMConfig(d *conf.Data) (*wgorm.Config, error) {
	db := d.GetDatabase()
	v, _ := strconv.ParseBool(db.GetEnableTracing())

	port, err := strconv.ParseInt(db.GetPort(), 10, 64)
	if err != nil {
		return nil, err
	}
	minPool, err := strconv.ParseInt(db.GetMinPool(), 10, 64)
	if err != nil {
		return nil, err
	}
	maxPool, err := strconv.ParseInt(db.GetMaxPool(), 10, 64)
	if err != nil {
		return nil, err
	}

	return &wgorm.Config{
		Host:          db.GetHost(),
		Port:          int(port),
		DBName:        db.GetDb(),
		MaxPool:       int(maxPool),
		MinPool:       int(minPool),
		Username:      db.GetUsername(),
		Password:      db.GetPassword(),
		EnableTracing: v,
	}, nil
}

func NewGORM(cfg *wgorm.Config) (*gorm.DB, func(), error) {
	db, closeDB, err := wgorm.New(cfg)

	if err != nil {
		return nil, nil, err
	}

	if os.Getenv("AUTO_MIGRATE") == "true" {
		if err = db.AutoMigrate(
			entity.Notification{},
		); err != nil {
			return nil, nil, err
		}
	}

	return db, closeDB, nil
}
