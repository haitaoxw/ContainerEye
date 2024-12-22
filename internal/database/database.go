package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/containereye/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	db   *gorm.DB
	once sync.Once
)

// Initialize initializes the database connection
func Initialize(dbPath string) error {
	var initErr error
	once.Do(func() {
		// Ensure the directory exists
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			initErr = fmt.Errorf("failed to create database directory: %v", err)
			return
		}

		var err error
		db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
		if err != nil {
			initErr = fmt.Errorf("failed to connect to database: %v", err)
			return
		}

		// Auto migrate the schema
		if err := db.AutoMigrate(
			&models.Container{},
			&models.ContainerStats{},
			&models.Alert{},
			&models.AlertRule{},
			&models.User{},
		); err != nil {
			initErr = fmt.Errorf("failed to migrate database: %v", err)
			return
		}

		log.Printf("Database initialized at %s", dbPath)
	})

	return initErr
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	if db == nil {
		panic("Database not initialized. Call Initialize() first")
	}
	return db
}

// Close closes the database connection
func Close() error {
	if db == nil {
		return nil
	}
	
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying *sql.DB: %v", err)
	}
	
	return sqlDB.Close()
}
