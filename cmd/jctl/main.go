package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/user/jctl/internal/config"
)

var (
	version = "0.1.0"
	cfg     *config.Config
	profile *config.Profile
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "jctl",
	Short: "Jenkins Control Tool - CLI for interacting with Jenkins CI/CD servers",
	Long: `jctl is a command-line interface tool for interacting with Jenkins CI/CD servers.
It enables developers to perform common Jenkins operations from the terminal, including
managing pipelines, viewing build information, accessing logs, and triggering builds.`,
	Version: version,
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().String("profile", "", "Profile to use (default: default profile from config)")
	rootCmd.PersistentFlags().String("jenkins-url", "", "Jenkins server URL")
	rootCmd.PersistentFlags().String("config", "", "Config file path (default: ~/.jctl/config.yaml)")
	rootCmd.PersistentFlags().String("output", "text", "Output format (text, json, yaml)")
	rootCmd.PersistentFlags().Bool("verbose", false, "Enable verbose logging")
	rootCmd.PersistentFlags().Duration("timeout", 0, "Request timeout duration")

	// Load configuration before executing commands
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	// Get CLI flag values
	configPath, _ := rootCmd.PersistentFlags().GetString("config")
	profileName, _ := rootCmd.PersistentFlags().GetString("profile")
	jenkinsURL, _ := rootCmd.PersistentFlags().GetString("jenkins-url")
	outputFormat, _ := rootCmd.PersistentFlags().GetString("output")
	timeout, _ := rootCmd.PersistentFlags().GetDuration("timeout")

	// Build CLI overrides profile
	profileOverrides := &config.Profile{}
	if jenkinsURL != "" {
		profileOverrides.Jenkins.URL = jenkinsURL
	}
	if timeout != 0 {
		profileOverrides.Jenkins.Timeout = timeout
	}
	if outputFormat != "" {
		profileOverrides.Output.Format = outputFormat
	}

	// Load configuration with precedence: CLI > env > file > defaults
	var err error
	cfg, profile, err = config.LoadWithOverrides(configPath, profileName, profileOverrides)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}
}
