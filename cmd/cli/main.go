package main

import (
	"fmt"
	"os"

	"containereye/internal/cli/commands"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "containereye",
	Short: "ContainerEye CLI - A container monitoring tool",
	Long: `ContainerEye CLI is a command-line tool for monitoring Docker containers.
It provides real-time and historical statistics, alerts management, and more.`,
}

func init() {
	// Add commands
	rootCmd.AddCommand(commands.NewContainerCommand())
	rootCmd.AddCommand(commands.NewStatsCommand())
	rootCmd.AddCommand(commands.NewAlertCommand())
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
