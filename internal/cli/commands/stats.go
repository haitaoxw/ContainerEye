package commands

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"containereye/internal/api/client"
	"github.com/spf13/cobra"
)

func NewStatsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Container statistics commands",
		Aliases: []string{"stat", "s"},
	}

	// Add subcommands
	cmd.AddCommand(newStatsShowCommand())
	cmd.AddCommand(newStatsHistoryCommand())
	cmd.AddCommand(newStatsExportCommand())

	return cmd
}

func newStatsShowCommand() *cobra.Command {
	var watch bool

	cmd := &cobra.Command{
		Use:   "show [container_id]",
		Short: "Show real-time container statistics",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewClient()
			if err != nil {
				return fmt.Errorf("failed to create client: %v", err)
			}

			if watch {
				ticker := time.NewTicker(2 * time.Second)
				defer ticker.Stop()

				for {
					if err := displayStats(c, args[0]); err != nil {
						return err
					}
					<-ticker.C
					fmt.Print("\033[H\033[2J") // Clear screen
				}
			}

			return displayStats(c, args[0])
		},
	}

	cmd.Flags().BoolVarP(&watch, "watch", "w", false, "Watch statistics in real-time")
	return cmd
}

func newStatsHistoryCommand() *cobra.Command {
	var (
		from  string
		to    string
		limit int
	)

	cmd := &cobra.Command{
		Use:   "history [container_id]",
		Short: "Show historical container statistics",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewClient()
			if err != nil {
				return fmt.Errorf("failed to create client: %v", err)
			}

			var fromTime, toTime *time.Time
			if from != "" {
				t, err := time.Parse(time.RFC3339, from)
				if err != nil {
					return fmt.Errorf("invalid from time: %v", err)
				}
				fromTime = &t
			}
			if to != "" {
				t, err := time.Parse(time.RFC3339, to)
				if err != nil {
					return fmt.Errorf("invalid to time: %v", err)
				}
				toTime = &t
			}

			stats, err := c.GetContainerStatsHistory(args[0], fromTime, toTime, limit)
			if err != nil {
				return fmt.Errorf("failed to get container stats history: %v", err)
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "TIMESTAMP\tCPU %\tMEM USAGE\tMEM %\tNET I/O\tBLOCK I/O")
			
			for _, stat := range stats {
				fmt.Fprintf(w, "%s\t%.2f%%\t%s\t%.2f%%\t%s\t%s\n",
					stat.Timestamp.Format(time.RFC3339),
					stat.CPUPercent,
					formatBytes(stat.MemoryUsage),
					stat.MemoryPercent,
					fmt.Sprintf("%s / %s", formatBytes(stat.NetworkRx), formatBytes(stat.NetworkTx)),
					fmt.Sprintf("%s / %s", formatBytes(stat.BlockRead), formatBytes(stat.BlockWrite)),
				)
			}
			
			return w.Flush()
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "Start time (RFC3339 format)")
	cmd.Flags().StringVar(&to, "to", "", "End time (RFC3339 format)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Limit the number of records")

	return cmd
}

func newStatsExportCommand() *cobra.Command {
	var (
		from   string
		to     string
		format string
		output string
	)

	cmd := &cobra.Command{
		Use:   "export [container_id]",
		Short: "Export container statistics",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewClient()
			if err != nil {
				return fmt.Errorf("failed to create client: %v", err)
			}

			var fromTime, toTime *time.Time
			if from != "" {
				t, err := time.Parse(time.RFC3339, from)
				if err != nil {
					return fmt.Errorf("invalid from time: %v", err)
				}
				fromTime = &t
			}
			if to != "" {
				t, err := time.Parse(time.RFC3339, to)
				if err != nil {
					return fmt.Errorf("invalid to time: %v", err)
				}
				toTime = &t
			}

			if err := c.ExportContainerStats(args[0], fromTime, toTime, format, output); err != nil {
				return fmt.Errorf("failed to export container stats: %v", err)
			}

			fmt.Printf("Statistics exported to %s\n", output)
			return nil
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "Start time (RFC3339 format)")
	cmd.Flags().StringVar(&to, "to", "", "End time (RFC3339 format)")
	cmd.Flags().StringVar(&format, "format", "csv", "Export format (csv/json)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file")
	cmd.MarkFlagRequired("output")

	return cmd
}

func displayStats(c *client.Client, containerID string) error {
	stats, err := c.GetContainerStats(containerID)
	if err != nil {
		return fmt.Errorf("failed to get container stats: %v", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "TIMESTAMP\tCPU %\tMEM USAGE\tMEM %\tNET I/O\tBLOCK I/O")
	
	fmt.Fprintf(w, "%s\t%.2f%%\t%s\t%.2f%%\t%s\t%s\n",
		stats.Timestamp.Format(time.RFC3339),
		stats.CPUPercent,
		formatBytes(stats.MemoryUsage),
		stats.MemoryPercent,
		fmt.Sprintf("%s / %s", formatBytes(stats.NetworkRx), formatBytes(stats.NetworkTx)),
		fmt.Sprintf("%s / %s", formatBytes(stats.BlockRead), formatBytes(stats.BlockWrite)),
	)
	
	return w.Flush()
}
