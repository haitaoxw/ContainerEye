package alert

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"

	"containereye/internal/models"
	"gorm.io/gorm"
)

type RuleManager struct {
	evaluator *RuleEvaluator
	db        *gorm.DB
}

func NewRuleManager(alertManager *AlertManager, db *gorm.DB) *RuleManager {
	return &RuleManager{
		evaluator: NewRuleEvaluator(alertManager, db),
		db:        db,
	}
}

func (rm *RuleManager) CreateRule(rule *models.AlertRule) error {
	return rm.db.Create(rule).Error
}

func (rm *RuleManager) UpdateRule(rule *models.AlertRule) error {
	return rm.db.Save(rule).Error
}

func (rm *RuleManager) DeleteRule(id uint) error {
	return rm.db.Delete(&models.AlertRule{}, id).Error
}

func (rm *RuleManager) GetRule(id uint) (*models.AlertRule, error) {
	var rule models.AlertRule
	if err := rm.db.First(&rule, id).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

func (rm *RuleManager) ListRules(enabled *bool) ([]models.AlertRule, error) {
	var rules []models.AlertRule
	query := rm.db
	if enabled != nil {
		query = query.Where("is_enabled = ?", *enabled)
	}
	if err := query.Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

func (rm *RuleManager) EnableRule(id uint) error {
	return rm.db.Model(&models.AlertRule{}).Where("id = ?", id).Update("is_enabled", true).Error
}

func (rm *RuleManager) DisableRule(id uint) error {
	return rm.db.Model(&models.AlertRule{}).Where("id = ?", id).Update("is_enabled", false).Error
}

func (rm *RuleManager) EvaluateRules(stats *models.ContainerStats) error {
	var rules []models.AlertRule
	if err := rm.db.Where("is_enabled = ?", true).Find(&rules).Error; err != nil {
		return fmt.Errorf("failed to fetch rules: %v", err)
	}

	for _, rule := range rules {
		// Skip if container targeting doesn't match
		if rule.ContainerID != "" && rule.ContainerID != stats.ContainerID {
			continue
		}
		if rule.ContainerName != "" && rule.ContainerName != stats.ContainerName {
			continue
		}

		if err := rm.evaluator.EvaluateMetric(&rule, stats); err != nil {
			return fmt.Errorf("failed to evaluate rule %d: %v", rule.ID, err)
		}
	}

	return nil
}

func (rm *RuleManager) CreateDefaultRules() error {
	rules := []models.AlertRule{
		{
			Name:        "High CPU Usage",
			Description: "Alert when CPU usage is above 90%",
			Metric:      models.MetricCPUUsage,
			Operator:    models.OperatorGT,
			Threshold:   90,
			Duration:    300,
			Level:       models.AlertLevelWarning,
			CooldownPeriod: 1800, // 30 minutes
		},
		{
			Name:        "Critical Memory Usage",
			Description: "Alert when memory usage is above 95%",
			Metric:      models.MetricMemoryUsage,
			Operator:    models.OperatorGT,
			Threshold:   95,
			Duration:    180,
			Level:       models.AlertLevelCritical,
			CooldownPeriod: 900, // 15 minutes
		},
		{
			Name:        "High Disk I/O",
			Description: "Alert when disk I/O is above 100MB/s",
			Metric:      models.MetricDiskIO,
			Operator:    models.OperatorGT,
			Threshold:   100 * 1024 * 1024, // 100MB/s
			Duration:    120,
			Level:       models.AlertLevelWarning,
			CooldownPeriod: 1200, // 20 minutes
		},
		{
			Name:        "High Network Traffic",
			Description: "Alert when network traffic is above 100MB/s",
			Metric:      models.MetricNetworkIO,
			Operator:    models.OperatorGT,
			Threshold:   100 * 1024 * 1024, // 100MB/s
			Duration:    120,
			Level:       models.AlertLevelWarning,
			CooldownPeriod: 1200, // 20 minutes
		},
	}

	for _, rule := range rules {
		if err := rm.CreateRule(&rule); err != nil {
			return fmt.Errorf("failed to create default rule %s: %v", rule.Name, err)
		}
	}

	return nil
}

func (rm *RuleManager) ImportRulesFromFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	var rules []models.AlertRule
	if err := json.Unmarshal(data, &rules); err != nil {
		return fmt.Errorf("failed to parse rules: %v", err)
	}

	// Begin transaction
	tx := rm.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for _, rule := range rules {
		// Clear ID to ensure new records are created
		rule.ID = 0
		if err := tx.Create(&rule).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to import rule '%s': %v", rule.Name, err)
		}
	}

	return tx.Commit().Error
}

func (rm *RuleManager) ExportRulesToFile(filename string) error {
	rules, err := rm.ListRules(nil)
	if err != nil {
		return fmt.Errorf("failed to fetch rules: %v", err)
	}

	data, err := json.MarshalIndent(rules, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal rules: %v", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	return nil
}

func (rm *RuleManager) TestRule(rule *models.AlertRule, startTime, endTime time.Time) ([]models.Alert, error) {
	var alerts []models.Alert
	
	// Generate test data points
	interval := time.Minute
	for t := startTime; t.Before(endTime); t = t.Add(interval) {
		stats := &models.ContainerStats{
			ContainerID:   "test-container",
			ContainerName: "test-container",
			Timestamp:     t,
		}

		// Set test metric value
		value := generateRandomValue(0, 100, rule.Threshold)
		switch rule.Metric {
		case models.MetricCPUUsage:
			stats.CPUPercent = value
		case models.MetricMemoryUsage:
			stats.MemoryPercent = value
		case models.MetricDiskIO:
			stats.DiskIOTotal = uint64(value)
		case models.MetricNetworkIO:
			stats.NetworkTotal = uint64(value)
		}

		// Evaluate rule with test data
		if err := rm.evaluator.EvaluateMetric(rule, stats); err != nil {
			return nil, fmt.Errorf("failed to evaluate test data: %v", err)
		}
	}

	// Fetch generated alerts
	if err := rm.db.Where("rule_id = ? AND created_at BETWEEN ? AND ?", 
		rule.ID, startTime, endTime).Find(&alerts).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch test alerts: %v", err)
	}

	return alerts, nil
}

func (rm *RuleManager) TestRuleWithSampleData(rule *models.AlertRule) ([]models.Alert, error) {
	// Generate sample data for the last hour
	endTime := time.Now()
	startTime := endTime.Add(-1 * time.Hour)
	interval := time.Minute

	var stats []models.ContainerStats
	currentTime := startTime

	// Generate sample data points
	for currentTime.Before(endTime) {
		// Generate random metric value around the threshold
		var metricValue float64
		switch rule.Metric {
		case models.MetricCPUUsage:
			metricValue = generateRandomValue(0, 100, rule.Threshold)
		case models.MetricMemoryUsage:
			metricValue = generateRandomValue(0, 100, rule.Threshold)
		case models.MetricDiskIO:
			metricValue = generateRandomValue(0, 1000*1024*1024, rule.Threshold) // 0-1000MB
		case models.MetricNetworkIO:
			metricValue = generateRandomValue(0, 100*1024*1024, rule.Threshold) // 0-100MB/s
		}

		stat := models.ContainerStats{
			ContainerID:     "test-container",
			ContainerName:   "test-container",
			Timestamp:       currentTime,
			CPUPercent:      metricValue,
			MemoryPercent:   metricValue,
			NetworkTotal:    uint64(metricValue),
			DiskIOTotal:     uint64(metricValue),
		}
		stats = append(stats, stat)
		currentTime = currentTime.Add(interval)
	}

	// Create temporary table for test data
	tempTableName := fmt.Sprintf("temp_container_stats_%d", time.Now().UnixNano())
	if err := rm.db.Table(tempTableName).AutoMigrate(&models.ContainerStats{}); err != nil {
		return nil, fmt.Errorf("failed to create temporary table: %v", err)
	}
	defer rm.db.Migrator().DropTable(tempTableName)

	// Insert sample data
	if err := rm.db.Table(tempTableName).Create(&stats).Error; err != nil {
		return nil, fmt.Errorf("failed to insert sample data: %v", err)
	}

	// Test rule with sample data
	alerts, err := rm.TestRule(rule, startTime, endTime)
	if err != nil {
		return nil, err
	}

	return alerts, nil
}

func generateRandomValue(min, max, threshold float64) float64 {
	// 70% chance to generate value around threshold
	if rand.Float64() < 0.7 {
		// Generate value within ±20% of threshold
		delta := threshold * 0.2
		return threshold + (rand.Float64()*2-1)*delta
	}
	// 30% chance to generate random value in full range
	return min + rand.Float64()*(max-min)
}
