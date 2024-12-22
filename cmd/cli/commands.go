package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"containereye/internal/models"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to ContainerEye",
	Run: func(cmd *cobra.Command, args []string) {
		username, _ := cmd.Flags().GetString("username")
		password, _ := cmd.Flags().GetString("password")
		
		// TODO: Implement login logic and save token
		token, err := authenticate(username, password)
		if err != nil {
			fmt.Printf("Login failed: %v\n", err)
			return
		}
		
		viper.Set("token", token)
		viper.WriteConfig()
		fmt.Println("Login successful")
	},
}

var containerCmd = &cobra.Command{
	Use:   "containers",
	Short: "List and manage containers",
	Run: func(cmd *cobra.Command, args []string) {
		containers, err := getContainers()
		if err != nil {
			fmt.Printf("Error getting containers: %v\n", err)
			return
		}
		
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.TabIndent)
		fmt.Fprintln(w, "ID\tNAME\tSTATUS\tCPU %\tMEM %\t")
		for _, c := range containers {
			fmt.Fprintf(w, "%s\t%s\t%s\t%.2f\t%.2f\t\n",
				c.ID[:12], c.Name, c.Status, c.CPUPercent, c.MemPercent)
		}
		w.Flush()
	},
}

var alertCmd = &cobra.Command{
	Use:   "alerts",
	Short: "Manage alerts",
	Run: func(cmd *cobra.Command, args []string) {
		alerts, err := getAlerts()
		if err != nil {
			fmt.Printf("Error getting alerts: %v\n", err)
			return
		}
		
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.TabIndent)
		fmt.Fprintln(w, "ID\tLEVEL\tCONTAINER\tMESSAGE\tSTATUS\t")
		for _, a := range alerts {
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t\n",
				a.ID, a.Level, a.ContainerName, a.Message, a.Status)
		}
		w.Flush()
	},
}

func addRuleCommands(rootCmd *cobra.Command) {
	var ruleCmd = &cobra.Command{
		Use:   "rule",
		Short: "Manage alert rules",
	}

	// List rules
	var enabledFlag *bool
	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "List alert rules",
		RunE: func(cmd *cobra.Command, args []string) error {
			rules, err := apiClient.ListRules(enabledFlag)
			if err != nil {
				return err
			}

			printRules(rules)
			return nil
		},
	}
	listCmd.Flags().BoolVar(enabledFlag, "enabled", false, "Filter by enabled status")

	// Get rule
	var getCmd = &cobra.Command{
		Use:   "get [id]",
		Short: "Get alert rule details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseUint(args[0], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid rule ID: %v", err)
			}

			rule, err := apiClient.GetRule(uint(id))
			if err != nil {
				return err
			}

			printRule(rule)
			return nil
		},
	}

	// Create rule
	var createCmd = &cobra.Command{
		Use:   "create",
		Short: "Create a new alert rule",
		RunE: func(cmd *cobra.Command, args []string) error {
			var rule models.AlertRule
			if err := json.NewDecoder(os.Stdin).Decode(&rule); err != nil {
				return fmt.Errorf("invalid rule JSON: %v", err)
			}

			if err := apiClient.CreateRule(&rule); err != nil {
				return err
			}

			fmt.Println("Rule created successfully")
			return nil
		},
	}

	// Update rule
	var updateCmd = &cobra.Command{
		Use:   "update [id]",
		Short: "Update an existing alert rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseUint(args[0], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid rule ID: %v", err)
			}

			var rule models.AlertRule
			if err := json.NewDecoder(os.Stdin).Decode(&rule); err != nil {
				return fmt.Errorf("invalid rule JSON: %v", err)
			}

			rule.ID = uint(id)
			if err := apiClient.UpdateRule(&rule); err != nil {
				return err
			}

			fmt.Println("Rule updated successfully")
			return nil
		},
	}

	// Delete rule
	var deleteCmd = &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete an alert rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseUint(args[0], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid rule ID: %v", err)
			}

			if err := apiClient.DeleteRule(uint(id)); err != nil {
				return err
			}

			fmt.Println("Rule deleted successfully")
			return nil
		},
	}

	// Enable rule
	var enableCmd = &cobra.Command{
		Use:   "enable [id]",
		Short: "Enable an alert rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseUint(args[0], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid rule ID: %v", err)
			}

			if err := apiClient.EnableRule(uint(id)); err != nil {
				return err
			}

			fmt.Println("Rule enabled successfully")
			return nil
		},
	}

	// Disable rule
	var disableCmd = &cobra.Command{
		Use:   "disable [id]",
		Short: "Disable an alert rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseUint(args[0], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid rule ID: %v", err)
			}

			if err := apiClient.DisableRule(uint(id)); err != nil {
				return err
			}

			fmt.Println("Rule disabled successfully")
			return nil
		},
	}

	// Validate rule
	var validateCmd = &cobra.Command{
		Use:   "validate",
		Short: "Validate an alert rule",
		RunE: func(cmd *cobra.Command, args []string) error {
			var rule models.AlertRule
			if err := json.NewDecoder(os.Stdin).Decode(&rule); err != nil {
				return fmt.Errorf("invalid rule JSON: %v", err)
			}

			if err := apiClient.ValidateRule(&rule); err != nil {
				return err
			}

			fmt.Println("Rule is valid")
			return nil
		},
	}

	// Import rules
	var importCmd = &cobra.Command{
		Use:   "import",
		Short: "Import alert rules from JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			var rules []models.AlertRule
			if err := json.NewDecoder(os.Stdin).Decode(&rules); err != nil {
				return fmt.Errorf("invalid rules JSON: %v", err)
			}

			if err := apiClient.ImportRules(rules); err != nil {
				return err
			}

			fmt.Printf("Successfully imported %d rules\n", len(rules))
			return nil
		},
	}

	// Export rules
	var exportCmd = &cobra.Command{
		Use:   "export",
		Short: "Export alert rules to JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			rules, err := apiClient.ExportRules()
			if err != nil {
				return err
			}

			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(rules)
		},
	}

	// Test rule
	var testCmd = &cobra.Command{
		Use:   "test",
		Short: "Test an alert rule",
		RunE: func(cmd *cobra.Command, args []string) error {
			var rule models.AlertRule
			if err := json.NewDecoder(os.Stdin).Decode(&rule); err != nil {
				return fmt.Errorf("invalid rule JSON: %v", err)
			}

			useSample, _ := cmd.Flags().GetBool("sample")
			startTime, _ := cmd.Flags().GetString("start")
			endTime, _ := cmd.Flags().GetString("end")

			var request struct {
				Rule      models.AlertRule `json:"rule"`
				StartTime *time.Time      `json:"start_time,omitempty"`
				EndTime   *time.Time      `json:"end_time,omitempty"`
				UseSample bool            `json:"use_sample"`
			}

			request.Rule = rule
			request.UseSample = useSample

			if !useSample {
				if startTime == "" || endTime == "" {
					return fmt.Errorf("start and end times are required for historical data testing")
				}

				st, err := time.Parse(time.RFC3339, startTime)
				if err != nil {
					return fmt.Errorf("invalid start time: %v", err)
				}
				request.StartTime = &st

				et, err := time.Parse(time.RFC3339, endTime)
				if err != nil {
					return fmt.Errorf("invalid end time: %v", err)
				}
				request.EndTime = &et
			}

			resp, err := apiClient.TestRule(&request)
			if err != nil {
				return err
			}

			// Print test results
			fmt.Printf("\nTest Results for Rule: %s\n", rule.Name)
			fmt.Printf("Duration: %s\n", resp.Summary.TestDuration)
			fmt.Printf("Total Alerts: %d\n", resp.Summary.TotalAlerts)
			fmt.Printf("Alerts per Hour: %.2f\n\n", resp.Summary.AlertsPerHour)

			if len(resp.Alerts) > 0 {
				fmt.Println("Alert Details:")
				fmt.Println(strings.Repeat("-", 80))
				for _, alert := range resp.Alerts {
					fmt.Printf("Container: %s\n", alert.ContainerName)
					fmt.Printf("Level: %s\n", alert.Level)
					fmt.Printf("Message: %s\n", alert.Message)
					fmt.Printf("Period: %s - %s\n", alert.StartTime.Format(time.RFC3339), alert.EndTime.Format(time.RFC3339))
					fmt.Println(strings.Repeat("-", 80))
				}
			} else {
				fmt.Println("No alerts generated during the test period.")
			}

			return nil
		},
	}

	testCmd.Flags().Bool("sample", false, "Use sample data for testing")
	testCmd.Flags().String("start", "", "Start time for historical data testing (RFC3339 format)")
	testCmd.Flags().String("end", "", "End time for historical data testing (RFC3339 format)")

	ruleCmd.AddCommand(listCmd)
	ruleCmd.AddCommand(getCmd)
	ruleCmd.AddCommand(createCmd)
	ruleCmd.AddCommand(updateCmd)
	ruleCmd.AddCommand(deleteCmd)
	ruleCmd.AddCommand(enableCmd)
	ruleCmd.AddCommand(disableCmd)
	ruleCmd.AddCommand(validateCmd)
	ruleCmd.AddCommand(importCmd)
	ruleCmd.AddCommand(exportCmd)
	ruleCmd.AddCommand(testCmd)

	rootCmd.AddCommand(ruleCmd)
}

func printRules(rules []models.AlertRule) {
	fmt.Printf("%-5s %-30s %-15s %-10s %-10s %-10s %s\n",
		"ID", "Name", "Metric", "Operator", "Threshold", "Duration", "Enabled")
	fmt.Println(strings.Repeat("-", 100))

	for _, rule := range rules {
		fmt.Printf("%-5d %-30s %-15s %-10s %-10.2f %-10d %v\n",
			rule.ID, rule.Name, rule.Metric, rule.Operator,
			rule.Threshold, rule.Duration, rule.IsEnabled)
	}
}

func printRule(rule *models.AlertRule) {
	fmt.Printf("ID:          %d\n", rule.ID)
	fmt.Printf("Name:        %s\n", rule.Name)
	fmt.Printf("Description: %s\n", rule.Description)
	fmt.Printf("Metric:      %s\n", rule.Metric)
	fmt.Printf("Operator:    %s\n", rule.Operator)
	fmt.Printf("Threshold:   %.2f\n", rule.Threshold)
	fmt.Printf("Duration:    %d seconds\n", rule.Duration)
	fmt.Printf("Level:       %s\n", rule.Level)
	fmt.Printf("Enabled:     %v\n", rule.IsEnabled)
	if rule.ContainerID != "" {
		fmt.Printf("Container:   %s\n", rule.ContainerID)
	}
	if rule.ContainerName != "" {
		fmt.Printf("Container:   %s\n", rule.ContainerName)
	}
	fmt.Printf("Created At:  %s\n", rule.CreatedAt.Format(time.RFC3339))
	fmt.Printf("Updated At:  %s\n", rule.UpdatedAt.Format(time.RFC3339))
}

// Report generation commands
var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate and manage reports",
}

var generateReportCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a new report",
	RunE: func(cmd *cobra.Command, args []string) error {
		reportType, _ := cmd.Flags().GetString("type")
		startTime, _ := cmd.Flags().GetString("start")
		endTime, _ := cmd.Flags().GetString("end")
		email, _ := cmd.Flags().GetString("email")
		
		// Parse time strings
		st, err := time.Parse(time.RFC3339, startTime)
		if err != nil {
			return fmt.Errorf("invalid start time: %v", err)
		}
		
		et, err := time.Parse(time.RFC3339, endTime)
		if err != nil {
			return fmt.Errorf("invalid end time: %v", err)
		}
		
		// Generate and send report
		err = apiClient.GenerateReport(reportType, st, et, email)
		if err != nil {
			return fmt.Errorf("failed to generate report: %v", err)
		}
		
		fmt.Printf("Report generated and sent to %s\n", email)
		return nil
	},
}

var scheduleReportCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Schedule periodic reports",
	RunE: func(cmd *cobra.Command, args []string) error {
		reportType, _ := cmd.Flags().GetString("type")
		schedule, _ := cmd.Flags().GetString("schedule")
		email, _ := cmd.Flags().GetString("email")
		
		err := apiClient.ScheduleReport(reportType, schedule, email)
		if err != nil {
			return fmt.Errorf("failed to schedule report: %v", err)
		}
		
		fmt.Printf("Report scheduled: %s report will be sent to %s %s\n", 
			reportType, email, schedule)
		return nil
	},
}

var listReportsCmd = &cobra.Command{
	Use:   "list",
	Short: "List scheduled reports",
	RunE: func(cmd *cobra.Command, args []string) error {
		reports, err := apiClient.ListScheduledReports()
		if err != nil {
			return fmt.Errorf("failed to list reports: %v", err)
		}
		
		if len(reports) == 0 {
			fmt.Println("No scheduled reports found")
			return nil
		}
		
		fmt.Println("Scheduled Reports:")
		fmt.Println("------------------")
		for _, r := range reports {
			fmt.Printf("ID: %d\n", r.ID)
			fmt.Printf("Type: %s\n", r.Type)
			fmt.Printf("Schedule: %s\n", r.Schedule)
			fmt.Printf("Email: %s\n", r.Email)
			fmt.Println("------------------")
		}
		
		return nil
	},
}

var deleteReportCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a scheduled report",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetInt("id")
		
		err := apiClient.DeleteScheduledReport(id)
		if err != nil {
			return fmt.Errorf("failed to delete report: %v", err)
		}
		
		fmt.Printf("Scheduled report %d deleted\n", id)
		return nil
	},
}

func init() {
	loginCmd.Flags().StringP("username", "u", "", "Username")
	loginCmd.Flags().StringP("password", "p", "", "Password")
	
	containerCmd.AddCommand(&cobra.Command{
		Use:   "stats [container-id]",
		Short: "Show detailed stats for a container",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 {
				fmt.Println("Container ID is required")
				return
			}
			stats, err := getContainerStats(args[0])
			if err != nil {
				fmt.Printf("Error getting container stats: %v\n", err)
				return
			}
			printContainerStats(stats)
		},
	})
	
	alertCmd.AddCommand(&cobra.Command{
		Use:   "acknowledge [alert-id]",
		Short: "Acknowledge an alert",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 {
				fmt.Println("Alert ID is required")
				return
			}
			err := acknowledgeAlert(args[0])
			if err != nil {
				fmt.Printf("Error acknowledging alert: %v\n", err)
				return
			}
			fmt.Println("Alert acknowledged successfully")
		},
	})
	
	addRuleCommands(rootCmd)
	
	// Report command flags
	generateReportCmd.Flags().String("type", "daily", "Report type (daily/weekly)")
	generateReportCmd.Flags().String("start", "", "Start time (RFC3339 format)")
	generateReportCmd.Flags().String("end", "", "End time (RFC3339 format)")
	generateReportCmd.Flags().String("email", "", "Email address to send report to")
	
	scheduleReportCmd.Flags().String("type", "daily", "Report type (daily/weekly)")
	scheduleReportCmd.Flags().String("schedule", "@daily", "Schedule (cron expression)")
	scheduleReportCmd.Flags().String("email", "", "Email address to send report to")
	
	deleteReportCmd.Flags().Int("id", 0, "ID of the scheduled report to delete")
	
	reportCmd.AddCommand(generateReportCmd)
	reportCmd.AddCommand(scheduleReportCmd)
	reportCmd.AddCommand(listReportsCmd)
	reportCmd.AddCommand(deleteReportCmd)
	
	rootCmd.AddCommand(reportCmd)
}

// API client functions
func authenticate(username, password string) (string, error) {
	// TODO: Implement API call
	return "", nil
}

func getContainers() ([]Container, error) {
	// TODO: Implement API call
	return nil, nil
}

func getContainerStats(id string) (*ContainerStats, error) {
	// TODO: Implement API call
	return nil, nil
}

func getAlerts() ([]Alert, error) {
	// TODO: Implement API call
	return nil, nil
}

func acknowledgeAlert(id string) error {
	// TODO: Implement API call
	return nil
}
