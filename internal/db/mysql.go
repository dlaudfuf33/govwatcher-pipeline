package db

import (
	"fmt"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"gwatch-data-pipeline/internal/logging"
)


var DB *gorm.DB
func InitDB() {
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASS")
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")

	if user == "" || pass == "" || host == "" || port == "" {
		logging.Errorf("One or more required DB environment variables are not set.")
		return
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/gwatch?charset=utf8mb4&parseTime=True&loc=Local", user, pass, host, port)

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		logging.Errorf("Failed to connect to DB: %v", err)
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

	logging.Infof("✅ Connected to MySQL successfully.")
}

func CloseDB() {
	sqlDB, err := DB.DB()
	if err != nil {
		logging.Warnf("Failed to get raw DB: %v", err)
		return
	}
	sqlDB.Close()
}
