package models

import (
	"time"

	"gorm.io/gorm"
)

// Container represents a Docker container
type Container struct {
	gorm.Model
	ContainerID   string `gorm:"uniqueIndex"`
	Name          string
	Image         string
	State         string
	Status        string
	Created       time.Time
	LastSeen      time.Time
	RestartCount  int
	LastStats     ContainerStats `gorm:"foreignKey:ContainerID;references:ContainerID"`
	StatsHistory  []ContainerStats
}

// ContainerStats represents container resource usage statistics
type ContainerStats struct {
	gorm.Model
	ContainerID   string    `json:"container_id" gorm:"index"`
	ContainerName string    `json:"container_name"`
	Timestamp     time.Time `json:"timestamp"`
	
	// CPU Statistics
	CPUPercent     float64 `json:"cpu_percent"`
	CPUSystemUsage uint64  `json:"cpu_system_usage"`
	CPUUsage       float64 `json:"cpu_usage"` // CPU usage in percentage
	
	// Memory Statistics
	MemoryUsage   uint64  `json:"memory_usage"`   // Memory usage in bytes
	MemoryLimit   uint64  `json:"memory_limit"`   // Memory limit in bytes
	MemoryPercent float64 `json:"memory_percent"` // Memory usage percentage
	
	// Network Statistics
	NetworkRx    uint64 `json:"network_rx"`     // Network received bytes
	NetworkTx    uint64 `json:"network_tx"`     // Network transmitted bytes
	NetworkTotal uint64 `json:"network_total"`  // Total network I/O
	
	// Disk I/O Statistics
	BlockRead    uint64 `json:"block_read"`    // Block IO read bytes
	BlockWrite   uint64 `json:"block_write"`   // Block IO write bytes
	DiskIOTotal  uint64 `json:"disk_io_total"` // Total disk I/O
	
	// Process Statistics
	PIDs         uint64 `json:"pids"`          // Number of processes
}
