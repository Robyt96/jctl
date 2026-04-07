package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
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

var buildsParamsCmd = &cobra.Command{
	Use:   "params <pipeline> <build-number>",
	Short: "Get parameters for a specific build",
	Long:  "Retrieve and display the parameters that were used when a specific build was triggered.",
	Example: `  jctl builds params my-pipeline 42
  jctl builds params folder/subfolder/pipeline 15
  jctl builds params backend-service 100 --output json
  jctl builds params my-pipeline 42 --output yaml`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("requires exactly 2 arguments: pipeline name and build number\n\nUsage:\n  %s\n\nExample:\n%s", cmd.Use, cmd.Example)
		}
		if len(args) > 2 {
			return fmt.Errorf("too many arguments: expected 2, got %d\n\nUsage:\n  %s\n\nExample:\n%s", len(args), cmd.Use, cmd.Example)
		}
		return nil
	},
	RunE: runBuildsParams,
}

func init() {
	rootCmd.AddCommand(buildsCmd)
	buildsCmd.AddCommand(buildsListCmd)
	buildsCmd.AddCommand(buildsParamsCmd)
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

	// Jenkins duration is in milliseconds, convert to Go duration
	d := time.Duration(duration) * time.Millisecond

	// Format based on duration length for readability
	// Short builds: show seconds only
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		// Medium builds: show minutes and seconds
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	} else {
		// Long builds: show hours and minutes (omit seconds for brevity)
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
}

// runBuildsParams executes the builds params command
func runBuildsParams(cmd *cobra.Command, args []string) error {
	// Extract pipeline name and build number from args (already validated by Args function)
	pipelineName := args[0]
	buildNumber := args[1]

	// Validate Jenkins URL is configured
	if profile.Jenkins.URL == "" {
		return fmt.Errorf("Jenkins URL not configured. Set it via --jenkins-url flag or config file")
	}

	// Parse build number to integer - Jenkins build numbers are always positive integers
	buildNum, err := strconv.Atoi(buildNumber)
	if err != nil {
		return fmt.Errorf("invalid build number: '%s'\n\nDetails: Build number must be a positive integer\nExample: jctl builds params my-pipeline 42", buildNumber)
	}

	// Validate build number is positive (Jenkins build numbers start at 1)
	if buildNum <= 0 {
		return fmt.Errorf("invalid build number: '%s'\n\nDetails: Build number must be a positive integer\nExample: jctl builds params my-pipeline 42", buildNumber)
	}

	// Get verbose flag
	verbose, _ := cmd.Flags().GetBool("verbose")

	// Initialize auth manager
	authMgr := auth.NewManager(profile.Auth.TokenFile)

	// Create Jenkins API client
	apiClient := client.NewClient(profile.Jenkins.URL, profile.Jenkins.Timeout, authMgr, profile.Name, verbose)

	// Call API to get build information
	ctx := context.Background()
	build, err := apiClient.GetBuildInfo(ctx, pipelineName, buildNum)
	if err != nil {
		// Check if it's a 404 error
		if clientErr, ok := err.(*client.APIError); ok && clientErr.StatusCode == 404 {
			// Could be either pipeline or build not found - provide helpful message
			return fmt.Errorf("build #%d not found for pipeline '%s'\n\nDetails: Either the pipeline or build does not exist on the Jenkins server\nSuggestion: Use 'jctl builds list %s' to see available builds", buildNum, pipelineName, pipelineName)
		}
		return fmt.Errorf("failed to get build info: %w", err)
	}

	// Extract parameters from build's actions array
	// Jenkins stores parameters in the actions array, potentially across multiple action objects
	params := build.ExtractParameters()

	// Handle case where no parameters exist (build was triggered without parameters)
	if len(params) == 0 {
		fmt.Printf("No parameters were used for build #%d of pipeline %s\n", buildNum, pipelineName)
		return nil
	}

	// Format and display output based on --output flag
	if err := formatParamsOutput(params, profile.Output.Format, pipelineName, buildNum); err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	return nil
}

// formatParamsOutput formats the parameter list according to the specified format
func formatParamsOutput(params []client.Parameter, format string, pipelineName string, buildNum int) error {
	switch format {
	case "json":
		return formatParamsJSON(params)
	case "yaml":
		return formatParamsYAML(params)
	case "text":
		return formatParamsText(params, pipelineName, buildNum)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// formatParamsJSON outputs parameters in JSON format
func formatParamsJSON(params []client.Parameter) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(params)
}

// formatParamsYAML outputs parameters in YAML format
func formatParamsYAML(params []client.Parameter) error {
	encoder := yaml.NewEncoder(os.Stdout)
	defer encoder.Close()
	return encoder.Encode(params)
}

// formatParamsText outputs parameters in human-readable text format
func formatParamsText(params []client.Parameter, pipelineName string, buildNum int) error {
	// Print header
	fmt.Printf("Parameters for build #%d of pipeline %s:\n", buildNum, pipelineName)
	fmt.Println()

	// Create tabwriter for aligned columns
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	defer w.Flush()

	// Print column headers
	fmt.Fprintln(w, "NAME\tVALUE")
	fmt.Fprintln(w, "----\t-----")

	// Print each parameter
	for _, param := range params {
		fmt.Fprintf(w, "%s\t%s\n", param.Name, param.StringValue())
	}

	return nil
}
