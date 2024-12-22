package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/containereye/internal/alert"
	"github.com/containereye/internal/database"
	"github.com/containereye/internal/models"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"
)

const (
	maxConcurrentCollections = 10
	maxBatchSize            = 100
	retryAttempts          = 3
	retryDelay             = 5 * time.Second
)

type Collector struct {
	dockerClient *client.Client
	ctx         context.Context
	ruleManager *alert.RuleManager
	interval    time.Duration
	mutex       sync.RWMutex
	containers  map[string]*models.ContainerStats
	stopChan    chan struct{}
	sem         *semaphore.Weighted
	metrics     *CollectorMetrics
}

type CollectorMetrics struct {
	mutex               sync.RWMutex
	totalCollections    uint64
	failedCollections   uint64
	totalProcessingTime time.Duration
	batchSize          int
}

func NewCollector(ruleManager *alert.RuleManager, interval time.Duration) (*Collector, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	
	return &Collector{
		dockerClient: cli,
		ctx:         ctx,
		ruleManager: ruleManager,
		interval:    interval,
		containers:  make(map[string]*models.ContainerStats),
		stopChan:    make(chan struct{}),
		sem:         semaphore.NewWeighted(maxConcurrentCollections),
		metrics:     &CollectorMetrics{batchSize: maxBatchSize},
	}, nil
}

func (c *Collector) Start() error {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	// Initial collection
	if err := c.collect(); err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-ticker.C:
				if err := c.collect(); err != nil {
					fmt.Printf("Error collecting stats: %v\n", err)
				}
			case <-c.stopChan:
				return
			}
		}
	}()

	return nil
}

func (c *Collector) Stop() {
	close(c.stopChan)
}

func (c *Collector) collect() error {
	startTime := time.Now()
	defer func() {
		c.metrics.mutex.Lock()
		c.metrics.totalProcessingTime += time.Since(startTime)
		c.metrics.mutex.Unlock()
	}()

	containers, err := c.dockerClient.ContainerList(c.ctx, types.ContainerListOptions{})
	if err != nil {
		c.metrics.mutex.Lock()
		c.metrics.failedCollections++
		c.metrics.mutex.Unlock()
		return fmt.Errorf("failed to list containers: %v", err)
	}

	// Create batches of containers
	batches := make([][]types.Container, 0)
	for i := 0; i < len(containers); i += c.metrics.batchSize {
		end := i + c.metrics.batchSize
		if end > len(containers) {
			end = len(containers)
		}
		batches = append(batches, containers[i:end])
	}

	// Process batches concurrently
	var wg sync.WaitGroup
	errChan := make(chan error, len(batches))

	for _, batch := range batches {
		wg.Add(1)
		go func(containers []types.Container) {
			defer wg.Done()

			if err := c.sem.Acquire(c.ctx, 1); err != nil {
				errChan <- err
				return
			}
			defer c.sem.Release(1)

			stats := make([]*models.ContainerStats, 0, len(containers))
			for _, container := range containers {
				stat, err := c.collectContainerStatsWithRetry(container.ID)
				if err != nil {
					errChan <- fmt.Errorf("error collecting stats for container %s: %v", container.ID, err)
					continue
				}
				stats = append(stats, stat)
			}

			// Batch insert into database
			if len(stats) > 0 {
				if err := c.batchInsertStats(stats); err != nil {
					errChan <- fmt.Errorf("error inserting stats batch: %v", err)
					return
				}

				// Update cache and evaluate rules
				c.mutex.Lock()
				for _, stat := range stats {
					c.containers[stat.ContainerID] = stat
					if err := c.ruleManager.EvaluateRules(stat); err != nil {
						errChan <- fmt.Errorf("error evaluating rules for container %s: %v", stat.ContainerID, err)
					}
				}
				c.mutex.Unlock()
			}
		}(batch)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errChan)

	// Collect all errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("collection errors: %v", errors)
	}

	c.metrics.mutex.Lock()
	c.metrics.totalCollections++
	c.metrics.mutex.Unlock()

	// Adjust batch size based on performance
	c.adjustBatchSize()

	return nil
}

func (c *Collector) collectContainerStatsWithRetry(containerID string) (*models.ContainerStats, error) {
	var lastErr error
	for attempt := 0; attempt < retryAttempts; attempt++ {
		stats, err := c.collectContainerStats(containerID)
		if err == nil {
			return stats, nil
		}
		lastErr = err
		time.Sleep(retryDelay)
	}
	return nil, fmt.Errorf("failed after %d attempts: %v", retryAttempts, lastErr)
}

func (c *Collector) batchInsertStats(stats []*models.ContainerStats) error {
	return database.GetDB().Transaction(func(tx *gorm.DB) error {
		for _, stat := range stats {
			if err := tx.Create(stat).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (c *Collector) adjustBatchSize() {
	c.metrics.mutex.Lock()
	defer c.metrics.mutex.Unlock()

	avgProcessingTime := c.metrics.totalProcessingTime.Seconds() / float64(c.metrics.totalCollections)
	
	// If average processing time is too high, decrease batch size
	if avgProcessingTime > 5.0 && c.metrics.batchSize > 10 {
		c.metrics.batchSize = c.metrics.batchSize * 8 / 10
	}
	
	// If average processing time is low, increase batch size
	if avgProcessingTime < 1.0 && c.metrics.batchSize < maxBatchSize {
		c.metrics.batchSize = c.metrics.batchSize * 12 / 10
		if c.metrics.batchSize > maxBatchSize {
			c.metrics.batchSize = maxBatchSize
		}
	}
}

func (c *Collector) GetMetrics() map[string]interface{} {
	c.metrics.mutex.RLock()
	defer c.metrics.mutex.RUnlock()

	return map[string]interface{}{
		"total_collections":     c.metrics.totalCollections,
		"failed_collections":    c.metrics.failedCollections,
		"avg_processing_time":   c.metrics.totalProcessingTime.Seconds() / float64(c.metrics.totalCollections),
		"current_batch_size":    c.metrics.batchSize,
		"goroutines":           runtime.NumGoroutine(),
		"max_concurrent_colls": maxConcurrentCollections,
	}
}

func (c *Collector) GetContainerStats(containerID string) (*models.ContainerStats, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	if stats, ok := c.containers[containerID]; ok {
		return stats, nil
	}
	return nil, fmt.Errorf("container stats not found")
}

func (c *Collector) CollectContainerStats() ([]*models.ContainerStats, error) {
	containers, err := c.dockerClient.ContainerList(c.ctx, types.ContainerListOptions{})
	if err != nil {
		return nil, err
	}

	stats := make([]*models.ContainerStats, 0, len(containers))
	for _, container := range containers {
		stat, err := c.collectContainerStatsWithRetry(container.ID)
		if err != nil {
			continue
		}
		stats = append(stats, stat)
	}

	return stats, nil
}

func (c *Collector) GetContainerInfo(containerID string) (*models.ContainerStats, error) {
	return c.collectContainerStatsWithRetry(containerID)
}

func (c *Collector) collectContainerStats(containerID string) (*models.ContainerStats, error) {
	resp, err := c.dockerClient.ContainerStats(c.ctx, containerID, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var stats types.StatsJSON
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, err
	}

	// Get container info for name
	info, err := c.dockerClient.ContainerInspect(c.ctx, containerID)
	if err != nil {
		return nil, err
	}

	// Calculate CPU usage percentage
	cpuPercent := calculateCPUPercentUnix(stats)
	
	// Calculate memory usage percentage
	memoryPercent := float64(stats.MemoryStats.Usage) / float64(stats.MemoryStats.Limit) * 100.0

	// Calculate total disk I/O
	var diskRead, diskWrite uint64
	for _, stat := range stats.BlkioStats.IoServiceBytesRecursive {
		switch stat.Op {
		case "Read":
			diskRead = stat.Value
		case "Write":
			diskWrite = stat.Value
		}
	}

	// Calculate total network I/O
	networkRx := calculateNetworkRx(stats.Networks)
	networkTx := calculateNetworkTx(stats.Networks)

	return &models.ContainerStats{
		ContainerID:   containerID,
		ContainerName: info.Name,
		Timestamp:     time.Now(),
		CPUPercent:    cpuPercent,
		MemoryUsage:   stats.MemoryStats.Usage,
		MemoryLimit:   stats.MemoryStats.Limit,
		MemoryPercent: memoryPercent,
		NetworkRx:     networkRx,
		NetworkTx:     networkTx,
		NetworkTotal:  networkRx + networkTx,
		BlockRead:     diskRead,
		BlockWrite:    diskWrite,
		DiskIOTotal:   diskRead + diskWrite,
	}, nil
}

func calculateCPUPercentUnix(stats types.StatsJSON) float64 {
	cpuPercent := 0.0
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage) - float64(stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage) - float64(stats.PreCPUStats.SystemUsage)

	if systemDelta > 0.0 && cpuDelta > 0.0 {
		cpuPercent = (cpuDelta / systemDelta) * float64(len(stats.CPUStats.CPUUsage.PercpuUsage)) * 100.0
	}
	return cpuPercent
}

func calculateNetworkRx(networks map[string]types.NetworkStats) uint64 {
	var rx uint64
	for _, network := range networks {
		rx += network.RxBytes
	}
	return rx
}

func calculateNetworkTx(networks map[string]types.NetworkStats) uint64 {
	var tx uint64
	for _, network := range networks {
		tx += network.TxBytes
	}
	return tx
}
