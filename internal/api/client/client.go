package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/containereye/internal/models"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient() (*Client, error) {
	baseURL := os.Getenv("CONTAINEREYE_API_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	apiKey := os.Getenv("CONTAINEREYE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("CONTAINEREYE_API_KEY environment variable is not set")
	}

	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

func (c *Client) ListContainers() ([]models.Container, error) {
	var containers []models.Container
	if err := c.get("/api/v1/containers", &containers); err != nil {
		return nil, err
	}
	return containers, nil
}

func (c *Client) GetContainerStats(containerID string) (*models.ContainerStats, error) {
	var stats models.ContainerStats
	if err := c.get(fmt.Sprintf("/api/v1/containers/%s/stats", containerID), &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

func (c *Client) GetContainerStatsHistory(containerID string, from, to *time.Time, limit int) ([]models.ContainerStats, error) {
	endpoint := fmt.Sprintf("/api/v1/containers/%s/stats", containerID)
	
	query := url.Values{}
	if from != nil {
		query.Set("start", from.Format(time.RFC3339))
	}
	if to != nil {
		query.Set("end", to.Format(time.RFC3339))
	}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}

	var stats []models.ContainerStats
	if err := c.get(endpoint+"?"+query.Encode(), &stats); err != nil {
		return nil, err
	}
	return stats, nil
}

func (c *Client) ListAlerts(status, level string) ([]models.Alert, error) {
	endpoint := "/api/v1/alerts"
	
	query := url.Values{}
	if status != "" {
		query.Set("status", status)
	}
	if level != "" {
		query.Set("level", level)
	}

	var alerts []models.Alert
	if err := c.get(endpoint+"?"+query.Encode(), &alerts); err != nil {
		return nil, err
	}
	return alerts, nil
}

func (c *Client) AcknowledgeAlert(alertID, comment string) error {
	data := map[string]string{
		"comment": comment,
	}
	return c.post(fmt.Sprintf("/api/v1/alerts/%s/acknowledge", alertID), data, nil)
}

func (c *Client) ResolveAlert(alertID, comment string) error {
	data := map[string]string{
		"comment": comment,
	}
	return c.post(fmt.Sprintf("/api/v1/alerts/%s/resolve", alertID), data, nil)
}

func (c *Client) ExportContainerStats(containerID string, from, to *time.Time, format, output string) error {
	endpoint := fmt.Sprintf("/api/v1/containers/%s/stats/export", containerID)
	
	query := url.Values{}
	query.Set("format", format)
	if from != nil {
		query.Set("start", from.Format(time.RFC3339))
	}
	if to != nil {
		query.Set("end", to.Format(time.RFC3339))
	}

	resp, err := c.doRequest(http.MethodGet, endpoint+"?"+query.Encode(), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func (c *Client) get(endpoint string, v interface{}) error {
	resp, err := c.doRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return json.NewDecoder(resp.Body).Decode(v)
}

func (c *Client) post(endpoint string, data, v interface{}) error {
	var body io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %v", err)
		}
		body = bytes.NewReader(jsonData)
	}

	resp, err := c.doRequest(http.MethodPost, endpoint, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if v != nil {
		return json.NewDecoder(resp.Body).Decode(v)
	}
	return nil
}

func (c *Client) doRequest(method, endpoint string, body io.Reader) (*http.Response, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %v", err)
	}
	u.Path = path.Join(u.Path, endpoint)

	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("X-API-Key", c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
			return nil, fmt.Errorf("API error: %s", errResp.Error)
		}
		return nil, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	return resp, nil
}
