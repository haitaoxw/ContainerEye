package models

import (
	"time"
	"gorm.io/gorm"
)

type Operator string

const (
	OperatorGT  Operator = ">"
	OperatorLT  Operator = "<"
	OperatorGTE Operator = ">="
	OperatorLTE Operator = "<="
	OperatorEQ  Operator = "=="
)

type Metric string

const (
	MetricCPUUsage    Metric = "cpu_percent"
	MetricMemoryUsage Metric = "memory_percent"
	MetricDiskIO      Metric = "disk_io_total"
	MetricNetworkIO   Metric = "network_total"
)

type AlertRule struct {
	gorm.Model
	Name           string    `json:"name" gorm:"uniqueIndex;not null"`
	Description    string    `json:"description"`
	ContainerID    string    `json:"container_id"`    // Optional, specific container
	ContainerName  string    `json:"container_name"`  // Optional, container name pattern
	Metric         Metric    `json:"metric" gorm:"not null"`
	Operator       Operator  `json:"operator" gorm:"not null"`
	Threshold      float64   `json:"threshold" gorm:"not null"`
	Duration       int       `json:"duration" gorm:"not null"` // In seconds
	CooldownPeriod int       `json:"cooldown_period"` // In seconds, minimum time between alerts
	Level          AlertLevel `json:"level" gorm:"not null"`
	IsEnabled      bool      `json:"is_enabled" gorm:"default:true"`
	LastTriggered  *time.Time `json:"last_triggered"`
	LastChecked    *time.Time `json:"last_checked"`
	TriggerCount   int       `json:"trigger_count" gorm:"default:0"`
	ResolvedCount  int       `json:"resolved_count" gorm:"default:0"`
}
