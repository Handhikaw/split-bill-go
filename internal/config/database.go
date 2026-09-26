package config

import (
	"log"

	// Import dua ini sangat penting

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitDB(cfg *Config) *gorm.DB {
	db, err := gorm.Open(postgres.Open(cfg.DBDsn()), &gorm.Config{})
	if err != nil {
		log.Fatalf("❌ failed to connect database: %v", err)
	}

	log.Println("✅ database connected")
	return db
}
