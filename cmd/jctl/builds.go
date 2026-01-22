package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/user/jctl/internal/auth"
	"github.com/user/jctl/internal/client"
	"gopkg.in/yaml.v3"
)

var buildsCmd = &cobra.Command{
	Use:   "builds",
	Short: "Manage pipeline builds",
	Long:  `Commands for listing and viewing pipeline builds.`,
}

var buildsListCmd = &cobra.Command{
	Use:   "list <pipeline>",
	Short: "List builds for a pipeline",
	Long:  `List all builds for a specific pipeline with their status and timestamps. Pipeline can be a simple name or a folder path (e.g., folder/subfolder/pipeline).`,
	Example: `  jctl builds list my-pipeline
  jctl builds list Odino/OdinoInfrastructureProducts
  jctl builds list backend-service --output json`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("pipeline name is required\n\nUsage:\n  %s\n\nExample:\n%s", cmd.Use, cmd.Example)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBuildsList(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(buildsCmd)
	buildsCmd.AddCommand(buildsListCmd)
}

// runBuildsList executes the builds list command
func runBuildsList(cmd *cobra.Command, args []string) error {
	// Validate Jenkins URL is configured
	if profile.Jenkins.URL == "" {
		return fmt.Errorf("Jenkins URL not configured. Set it via --jenkins-url flag or config file")
	}

	// Get pipeline name from arguments (already validated by Args function)
	pipelineName := args[0]

	// Get verbose flag
	verbose, _ := cmd.Flags().GetBool("verbose")

	// Initialize auth manager
	authMgr := auth.NewManager(profile.Auth.TokenFile)

	// Create Jenkins API client
	apiClient := client.NewClient(profile.Jenkins.URL, profile.Jenkins.Timeout, authMgr, profile.Name, verbose)

	// Call API to list builds
	ctx := context.Background()
	builds, err := apiClient.ListBuilds(ctx, pipelineName)
	if err != nil {
		// Check if it's a 404 error (non-existent pipeline)
		if clientErr, ok := err.(*client.APIError); ok && clientErr.StatusCode == 404 {
			return fmt.Errorf("pipeline '%s' not found\n\nDetails: No job with name '%s' exists on the Jenkins server\nSuggestion: Use 'jctl pipelines list' to see available pipelines", pipelineName, pipelineName)
		}
		return fmt.Errorf("failed to list builds: %w", err)
	}

	// Handle empty build list
	if len(builds) == 0 {
		fmt.Printf("No builds found for pipeline: %s\n", pipelineName)
		return nil
	}

	// Format and display output based on --output flag
	if err := formatBuildsOutput(builds, profile.Output.Format, pipelineName); err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	return nil
}

// formatBuildsOutput formats the build list according to the specified format
func formatBuildsOutput(builds []client.Build, format string, pipelineName string) error {
	switch format {
	case "json":
		return formatBuildsJSON(builds)
	case "yaml":
		return formatBuildsYAML(builds)
	case "text":
		return formatBuildsText(builds, pipelineName)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// formatBuildsJSON outputs builds in JSON format
func formatBuildsJSON(builds []client.Build) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(builds)
}

// formatBuildsYAML outputs builds in YAML format
func formatBuildsYAML(builds []client.Build) error {
	encoder := yaml.NewEncoder(os.Stdout)
	defer encoder.Close()
	return encoder.Encode(builds)
}

// formatBuildsText outputs builds in human-readable text format
func formatBuildsText(builds []client.Build, pipelineName string) error {
	// Print header
	fmt.Printf("Builds for pipeline: %s\n", pipelineName)
	fmt.Println()

	// Create tabwriter for aligned columns
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	defer w.Flush()

	// Print column headers
	fmt.Fprintln(w, "BUILD\tSTATUS\tTIMESTAMP\tDURATION")
	fmt.Fprintln(w, "-----\t------\t---------\t--------")

	// Print each build
	for _, build := range builds {
		buildNum := fmt.Sprintf("#%d", build.Number)
		status := getBuildStatus(build)
		timestamp := formatBuildTimestamp(build.Timestamp)
		duration := formatBuildDuration(build.Duration, build.Building)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", buildNum, status, timestamp, duration)
	}

	return nil
}

// getBuildStatus returns a human-readable status for a build
func getBuildStatus(build client.Build) string {
	if build.Building {
		return "Building"
	}

	switch build.Result {
	case "SUCCESS":
		return "Success"
	case "FAILURE":
		return "Failed"
	case "ABORTED":
		return "Aborted"
	case "UNSTABLE":
		return "Unstable"
	case "NOT_BUILT":
		return "Not Built"
	case "":
		// Empty result typically means in progress
		return "In Progress"
	default:
		return build.Result
	}
}

// formatBuildTimestamp converts Unix timestamp (milliseconds) to human-readable format
func formatBuildTimestamp(timestamp int64) string {
	if timestamp == 0 {
		return "N/A"
	}

	// Jenkins timestamps are in milliseconds
	t := time.Unix(timestamp/1000, 0)
	return t.Format("2006-01-02 15:04:05")
}

// formatBuildDuration converts duration (milliseconds) to human-readable format
func formatBuildDuration(duration int64, building bool) string {
	if building {
		return "In Progress"
	}

	if duration == 0 {
		return "N/A"
	}

	// Jenkins duration is in milliseconds
	d := time.Duration(duration) * time.Millisecond

	// Format based on duration length
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	} else {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
}
