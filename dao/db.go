package dao

import (
	"fmt"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

func ConnectDatabase() error {
	var err error
	db, err = gorm.Open(sqlite.Open("chain.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("open database error: %v", err)
	}
	db.AutoMigrate(
		&Block{},
		&Address{},
		&Transfer{},
	)

	return nil
}

func Transaction() *gorm.DB {
	return db.Begin()
}

func DB() *gorm.DB {
	return db
}
