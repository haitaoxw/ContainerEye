package models

import (
	"time"

	"gorm.io/gorm"
)

type ReportSchedule struct {
	gorm.Model
	Name        string    `json:"name" gorm:"uniqueIndex;not null"`
	Type        string    `json:"type" gorm:"not null"`
	Schedule    string    `json:"schedule" gorm:"not null"` // Cron expression
	LastRun     time.Time `json:"last_run"`
	NextRun     time.Time `json:"next_run"`
	Recipients  []string  `json:"recipients" gorm:"type:json"`
	IsEnabled   bool      `json:"is_enabled" gorm:"default:true"`
	Description string    `json:"description"`
}

type ReportType string

const (
	ReportTypeDaily   ReportType = "daily"
	ReportTypeWeekly  ReportType = "weekly"
	ReportTypeMonthly ReportType = "monthly"
	ReportTypeCustom  ReportType = "custom"
)
