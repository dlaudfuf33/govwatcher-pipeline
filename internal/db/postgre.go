package db

import (
	"fmt"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"gwatch-data-pipeline/internal/logging"
)

var DB *gorm.DB

func InitDB() {
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASS")
	host := os.Getenv("DB_HOST")
	db := os.Getenv("DB_NAME")
	port := os.Getenv("DB_PORT")

	if user == "" || pass == "" || host == "" || port == "" || db == "" {
		logging.Errorf("One or more required DB environment variables are not set.")
		return
	}

	// PostgreSQL DSN
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Seoul search_path=public",
		host, user, pass, db, port)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		logging.Errorf("Failed to connect to PostgreSQL: %v", err)
		return
	}

	sqlDB, err := DB.DB()
	if err != nil {
		logging.Errorf("Failed to get raw DB instance: %v", err)
		return
	}

	sqlDB.SetMaxOpenConns(30)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)

	logging.Infof("Connected to PostgreSQL successfully.")
}

func CloseDB() {
	sqlDB, err := DB.DB()
	if err != nil {
		logging.Warnf("Failed to get raw DB: %v", err)
		return
	}
	sqlDB.Close()
}
