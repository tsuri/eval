package db

import (
	"fmt"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func NewDB(service string, schema ...interface{}) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("/data/sqlite/%s.db", service)), &gorm.Config{})
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
