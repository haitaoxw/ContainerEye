package main

import (
	"log"
	"time"

	"github.com/containereye/internal/api"
	"github.com/containereye/internal/monitor"
	"github.com/containereye/internal/alert"
	"github.com/containereye/internal/config"
	"github.com/containereye/internal/database"
	"github.com/containereye/internal/models"
)

func main() {
	// Initialize configuration
	cfg := config.LoadConfig()

	// Initialize database
	if err := database.Initialize(cfg.Database.Path); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	db := database.GetDB()

	// Initialize alert manager
	alertConfig := &alert.Config{
		SlackToken:     cfg.Alert.Slack.Token,
		SlackChannel:   cfg.Alert.Slack.Channel,
		SMTPHost:       cfg.Alert.Email.SMTPHost,
		SMTPPort:       cfg.Alert.Email.SMTPPort,
		EmailFrom:      cfg.Alert.Email.From,
		EmailPassword:  cfg.Alert.Email.Password,
		EmailReceivers: cfg.Alert.Email.ToReceivers,
	}
	alertManager := alert.NewAlertManager(alertConfig)
	
	// Initialize rule manager
	ruleManager := alert.NewRuleManager(alertManager, db)
	
	// Create default rules if none exist
	var ruleCount int64
	if err := db.Model(&models.AlertRule{}).Count(&ruleCount).Error; err != nil {
		log.Printf("Warning: Failed to count rules: %v", err)
	} else if ruleCount == 0 {
		if err := ruleManager.CreateDefaultRules(); err != nil {
			log.Printf("Warning: Failed to create default rules: %v", err)
		}
	}

	// Initialize collector with 30-second interval
	collector, err := monitor.NewCollector(ruleManager, 30*time.Second)
	if err != nil {
		log.Fatalf("Failed to create collector: %v", err)
	}

	// Start collector
	if err := collector.Start(); err != nil {
		log.Fatalf("Failed to start collector: %v", err)
	}
	defer collector.Stop()

	// Initialize and start API server
	server := api.NewServer(collector, alertManager, ruleManager)
	if err := server.Start(cfg.Server.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
