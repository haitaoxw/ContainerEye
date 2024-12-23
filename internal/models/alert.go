package models

import (
	"time"
	"gorm.io/gorm"
)

type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "INFO"
	AlertLevelWarning  AlertLevel = "WARNING"
	AlertLevelCritical AlertLevel = "CRITICAL"
)

type AlertStatus string

const (
	AlertStatusPending      AlertStatus = "PENDING"
	AlertStatusActive       AlertStatus = "ACTIVE"
	AlertStatusResolved     AlertStatus = "RESOLVED"
	AlertStatusAcknowledged AlertStatus = "ACKNOWLEDGED"
)

type Alert struct {
	gorm.Model
	RuleID          uint        `json:"rule_id"`
	RuleName        string      `json:"rule_name"`
	ContainerID     string      `json:"container_id"`
	ContainerName   string      `json:"container_name"`
	Metric          string      `json:"metric"`
	Threshold       float64     `json:"threshold"`
	CurrentValue    float64     `json:"current_value"`
	Level           AlertLevel  `json:"level"`
	Message         string      `json:"message"`
	Status          AlertStatus `json:"status"`
	StartTime       time.Time   `json:"start_time"`
	EndTime         time.Time   `json:"end_time"`
	Value           float64     `json:"value"`
	AcknowledgedBy  string      `json:"acknowledged_by,omitempty"`
	AcknowledgedAt  time.Time   `json:"acknowledged_at,omitempty"`
	ResolvedBy      string      `json:"resolved_by,omitempty"`
	ResolvedAt      time.Time   `json:"resolved_at,omitempty"`
}
