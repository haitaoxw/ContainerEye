package commands

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/containereye/internal/api/client"
	"github.com/spf13/cobra"
)

func NewContainerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "container",
		Short: "Container management commands",
		Aliases: []string{"containers", "c"},
	}

	// Add subcommands
	cmd.AddCommand(newContainerListCommand())
	cmd.AddCommand(newContainerStatsCommand())

	return cmd
}

func newContainerListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all containers",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewClient()
			if err != nil {
				return fmt.Errorf("failed to create client: %v", err)
			}

			containers, err := c.ListContainers()
			if err != nil {
				return fmt.Errorf("failed to list containers: %v", err)
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tIMAGE\tSTATUS\tCREATED")
			
			for _, container := range containers {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					container.ContainerID[:12],
					container.Name,
					container.Image,
					container.Status,
					container.Created.Format(time.RFC3339),
				)
			}
			
			return w.Flush()
		},
	}

	return cmd
}

func newContainerStatsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats [container_id]",
		Short: "Show container statistics",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewClient()
			if err != nil {
				return fmt.Errorf("failed to create client: %v", err)
			}

			stats, err := c.GetContainerStats(args[0])
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
		},
	}

	return cmd
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
