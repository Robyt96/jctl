package main

import (
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  `Commands for viewing and managing jctl configuration.`,
}

var configShowCmd = &cobra.Command{
	Use:     "show",
	Short:   "Show current configuration",
	Long:    `Display the current effective configuration with sources.`,
	Example: `  jctl config show`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Implement config display
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
}
