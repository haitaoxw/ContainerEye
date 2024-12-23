package alert

import (
	"fmt"
	"sync"
	"time"

	"containereye/internal/models"
	"gorm.io/gorm"
)

type AlertStatus string

const (
	AlertStatusNew       AlertStatus = "NEW"
	AlertStatusAcked    AlertStatus = "ACKNOWLEDGED"
	AlertStatusResolved AlertStatus = "RESOLVED"
	AlertStatusClosed   AlertStatus = "CLOSED"
)

type AlertHandler struct {
	db     *gorm.DB
	mutex  sync.RWMutex
	alerts map[uint]*models.Alert
}

type AlertUpdate struct {
	ID          uint
	Status      models.AlertStatus
	Comment     string
	Handler     string
	UpdatedAt   time.Time
}

func NewAlertHandler(db *gorm.DB) *AlertHandler {
	return &AlertHandler{
		db:     db,
		alerts: make(map[uint]*models.Alert),
	}
}

func (h *AlertHandler) HandleAlert(alert *models.Alert) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Check if alert already exists
	if existing, ok := h.alerts[alert.ID]; ok {
		// Update existing alert
		existing.Value = alert.Value
		existing.CurrentValue = alert.CurrentValue
		if alert.Status == models.AlertStatusResolved {
			existing.Status = models.AlertStatusResolved
			existing.ResolvedAt = time.Now()
		}
		return h.updateAlert(existing)
	}

	// Set initial status
	alert.Status = models.AlertStatusPending
	alert.StartTime = time.Now()

	// Save to database
	if err := h.db.Create(alert).Error; err != nil {
		return fmt.Errorf("failed to create alert: %v", err)
	}

	// Add to active alerts
	h.alerts[alert.ID] = alert

	return nil
}

func (h *AlertHandler) UpdateAlertStatus(update AlertUpdate) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	alert, ok := h.alerts[update.ID]
	if !ok {
		return fmt.Errorf("alert not found: %d", update.ID)
	}

	// Update alert status
	alert.Status = update.Status
	alert.AcknowledgedBy = update.Handler
	alert.Message = update.Comment
	alert.UpdatedAt = update.UpdatedAt

	// Update timestamps based on status
	switch update.Status {
	case models.AlertStatusAcknowledged:
		alert.AcknowledgedAt = update.UpdatedAt
	case models.AlertStatusResolved:
		alert.ResolvedAt = update.UpdatedAt
	}

	// Save to database
	if err := h.updateAlert(alert); err != nil {
		return err
	}

	// Remove from active alerts if resolved
	if update.Status == models.AlertStatusResolved {
		delete(h.alerts, alert.ID)
	}

	return nil
}

func (h *AlertHandler) CheckEscalations() []models.Alert {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	now := time.Now()
	var needEscalation []models.Alert

	for _, alert := range h.alerts {
		// Escalate if alert has been active for more than the threshold time
		if alert.Status == models.AlertStatusActive {
			threshold := h.getEscalationThreshold(alert.Level)
			if now.Sub(alert.StartTime) > threshold {
				needEscalation = append(needEscalation, *alert)
			}
		}
	}

	return needEscalation
}

func (h *AlertHandler) GetActiveAlerts() []models.Alert {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	alerts := make([]models.Alert, 0, len(h.alerts))
	for _, alert := range h.alerts {
		alerts = append(alerts, *alert)
	}
	return alerts
}

// Internal helper functions

func (h *AlertHandler) updateAlert(alert *models.Alert) error {
	return h.db.Save(alert).Error
}

func (h *AlertHandler) getEscalationThreshold(level models.AlertLevel) time.Duration {
	switch level {
	case models.AlertLevelCritical:
		return 15 * time.Minute
	case models.AlertLevelWarning:
		return 30 * time.Minute
	default:
		return 1 * time.Hour
	}
}
