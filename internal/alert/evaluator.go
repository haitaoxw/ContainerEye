package alert

import (
	"fmt"
	"sync"
	"time"

	"github.com/containereye/internal/models"
	"gorm.io/gorm"
)

type RuleEvaluator struct {
	alertManager *AlertManager
	db          *gorm.DB
	stateCache  map[uint]*ruleState
	mutex       sync.RWMutex
}

type ruleState struct {
	ViolationStart time.Time
	IsViolating    bool
	LastValue      float64
}

func NewRuleEvaluator(alertManager *AlertManager, db *gorm.DB) *RuleEvaluator {
	return &RuleEvaluator{
		alertManager: alertManager,
		db:          db,
		stateCache:  make(map[uint]*ruleState),
	}
}

func (e *RuleEvaluator) EvaluateMetric(rule *models.AlertRule, stats *models.ContainerStats) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	state, ok := e.stateCache[rule.ID]
	if !ok {
		state = &ruleState{}
		e.stateCache[rule.ID] = state
	}

	currentValue := e.extractMetricValue(rule.Metric, stats)
	isViolating := e.evaluateCondition(rule.Operator, currentValue, rule.Threshold)
	now := time.Now()

	if isViolating {
		if !state.IsViolating {
			// Condition just started violating
			state.ViolationStart = now
			state.IsViolating = true
		}

		// Check if violation duration exceeds rule duration
		if time.Since(state.ViolationStart) >= time.Duration(rule.Duration)*time.Second {
			// Create and send alert
			alert := &models.Alert{
				RuleID:        rule.ID,
				ContainerID:   stats.ContainerID,
				ContainerName: stats.ContainerName,
				Level:         rule.Level,
				Metric:        string(rule.Metric),
				Threshold:     rule.Threshold,
				CurrentValue:  currentValue,
				Message:       e.formatAlertMessage(rule, currentValue),
				Status:        models.AlertStatusActive,
				StartTime:     state.ViolationStart,
				Value:         currentValue,
			}
			
			if err := e.alertManager.SendAlert(alert); err != nil {
				return fmt.Errorf("failed to send alert: %v", err)
			}
			
			// Update rule statistics
			rule.LastTriggered = &now
			rule.TriggerCount++
			if err := e.db.Save(rule).Error; err != nil {
				return fmt.Errorf("failed to update rule: %v", err)
			}
		}
	} else {
		if state.IsViolating {
			// Condition just stopped violating
			state.IsViolating = false
		}
	}

	state.LastValue = currentValue
	return nil
}

func (e *RuleEvaluator) evaluateCondition(operator models.Operator, current, threshold float64) bool {
	switch operator {
	case models.OperatorGT:
		return current > threshold
	case models.OperatorLT:
		return current < threshold
	case models.OperatorGTE:
		return current >= threshold
	case models.OperatorLTE:
		return current <= threshold
	case models.OperatorEQ:
		return current == threshold
	default:
		return false
	}
}

func (e *RuleEvaluator) extractMetricValue(metric models.Metric, stats *models.ContainerStats) float64 {
	switch metric {
	case models.MetricCPUUsage:
		return stats.CPUPercent
	case models.MetricMemoryUsage:
		return stats.MemoryPercent
	case models.MetricDiskIO:
		return float64(stats.DiskIOTotal)
	case models.MetricNetworkIO:
		return float64(stats.NetworkTotal)
	default:
		return 0
	}
}

func (e *RuleEvaluator) formatAlertMessage(rule *models.AlertRule, currentValue float64) string {
	return fmt.Sprintf("Alert: %s - %s is %.2f (threshold: %.2f) for container %s",
		rule.Name,
		rule.Metric,
		currentValue,
		rule.Threshold,
		rule.ContainerName)
}
