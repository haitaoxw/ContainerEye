package report

import (
	"bytes"
	"fmt"
	"html/template"
	"time"
	"sort"
	
	"github.com/jordan-wright/email"
	"containereye/internal/models"
	"containereye/internal/database"
)

type ReportGenerator struct {
	db        *database.DB
	templates map[string]*template.Template
}

type ReportData struct {
	StartTime     time.Time
	EndTime       time.Time
	AlertSummary  AlertSummary
	TopContainers []ContainerSummary
	Trends        TrendData
}

type AlertSummary struct {
	TotalAlerts     int
	CriticalAlerts  int
	WarningAlerts   int
	InfoAlerts      int
	TopRules        []RuleSummary
}

type RuleSummary struct {
	RuleName    string
	AlertCount  int
	Level       string
	TopTargets  []string
}

type ContainerSummary struct {
	ContainerID   string
	ContainerName string
	AlertCount    int
	CpuAvg       float64
	MemAvg       float64
	DiskAvg      float64
	NetAvg       float64
}

type TrendData struct {
	CpuTrend    []TimeSeriesPoint
	MemoryTrend []TimeSeriesPoint
	DiskTrend   []TimeSeriesPoint
	NetTrend    []TimeSeriesPoint
}

type TimeSeriesPoint struct {
	Timestamp time.Time
	Value     float64
}

func NewReportGenerator(db *database.DB) (*ReportGenerator, error) {
	templates := make(map[string]*template.Template)
	
	// Load HTML templates
	daily, err := template.ParseFiles("templates/daily_report.html")
	if err != nil {
		return nil, fmt.Errorf("failed to load daily report template: %v", err)
	}
	templates["daily"] = daily
	
	weekly, err := template.ParseFiles("templates/weekly_report.html")
	if err != nil {
		return nil, fmt.Errorf("failed to load weekly report template: %v", err)
	}
	templates["weekly"] = weekly
	
	return &ReportGenerator{
		db:        db,
		templates: templates,
	}, nil
}

func (g *ReportGenerator) GenerateReport(reportType string, startTime, endTime time.Time) (*email.Email, error) {
	// Get report data
	data, err := g.collectReportData(startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to collect report data: %v", err)
	}
	
	// Get template
	tmpl, ok := g.templates[reportType]
	if !ok {
		return nil, fmt.Errorf("unknown report type: %s", reportType)
	}
	
	// Generate HTML
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %v", err)
	}
	
	// Create email
	e := &email.Email{
		To:      []string{},  // Will be set by caller
		From:    "ContainerEye <noreply@containereye.io>",
		Subject: fmt.Sprintf("ContainerEye %s Report (%s - %s)", 
			reportType, 
			startTime.Format("2006-01-02"), 
			endTime.Format("2006-01-02")),
		HTML:    buf.Bytes(),
	}
	
	return e, nil
}

func (g *ReportGenerator) collectReportData(startTime, endTime time.Time) (*ReportData, error) {
	data := &ReportData{
		StartTime: startTime,
		EndTime:   endTime,
	}
	
	// Get alerts in time range
	var alerts []models.Alert
	if err := g.db.Where("created_at BETWEEN ? AND ?", startTime, endTime).
		Find(&alerts).Error; err != nil {
		return nil, err
	}
	
	// Process alerts
	data.AlertSummary = g.processAlerts(alerts)
	
	// Get container stats
	var stats []models.ContainerStats
	if err := g.db.Where("timestamp BETWEEN ? AND ?", startTime, endTime).
		Find(&stats).Error; err != nil {
		return nil, err
	}
	
	// Process container stats
	data.TopContainers = g.processContainerStats(stats)
	data.Trends = g.calculateTrends(stats)
	
	return data, nil
}

func (g *ReportGenerator) processAlerts(alerts []models.Alert) AlertSummary {
	summary := AlertSummary{}
	ruleAlerts := make(map[string]*RuleSummary)
	
	for _, alert := range alerts {
		summary.TotalAlerts++
		switch alert.Level {
		case "CRITICAL":
			summary.CriticalAlerts++
		case "WARNING":
			summary.WarningAlerts++
		case "INFO":
			summary.InfoAlerts++
		}
		
		// Process rule statistics
		if rs, ok := ruleAlerts[alert.RuleName]; ok {
			rs.AlertCount++
			// Add target if not already in top targets
			found := false
			for _, t := range rs.TopTargets {
				if t == alert.ContainerName {
					found = true
					break
				}
			}
			if !found && len(rs.TopTargets) < 5 {
				rs.TopTargets = append(rs.TopTargets, alert.ContainerName)
			}
		} else {
			ruleAlerts[alert.RuleName] = &RuleSummary{
				RuleName:   alert.RuleName,
				AlertCount: 1,
				Level:      alert.Level,
				TopTargets: []string{alert.ContainerName},
			}
		}
	}
	
	// Convert map to slice and sort
	for _, rs := range ruleAlerts {
		summary.TopRules = append(summary.TopRules, *rs)
	}
	sort.Slice(summary.TopRules, func(i, j int) bool {
		return summary.TopRules[i].AlertCount > summary.TopRules[j].AlertCount
	})
	
	// Keep only top 10 rules
	if len(summary.TopRules) > 10 {
		summary.TopRules = summary.TopRules[:10]
	}
	
	return summary
}

func (g *ReportGenerator) processContainerStats(stats []models.ContainerStats) []ContainerSummary {
	containers := make(map[string]*ContainerSummary)
	
	for _, stat := range stats {
		if cs, ok := containers[stat.ContainerID]; ok {
			cs.CpuAvg += stat.CPUUsage
			cs.MemAvg += stat.MemoryUsage
			cs.DiskAvg += stat.DiskUsage
			cs.NetAvg += stat.NetworkUsage
		} else {
			containers[stat.ContainerID] = &ContainerSummary{
				ContainerID:   stat.ContainerID,
				ContainerName: stat.ContainerName,
				CpuAvg:       stat.CPUUsage,
				MemAvg:       stat.MemoryUsage,
				DiskAvg:      stat.DiskUsage,
				NetAvg:       stat.NetworkUsage,
			}
		}
	}
	
	// Calculate averages
	for _, cs := range containers {
		count := float64(len(stats))
		cs.CpuAvg /= count
		cs.MemAvg /= count
		cs.DiskAvg /= count
		cs.NetAvg /= count
	}
	
	// Convert map to slice and sort by resource usage
	var result []ContainerSummary
	for _, cs := range containers {
		result = append(result, *cs)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CpuAvg > result[j].CpuAvg
	})
	
	// Keep only top 10 containers
	if len(result) > 10 {
		result = result[:10]
	}
	
	return result
}

func (g *ReportGenerator) calculateTrends(stats []models.ContainerStats) TrendData {
	trends := TrendData{
		CpuTrend:    make([]TimeSeriesPoint, 0),
		MemoryTrend: make([]TimeSeriesPoint, 0),
		DiskTrend:   make([]TimeSeriesPoint, 0),
		NetTrend:    make([]TimeSeriesPoint, 0),
	}
	
	// Group stats by timestamp
	timePoints := make(map[time.Time]struct{
		cpu, mem, disk, net float64
		count              int
	})
	
	for _, stat := range stats {
		rounded := stat.Timestamp.Truncate(time.Hour)
		point := timePoints[rounded]
		point.cpu += stat.CPUUsage
		point.mem += stat.MemoryUsage
		point.disk += stat.DiskUsage
		point.net += stat.NetworkUsage
		point.count++
		timePoints[rounded] = point
	}
	
	// Convert to time series
	var times []time.Time
	for t := range timePoints {
		times = append(times, t)
	}
	sort.Slice(times, func(i, j int) bool {
		return times[i].Before(times[j])
	})
	
	for _, t := range times {
		point := timePoints[t]
		count := float64(point.count)
		
		trends.CpuTrend = append(trends.CpuTrend, TimeSeriesPoint{
			Timestamp: t,
			Value:     point.cpu / count,
		})
		trends.MemoryTrend = append(trends.MemoryTrend, TimeSeriesPoint{
			Timestamp: t,
			Value:     point.mem / count,
		})
		trends.DiskTrend = append(trends.DiskTrend, TimeSeriesPoint{
			Timestamp: t,
			Value:     point.disk / count,
		})
		trends.NetTrend = append(trends.NetTrend, TimeSeriesPoint{
			Timestamp: t,
			Value:     point.net / count,
		})
	}
	
	return trends
}
