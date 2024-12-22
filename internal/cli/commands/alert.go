package commands

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/containereye/internal/api/client"
	"github.com/spf13/cobra"
)

func NewAlertCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "alert",
		Short:   "Alert management commands",
		Aliases: []string{"alerts", "a"},
	}

	// Add subcommands
	cmd.AddCommand(newAlertListCommand())
	cmd.AddCommand(newAlertAcknowledgeCommand())
	cmd.AddCommand(newAlertResolveCommand())

	return cmd
}

func newAlertListCommand() *cobra.Command {
	var (
		status string
		level  string
	)

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List alerts",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewClient()
			if err != nil {
				return fmt.Errorf("failed to create client: %v", err)
			}

			alerts, err := c.ListAlerts(status, level)
			if err != nil {
				return fmt.Errorf("failed to list alerts: %v", err)
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "ID\tCONTAINER\tLEVEL\tMETRIC\tVALUE\tSTATUS\tTIME")
			
			for _, alert := range alerts {
				fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%.2f\t%s\t%s\n",
					alert.ID,
					alert.ContainerName,
					alert.Level,
					alert.Metric,
					alert.Value,
					alert.Status,
					alert.StartTime.Format(time.RFC3339),
				)
			}
			
			return w.Flush()
		},
	}

	cmd.Flags().StringVar(&status, "status", "", "Filter by alert status (pending/active/acknowledged/resolved)")
	cmd.Flags().StringVar(&level, "level", "", "Filter by alert level (info/warning/critical)")

	return cmd
}

func newAlertAcknowledgeCommand() *cobra.Command {
	var comment string

	cmd := &cobra.Command{
		Use:     "acknowledge [alert_id]",
		Short:   "Acknowledge an alert",
		Aliases: []string{"ack"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewClient()
			if err != nil {
				return fmt.Errorf("failed to create client: %v", err)
			}

			if err := c.AcknowledgeAlert(args[0], comment); err != nil {
				return fmt.Errorf("failed to acknowledge alert: %v", err)
			}

			fmt.Printf("Alert %s acknowledged\n", args[0])
			return nil
		},
	}

	cmd.Flags().StringVar(&comment, "comment", "", "Add a comment to the acknowledgment")
	return cmd
}

func newAlertResolveCommand() *cobra.Command {
	var comment string

	cmd := &cobra.Command{
		Use:   "resolve [alert_id]",
		Short: "Resolve an alert",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewClient()
			if err != nil {
				return fmt.Errorf("failed to create client: %v", err)
			}

			if err := c.ResolveAlert(args[0], comment); err != nil {
				return fmt.Errorf("failed to resolve alert: %v", err)
			}

			fmt.Printf("Alert %s resolved\n", args[0])
			return nil
		},
	}

	cmd.Flags().StringVar(&comment, "comment", "", "Add a comment to the resolution")
	return cmd
}
