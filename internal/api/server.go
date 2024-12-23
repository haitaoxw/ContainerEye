package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
	
	"containereye/internal/alert"
	"containereye/internal/auth"
	"containereye/internal/database"
	"containereye/internal/models"
	"containereye/internal/monitor"
	
	"github.com/gin-gonic/gin"
)

type Server struct {
	collector    *monitor.Collector
	alertManager *alert.AlertManager
	ruleManager  *alert.RuleManager
	router      *gin.Engine
}

func NewServer(collector *monitor.Collector, alertManager *alert.AlertManager, ruleManager *alert.RuleManager) *Server {
	server := &Server{
		collector:    collector,
		alertManager: alertManager,
		ruleManager:  ruleManager,
		router:      gin.Default(),
	}
	
	server.setupRoutes()
	return server
}

func (s *Server) setupRoutes() {
	// Public routes
	s.router.POST("/api/v1/auth/login", s.login)
	s.router.POST("/api/v1/auth/register", s.register)
	
	// Protected routes (require authentication)
	api := s.router.Group("/api/v1")
	api.Use(auth.AuthMiddleware())
	
	// Container monitoring endpoints
	api.GET("/containers", s.listContainers)
	api.GET("/containers/:id/stats", s.getContainerStats)
	
	// Alert management endpoints
	api.GET("/alerts", s.listAlerts)
	api.POST("/alerts", auth.RequireRole(models.RoleAdmin, models.RoleUser), s.createAlert)
	api.PUT("/alerts/:id/acknowledge", auth.RequireRole(models.RoleAdmin, models.RoleUser), s.acknowledgeAlert)
	api.PUT("/alerts/:id/resolve", auth.RequireRole(models.RoleAdmin, models.RoleUser), s.resolveAlert)
	
	// Rule management endpoints
	rules := api.Group("/rules")
	{
		rules.GET("", s.listRules)
		rules.GET("/:id", s.getRule)
		rules.POST("", auth.RequireRole(models.RoleAdmin), s.createRule)
		rules.PUT("/:id", auth.RequireRole(models.RoleAdmin), s.updateRule)
		rules.DELETE("/:id", auth.RequireRole(models.RoleAdmin), s.deleteRule)
		rules.PUT("/:id/enable", auth.RequireRole(models.RoleAdmin), s.enableRule)
		rules.PUT("/:id/disable", auth.RequireRole(models.RoleAdmin), s.disableRule)
		rules.POST("/validate", auth.RequireRole(models.RoleAdmin), s.validateRule)
		rules.POST("/import", auth.RequireRole(models.RoleAdmin), s.importRules)
		rules.GET("/export", auth.RequireRole(models.RoleAdmin), s.exportRules)
		rules.POST("/test", auth.RequireRole(models.RoleAdmin), s.testRule)
	}
	
	// User management endpoints
	admin := api.Group("/admin")
	admin.Use(auth.RequireRole(models.RoleAdmin))
	admin.GET("/users", s.listUsers)
	admin.POST("/users", s.createUser)
	admin.PUT("/users/:id", s.updateUser)
	admin.DELETE("/users/:id", s.deleteUser)
}

func (s *Server) Start(port int) error {
	return s.router.Run(fmt.Sprintf(":%d", port))
}

func (s *Server) listContainers(c *gin.Context) {
	stats, err := s.collector.CollectContainerStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, stats)
}

func (s *Server) getContainerStats(c *gin.Context) {
	containerID := c.Param("id")
	var stats []models.ContainerStats

	query := database.GetDB().Where("container_id = ?", containerID)

	// Add time range filter if provided
	if startTime := c.Query("start"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			query = query.Where("timestamp >= ?", t)
		}
	}
	if endTime := c.Query("end"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			query = query.Where("timestamp <= ?", t)
		}
	}

	// Add limit if provided
	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			query = query.Limit(l)
		}
	}

	// Execute query
	if err := query.Order("timestamp desc").Find(&stats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch container stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (s *Server) listAlerts(c *gin.Context) {
	// TODO: Implement alert listing with filtering and pagination
	c.JSON(http.StatusOK, []models.Alert{})
}

func (s *Server) createAlert(c *gin.Context) {
	var alert models.Alert
	if err := c.BindJSON(&alert); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// TODO: Save alert to database
	
	// Send notifications based on alert level
	if alert.Level == models.AlertLevelCritical {
		go s.alertManager.SendSlackAlert(&alert)
		go s.alertManager.SendEmailAlert(&alert)
	} else if alert.Level == models.AlertLevelWarning {
		go s.alertManager.SendSlackAlert(&alert)
	}
	
	c.JSON(http.StatusCreated, alert)
}

func (s *Server) acknowledgeAlert(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.alertManager.AcknowledgeAlert(c.Param("id"), req.UserID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

func (s *Server) resolveAlert(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.alertManager.ResolveAlert(c.Param("id"), req.UserID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

func (s *Server) login(c *gin.Context) {
	var loginReq struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&loginReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	var user models.User
	if err := database.GetDB().Where("username = ?", loginReq.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	
	if !user.CheckPassword(loginReq.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	
	token, err := auth.GenerateToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"token": token})
}

func (s *Server) register(c *gin.Context) {
	// TODO: Implement user registration
}

func (s *Server) listUsers(c *gin.Context) {
	// TODO: Implement user listing
}

func (s *Server) createUser(c *gin.Context) {
	// TODO: Implement user creation
}

func (s *Server) updateUser(c *gin.Context) {
	// TODO: Implement user update
}

func (s *Server) deleteUser(c *gin.Context) {
	// TODO: Implement user deletion
}

// Rule management handlers
func (s *Server) listRules(c *gin.Context) {
	enabled := c.Query("enabled")
	var enabledPtr *bool
	if enabled != "" {
		enabledBool := enabled == "true"
		enabledPtr = &enabledBool
	}

	rules, err := s.ruleManager.ListRules(enabledPtr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, rules)
}

func (s *Server) getRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule ID"})
		return
	}

	rule, err := s.ruleManager.GetRule(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
		return
	}

	c.JSON(http.StatusOK, rule)
}

func (s *Server) createRule(c *gin.Context) {
	var rule models.AlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate rule
	if err := s.validateRuleFields(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.ruleManager.CreateRule(&rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, rule)
}

func (s *Server) updateRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule ID"})
		return
	}

	var rule models.AlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rule.ID = uint(id)
	
	// Validate rule
	if err := s.validateRuleFields(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.ruleManager.UpdateRule(&rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, rule)
}

func (s *Server) deleteRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule ID"})
		return
	}

	if err := s.ruleManager.DeleteRule(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "rule deleted successfully"})
}

func (s *Server) enableRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule ID"})
		return
	}

	if err := s.ruleManager.EnableRule(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "rule enabled successfully"})
}

func (s *Server) disableRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule ID"})
		return
	}

	if err := s.ruleManager.DisableRule(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "rule disabled successfully"})
}

func (s *Server) validateRule(c *gin.Context) {
	var rule models.AlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.validateRuleFields(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "rule is valid"})
}

func (s *Server) validateRuleFields(rule *models.AlertRule) error {
	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}

	if !isValidMetric(rule.Metric) {
		return fmt.Errorf("invalid metric: %s", rule.Metric)
	}

	if !isValidOperator(rule.Operator) {
		return fmt.Errorf("invalid operator: %s", rule.Operator)
	}

	if !isValidAlertLevel(rule.Level) {
		return fmt.Errorf("invalid alert level: %s", rule.Level)
	}

	if rule.Duration <= 0 {
		return fmt.Errorf("duration must be positive")
	}

	return nil
}

func (s *Server) importRules(c *gin.Context) {
	var rules []models.AlertRule
	if err := c.ShouldBindJSON(&rules); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	for _, rule := range rules {
		if err := s.validateRuleFields(&rule); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid rule '%s': %v", rule.Name, err)})
			return
		}
	}

	for _, rule := range rules {
		if err := s.ruleManager.CreateRule(&rule); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to import rule '%s': %v", rule.Name, err)})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("successfully imported %d rules", len(rules))})
}

func (s *Server) exportRules(c *gin.Context) {
	rules, err := s.ruleManager.ListRules(nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, rules)
}

func (s *Server) testRule(c *gin.Context) {
	var request struct {
		Rule      models.AlertRule `json:"rule"`
		StartTime *time.Time      `json:"start_time,omitempty"`
		EndTime   *time.Time      `json:"end_time,omitempty"`
		UseSample bool            `json:"use_sample"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate rule
	if err := s.validateRuleFields(&request.Rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var alerts []models.Alert
	var err error

	if request.UseSample {
		alerts, err = s.ruleManager.TestRuleWithSampleData(&request.Rule)
	} else {
		if request.StartTime == nil || request.EndTime == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "start_time and end_time are required for historical data testing"})
			return
		}
		alerts, err = s.ruleManager.TestRule(&request.Rule, *request.StartTime, *request.EndTime)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"rule":   request.Rule,
		"alerts": alerts,
		"summary": gin.H{
			"total_alerts":    len(alerts),
			"test_duration":   request.EndTime.Sub(*request.StartTime).String(),
			"alerts_per_hour": float64(len(alerts)) / request.EndTime.Sub(*request.StartTime).Hours(),
		},
	})
}

// Helper functions
func isValidMetric(metric models.Metric) bool {
	validMetrics := map[models.Metric]bool{
		models.MetricCPUUsage:    true,
		models.MetricMemoryUsage: true,
		models.MetricDiskIO:      true,
		models.MetricNetworkIO:   true,
	}
	return validMetrics[metric]
}

func isValidOperator(operator models.Operator) bool {
	validOperators := map[models.Operator]bool{
		models.OperatorGT:  true,
		models.OperatorLT:  true,
		models.OperatorGTE: true,
		models.OperatorLTE: true,
		models.OperatorEQ:  true,
	}
	return validOperators[operator]
}

func isValidAlertLevel(level models.AlertLevel) bool {
	validLevels := map[models.AlertLevel]bool{
		models.AlertLevelInfo:     true,
		models.AlertLevelWarning:  true,
		models.AlertLevelCritical: true,
	}
	return validLevels[level]
}
