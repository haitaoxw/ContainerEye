package alert

import (
	"encoding/json"
	"fmt"
	"time"

	"containereye/internal/database"
	"containereye/internal/models"
	"github.com/slack-go/slack"
	"gopkg.in/gomail.v2"
	"gorm.io/gorm"
	"strconv"
)

type AlertManager struct {
	slackClient *slack.Client
	emailDialer *gomail.Dialer
	config      *Config
	db          *gorm.DB
}

type Config struct {
	SlackToken     string
	SlackChannel   string
	SMTPHost       string
	SMTPPort       int
	EmailFrom      string
	EmailPassword  string
	EmailReceivers []string
}

func NewAlertManager(config *Config) *AlertManager {
	slackClient := slack.New(config.SlackToken)
	emailDialer := gomail.NewDialer(config.SMTPHost, config.SMTPPort, config.EmailFrom, config.EmailPassword)

	return &AlertManager{
		slackClient: slackClient,
		emailDialer: emailDialer,
		config:      config,
		db:          database.GetDB(),
	}
}

// SendAlert sends an alert through configured channels (Slack and Email)
func (am *AlertManager) SendAlert(alert *models.Alert) error {
	// Save alert to database
	if err := am.db.Create(alert).Error; err != nil {
		return fmt.Errorf("failed to save alert: %v", err)
	}

	// Send notifications
	if err := am.SendSlackAlert(alert); err != nil {
		return fmt.Errorf("failed to send slack alert: %v", err)
	}

	if err := am.SendEmailAlert(alert); err != nil {
		return fmt.Errorf("failed to send email alert: %v", err)
	}

	return nil
}

// AcknowledgeAlert marks an alert as acknowledged
func (am *AlertManager) AcknowledgeAlert(alertID string, userID string) error {
	var alert models.Alert
	if err := am.db.First(&alert, "id = ?", alertID).Error; err != nil {
		return fmt.Errorf("failed to find alert: %v", err)
	}

	alert.Status = models.AlertStatusAcknowledged
	alert.AcknowledgedBy = userID
	alert.AcknowledgedAt = time.Now()

	if err := am.db.Save(&alert).Error; err != nil {
		return fmt.Errorf("failed to update alert: %v", err)
	}

	return nil
}

// ResolveAlert marks an alert as resolved
func (am *AlertManager) ResolveAlert(alertID string, userID string) error {
	var alert models.Alert
	if err := am.db.First(&alert, "id = ?", alertID).Error; err != nil {
		return fmt.Errorf("failed to find alert: %v", err)
	}

	alert.Status = models.AlertStatusResolved
	alert.ResolvedBy = userID
	alert.ResolvedAt = time.Now()

	if err := am.db.Save(&alert).Error; err != nil {
		return fmt.Errorf("failed to update alert: %v", err)
	}

	return nil
}

func (am *AlertManager) SendSlackAlert(alert *models.Alert) error {
	attachment := slack.Attachment{
		Color: getAlertColor(alert.Level),
		Fields: []slack.AttachmentField{
			{
				Title: "Container",
				Value: alert.ContainerName,
				Short: true,
			},
			{
				Title: "Metric",
				Value: alert.Metric,
				Short: true,
			},
			{
				Title: "Current Value",
				Value: fmt.Sprintf("%.2f", alert.CurrentValue),
				Short: true,
			},
			{
				Title: "Threshold",
				Value: fmt.Sprintf("%.2f", alert.Threshold),
				Short: true,
			},
		},
		Footer: "Container Monitor Alert",
		Ts:     json.Number(strconv.FormatInt(time.Now().Unix(), 10)),
	}

	_, _, err := am.slackClient.PostMessage(
		am.config.SlackChannel,
		slack.MsgOptionAttachments(attachment),
	)
	return err
}

func (am *AlertManager) SendEmailAlert(alert *models.Alert) error {
	m := gomail.NewMessage()
	m.SetHeader("From", am.config.EmailFrom)
	m.SetHeader("To", am.config.EmailReceivers...)
	m.SetHeader("Subject", "Container Alert: "+string(alert.Level))
	
	body := fmt.Sprintf(`
		Container: %s
		Alert Level: %s
		Metric: %s
		Current Value: %.2f
		Threshold: %.2f
		Message: %s
		Time: %s
	`, alert.ContainerName, alert.Level, alert.Metric, 
	   alert.CurrentValue, alert.Threshold, alert.Message,
	   time.Now().Format(time.RFC3339))
	
	m.SetBody("text/plain", body)
	
	return am.emailDialer.DialAndSend(m)
}

func getAlertColor(level models.AlertLevel) string {
	switch level {
	case models.AlertLevelInfo:
		return "#36a64f"
	case models.AlertLevelWarning:
		return "#ffcc00"
	case models.AlertLevelCritical:
		return "#ff0000"
	default:
		return "#000000"
	}
}
