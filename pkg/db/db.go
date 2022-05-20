package db

import (
	"fmt"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewDB(service string, schema ...interface{}) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("/data/sqlite/%s.db", service)),
		&gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})
	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema
	err = db.AutoMigrate(schema...)
	if err != nil {
		return nil, err
	}
	return db, nil
}
