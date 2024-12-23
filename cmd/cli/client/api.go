package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/viper"
	"containereye/internal/models"
)

// Container represents a Docker container
type Container struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Image   string    `json:"image"`
	Status  string    `json:"status"`
	Created time.Time `json:"created"`
}

// ContainerStats represents container resource usage statistics
type ContainerStats struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage uint64  `json:"memory_usage"`
	DiskIO      uint64  `json:"disk_io"`
	NetworkIO   uint64  `json:"network_io"`
}

// Alert represents a container alert
type Alert struct {
	ID        string    `json:"id"`
	Container string    `json:"container"`
	Message   string    `json:"message"`
	Level     string    `json:"level"`
	Time      time.Time `json:"time"`
}

type APIClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *APIClient) doRequest(method, path string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	token := viper.GetString("token")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %s", string(respBody))
	}

	return respBody, nil
}

func (c *APIClient) Login(username, password string) (string, error) {
	body := map[string]string{
		"username": username,
		"password": password,
	}

	resp, err := c.doRequest("POST", "/api/v1/auth/login", body)
	if err != nil {
		return "", err
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", err
	}

	return result.Token, nil
}

func (c *APIClient) GetContainers() ([]Container, error) {
	resp, err := c.doRequest("GET", "/api/v1/containers", nil)
	if err != nil {
		return nil, err
	}

	var containers []Container
	if err := json.Unmarshal(resp, &containers); err != nil {
		return nil, err
	}

	return containers, nil
}

func (c *APIClient) GetContainerStats(id string) (*ContainerStats, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/v1/containers/%s/stats", id), nil)
	if err != nil {
		return nil, err
	}

	var stats ContainerStats
	if err := json.Unmarshal(resp, &stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

func (c *APIClient) GetAlerts() ([]Alert, error) {
	resp, err := c.doRequest("GET", "/api/v1/alerts", nil)
	if err != nil {
		return nil, err
	}

	var alerts []Alert
	if err := json.Unmarshal(resp, &alerts); err != nil {
		return nil, err
	}

	return alerts, nil
}

func (c *APIClient) AcknowledgeAlert(id string) error {
	_, err := c.doRequest("PUT", fmt.Sprintf("/api/v1/alerts/%s/acknowledge", id), nil)
	return err
}

func (c *APIClient) ListRules(enabled *bool) ([]models.AlertRule, error) {
	url := fmt.Sprintf("%s/api/v1/rules", c.baseURL)
	if enabled != nil {
		url = fmt.Sprintf("%s?enabled=%v", url, *enabled)
	}

	var rules []models.AlertRule
	resp, err := c.doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(resp, &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

func (c *APIClient) GetRule(id uint) (*models.AlertRule, error) {
	url := fmt.Sprintf("%s/api/v1/rules/%d", c.baseURL, id)

	var rule models.AlertRule
	resp, err := c.doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(resp, &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

func (c *APIClient) CreateRule(rule *models.AlertRule) error {
	url := fmt.Sprintf("%s/api/v1/rules", c.baseURL)
	_, err := c.doRequest("POST", url, rule)
	return err
}

func (c *APIClient) UpdateRule(rule *models.AlertRule) error {
	url := fmt.Sprintf("%s/api/v1/rules/%d", c.baseURL, rule.ID)
	_, err := c.doRequest("PUT", url, rule)
	return err
}

func (c *APIClient) DeleteRule(id uint) error {
	url := fmt.Sprintf("%s/api/v1/rules/%d", c.baseURL, id)
	_, err := c.doRequest("DELETE", url, nil)
	return err
}

func (c *APIClient) EnableRule(id uint) error {
	url := fmt.Sprintf("%s/api/v1/rules/%d/enable", c.baseURL, id)
	_, err := c.doRequest("PUT", url, nil)
	return err
}

func (c *APIClient) DisableRule(id uint) error {
	url := fmt.Sprintf("%s/api/v1/rules/%d/disable", c.baseURL, id)
	_, err := c.doRequest("PUT", url, nil)
	return err
}

func (c *APIClient) ValidateRule(rule *models.AlertRule) error {
	url := fmt.Sprintf("%s/api/v1/rules/validate", c.baseURL)
	_, err := c.doRequest("POST", url, rule)
	return err
}

func (c *APIClient) ImportRules(rules []models.AlertRule) error {
	url := fmt.Sprintf("%s/api/v1/rules/import", c.baseURL)
	_, err := c.doRequest("POST", url, rules)
	return err
}

func (c *APIClient) ExportRules() ([]models.AlertRule, error) {
	url := fmt.Sprintf("%s/api/v1/rules/export", c.baseURL)

	var rules []models.AlertRule
	resp, err := c.doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(resp, &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

func (c *APIClient) TestRule(rule *models.AlertRule) ([]models.Alert, error) {
	body, err := json.Marshal(rule)
	if err != nil {
		return nil, err
	}

	data, err := c.doRequest("POST", "/api/rules/test", body)
	if err != nil {
		return nil, err
	}

	var alerts []models.Alert
	if err := json.Unmarshal(data, &alerts); err != nil {
		return nil, err
	}

	return alerts, nil
}

func (c *APIClient) GenerateReport(reportType string, startTime, endTime time.Time) error {
	params := map[string]interface{}{
		"type":       reportType,
		"start_time": startTime,
		"end_time":   endTime,
	}

	body, err := json.Marshal(params)
	if err != nil {
		return err
	}

	_, err = c.doRequest("POST", "/api/reports/generate", body)
	return err
}

func (c *APIClient) ScheduleReport(schedule *models.ReportSchedule) error {
	body, err := json.Marshal(schedule)
	if err != nil {
		return err
	}

	_, err = c.doRequest("POST", "/api/reports/schedule", body)
	return err
}

func (c *APIClient) ListScheduledReports() ([]models.ReportSchedule, error) {
	data, err := c.doRequest("GET", "/api/reports/schedule", nil)
	if err != nil {
		return nil, err
	}

	var schedules []models.ReportSchedule
	if err := json.Unmarshal(data, &schedules); err != nil {
		return nil, err
	}

	return schedules, nil
}

func (c *APIClient) DeleteScheduledReport(id string) error {
	_, err := c.doRequest("DELETE", fmt.Sprintf("/api/reports/schedule/%s", id), nil)
	return err
}
