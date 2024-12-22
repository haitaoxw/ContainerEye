package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	
	"containereye/internal/models"
)

type SlackNotifier struct {
	WebhookURL string
	Channel    string
	Username   string
}

type SlackMessage struct {
	Channel     string        `json:"channel,omitempty"`
	Username    string        `json:"username,omitempty"`
	IconEmoji   string        `json:"icon_emoji,omitempty"`
	Attachments []Attachment  `json:"attachments"`
}

type Attachment struct {
	Color      string   `json:"color"`
	Title      string   `json:"title"`
	Text       string   `json:"text"`
	Fields     []Field  `json:"fields"`
	Footer     string   `json:"footer"`
	Ts         int64    `json:"ts"`
}

type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

func NewSlackNotifier(webhookURL, channel, username string) *SlackNotifier {
	return &SlackNotifier{
		WebhookURL: webhookURL,
		Channel:    channel,
		Username:   username,
	}
}

func (s *SlackNotifier) Notify(alert *models.Alert) error {
	// Create message
	msg := &SlackMessage{
		Channel:  s.Channel,
		Username: s.Username,
		IconEmoji: getAlertEmoji(alert.Level),
		Attachments: []Attachment{
			{
				Color:  getAlertColor(alert.Level),
				Title:  fmt.Sprintf("ContainerEye Alert: %s", alert.RuleName),
				Text:   alert.Message,
				Fields: []Field{
					{
						Title: "Container",
						Value: alert.ContainerName,
						Short: true,
					},
					{
						Title: "Level",
						Value: string(alert.Level),
						Short: true,
					},
					{
						Title: "Metric",
						Value: string(alert.Metric),
						Short: true,
					},
					{
						Title: "Value",
						Value: fmt.Sprintf("%.2f", alert.Value),
						Short: true,
					},
					{
						Title: "Started",
						Value: alert.StartTime.Format(time.RFC3339),
						Short: true,
					},
					{
						Title: "Duration",
						Value: alert.EndTime.Sub(alert.StartTime).String(),
						Short: true,
					},
				},
				Footer: "ContainerEye Alert System",
				Ts:     time.Now().Unix(),
			},
		},
	}

	// Marshal message
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal slack message: %v", err)
	}

	// Send request
	resp, err := http.Post(s.WebhookURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to send slack message: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack API returned non-200 status code: %d", resp.StatusCode)
	}

	return nil
}

func getAlertColor(level models.AlertLevel) string {
	switch level {
	case models.AlertLevelCritical:
		return "#FF0000"
	case models.AlertLevelWarning:
		return "#FFA500"
	case models.AlertLevelInfo:
		return "#0000FF"
	default:
		return "#808080"
	}
}

func getAlertEmoji(level models.AlertLevel) string {
	switch level {
	case models.AlertLevelCritical:
		return ":red_circle:"
	case models.AlertLevelWarning:
		return ":warning:"
	case models.AlertLevelInfo:
		return ":information_source:"
	default:
		return ":bell:"
	}
}
