package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/user/jctl/internal/auth"
	"github.com/user/jctl/internal/client"
	"gopkg.in/yaml.v3"
)

var triggerCmd = &cobra.Command{
	Use:   "trigger <pipeline>",
	Short: "Trigger a pipeline build",
	Long:  `Trigger a new build for a specific pipeline with optional parameters. Pipeline can be a simple name or a folder path (e.g., folder/subfolder/pipeline).`,
	Example: `  jctl trigger my-pipeline
  jctl trigger backend-service --param ENV=staging --param VERSION=1.2.3
  jctl trigger folder/subfolder/pipeline --param BRANCH=main`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("pipeline name is required\n\nUsage:\n  %s\n\nExample:\n%s", cmd.Use, cmd.Example)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTrigger(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(triggerCmd)

	// Add --param flag for build parameters
	triggerCmd.Flags().StringArrayP("param", "p", []string{}, "Build parameters in key=value format")
	// Add --follow flag for progressive log streaming
	triggerCmd.Flags().BoolP("follow", "f", false, "Follow log output after triggering (stream logs as build progresses)")
}

// runTrigger executes the trigger command
func runTrigger(cmd *cobra.Command, args []string) error {
	// Validate Jenkins URL is configured
	if profile.Jenkins.URL == "" {
		return fmt.Errorf("Jenkins URL not configured. Set it via --jenkins-url flag or config file")
	}

	// Get pipeline name from arguments (already validated by Args function)
	pipelineName := args[0]

	// Get verbose flag
	verbose, _ := cmd.Flags().GetBool("verbose")

	// Get follow flag
	follow, _ := cmd.Flags().GetBool("follow")

	// Parse parameters from --param flags
	paramFlags, _ := cmd.Flags().GetStringArray("param")
	params, err := parseParameters(paramFlags)
	if err != nil {
		return fmt.Errorf("invalid parameter format: %w\n\nSuggestion: Use --param KEY=VALUE format (e.g., --param ENV=staging)", err)
	}

	// Initialize auth manager
	authMgr := auth.NewManager(profile.Auth.TokenFile)

	// Create Jenkins API client
	apiClient := client.NewClient(profile.Jenkins.URL, profile.Jenkins.Timeout, authMgr, profile.Name, verbose)

	// First, validate that the pipeline exists and get its parameter definitions
	ctx := context.Background()
	job, err := apiClient.GetJob(ctx, pipelineName)
	if err != nil {
		// Check if it's a 404 error (non-existent pipeline)
		if clientErr, ok := err.(*client.APIError); ok && clientErr.StatusCode == 404 {
			return fmt.Errorf("pipeline '%s' not found\n\nDetails: No job with name '%s' exists on the Jenkins server\nSuggestion: Use 'jctl pipelines list' to see available pipelines", pipelineName, pipelineName)
		}
		return fmt.Errorf("failed to validate pipeline: %w", err)
	}

	// Check if the job is buildable
	if !job.Buildable {
		return fmt.Errorf("pipeline '%s' is not buildable\n\nDetails: This job cannot be built (it may be disabled or a folder)\nSuggestion: Check the pipeline configuration in Jenkins", pipelineName)
	}

	// Validate parameters if the job is parameterized
	if job.IsParameterized() {
		if err := validateParameters(job, params); err != nil {
			return err
		}

		// Apply default values for missing optional parameters
		params = applyDefaultParameters(job, params)
	}

	// Trigger the build
	queueItem, err := apiClient.TriggerBuild(ctx, pipelineName, params)
	if err != nil {
		return fmt.Errorf("failed to trigger build: %w", err)
	}

	// If follow flag is set, wait for build to start and stream logs
	if follow {
		return followTriggeredBuild(ctx, apiClient, pipelineName, queueItem)
	}

	// Format and display output based on --output flag
	if err := formatTriggerOutput(queueItem, pipelineName, params, profile.Output.Format); err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	return nil
}

// parseParameters parses parameter flags in KEY=VALUE format into a map
func parseParameters(paramFlags []string) (map[string]string, error) {
	params := make(map[string]string)

	for _, param := range paramFlags {
		// Split on first '=' only
		parts := strings.SplitN(param, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("parameter '%s' is not in KEY=VALUE format", param)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if key == "" {
			return nil, fmt.Errorf("parameter key cannot be empty")
		}

		params[key] = value
	}

	return params, nil
}

// validateParameters validates that all required parameters are provided
func validateParameters(job *client.Job, providedParams map[string]string) error {
	paramDefs := job.GetParameterDefinitions()
	if len(paramDefs) == 0 {
		return nil
	}

	// Check for required parameters (those without default values)
	var missingParams []string
	for _, paramDef := range paramDefs {
		// If parameter has no default value and wasn't provided, it's missing
		hasDefault := paramDef.DefaultValue != nil && paramDef.DefaultValue.Value != nil
		_, provided := providedParams[paramDef.Name]

		if !hasDefault && !provided {
			missingParams = append(missingParams, paramDef.Name)
		}
	}

	if len(missingParams) > 0 {
		return fmt.Errorf("missing required parameters: %s\n\nDetails: The following parameters are required but were not provided: %s\nSuggestion: Add missing parameters using --param flags (e.g., --param %s=value)",
			strings.Join(missingParams, ", "),
			strings.Join(missingParams, ", "),
			missingParams[0])
	}

	return nil
}

// applyDefaultParameters applies default values for parameters that weren't provided
func applyDefaultParameters(job *client.Job, providedParams map[string]string) map[string]string {
	paramDefs := job.GetParameterDefinitions()
	if len(paramDefs) == 0 {
		return providedParams
	}

	// Create a new map with provided params
	result := make(map[string]string)
	for k, v := range providedParams {
		result[k] = v
	}

	// Add default values for parameters that weren't provided
	for _, paramDef := range paramDefs {
		_, provided := result[paramDef.Name]
		if !provided && paramDef.DefaultValue != nil && paramDef.DefaultValue.Value != nil {
			// Convert default value to string
			defaultValue := fmt.Sprintf("%v", paramDef.DefaultValue.Value)
			result[paramDef.Name] = defaultValue
		}
	}

	return result
}

// formatTriggerOutput formats the trigger result according to the specified format
func formatTriggerOutput(queueItem *client.QueueItem, pipelineName string, params map[string]string, format string) error {
	switch format {
	case "json":
		return formatTriggerJSON(queueItem, pipelineName, params)
	case "yaml":
		return formatTriggerYAML(queueItem, pipelineName, params)
	case "text":
		return formatTriggerText(queueItem, pipelineName, params)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// formatTriggerJSON outputs trigger result in JSON format
func formatTriggerJSON(queueItem *client.QueueItem, pipelineName string, params map[string]string) error {
	output := map[string]interface{}{
		"pipeline":   pipelineName,
		"parameters": params,
		"queueItem":  queueItem,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// formatTriggerYAML outputs trigger result in YAML format
func formatTriggerYAML(queueItem *client.QueueItem, pipelineName string, params map[string]string) error {
	output := map[string]interface{}{
		"pipeline":   pipelineName,
		"parameters": params,
		"queueItem":  queueItem,
	}

	encoder := yaml.NewEncoder(os.Stdout)
	defer encoder.Close()
	return encoder.Encode(output)
}

// formatTriggerText outputs trigger result in human-readable text format
func formatTriggerText(queueItem *client.QueueItem, pipelineName string, params map[string]string) error {
	fmt.Printf("✓ Build triggered successfully for pipeline: %s\n", pipelineName)

	if len(params) > 0 {
		fmt.Println("\nParameters:")
		for key, value := range params {
			fmt.Printf("  %s = %s\n", key, value)
		}
	}

	if queueItem.Why != "" {
		fmt.Printf("\nStatus: %s\n", queueItem.Why)
	}

	return nil
}

// followTriggeredBuild waits for a triggered build to start and streams its logs
func followTriggeredBuild(ctx context.Context, apiClient *client.Client, pipelineName string, queueItem *client.QueueItem) error {
	// Set up signal handling for graceful interruption (Ctrl+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	// Create a context that can be cancelled on interrupt
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Handle interrupt signal in a goroutine
	go func() {
		<-sigChan
		fmt.Fprintln(os.Stderr, "\nInterrupted. Exiting...")
		cancel()
	}()

	fmt.Printf("✓ Build triggered successfully for pipeline: %s\n", pipelineName)
	fmt.Printf("Queue ID: %d\n", queueItem.ID)
	fmt.Println("Waiting for build to start...")

	// Poll queue item until build starts
	buildNumber, err := waitForBuildStart(ctx, apiClient, queueItem.ID)
	if err != nil {
		return err
	}

	fmt.Printf("\n✓ Build #%d started\n", buildNumber)
	fmt.Println("Streaming logs...")
	fmt.Println()

	// Stream logs progressively using the shared function from logs.go
	if err := streamBuildLogs(ctx, apiClient, pipelineName, buildNumber); err != nil {
		return err
	}

	// Get final build status
	build, err := apiClient.GetBuildInfo(ctx, pipelineName, buildNumber)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nWarning: Failed to retrieve final build status: %v\n", err)
		return nil
	}

	// Display final build status
	fmt.Printf("\n✓ Build #%d completed with status: %s\n", buildNumber, build.Result)

	return nil
}

// waitForBuildStart polls the queue item until the build starts and returns the build number
func waitForBuildStart(ctx context.Context, apiClient *client.Client, queueID int) (int, error) {
	pollInterval := 2 * time.Second
	maxWaitTime := 5 * time.Minute
	startTime := time.Now()

	for {
		// Check if context was cancelled
		select {
		case <-ctx.Done():
			return 0, fmt.Errorf("cancelled while waiting for build to start")
		default:
		}

		// Check if we've exceeded max wait time
		if time.Since(startTime) > maxWaitTime {
			return 0, fmt.Errorf("timeout waiting for build to start after %v", maxWaitTime)
		}

		// Get queue item status
		queueItem, err := apiClient.GetQueueItem(ctx, queueID)
		if err != nil {
			// If queue item is not found, it might have been removed after build started
			// This is a race condition - we'll return an error and let the user retry
			if clientErr, ok := err.(*client.APIError); ok && clientErr.StatusCode == 404 {
				return 0, fmt.Errorf("queue item not found - build may have already started and completed")
			}
			return 0, fmt.Errorf("failed to check queue status: %w", err)
		}

		// Check if build has started (executable is present)
		if queueItem.Executable != nil {
			return queueItem.Executable.Number, nil
		}

		// Wait before polling again
		select {
		case <-ctx.Done():
			return 0, fmt.Errorf("cancelled while waiting for build to start")
		case <-time.After(pollInterval):
			// Continue to next iteration
		}
	}
}

// streamBuildLogs streams logs progressively without signal handling
// (signal handling is done by the caller)
func streamBuildLogs(ctx context.Context, apiClient *client.Client, pipelineName string, buildNumber int) error {
	var offset int64 = 0
	pollInterval := 2 * time.Second

	for {
		// Check if context was cancelled
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		// Get progressive log content
		logResp, err := apiClient.GetProgressiveLog(ctx, pipelineName, buildNumber, offset)
		if err != nil {
			// Check if it's a 404 error (non-existent build or pipeline)
			if clientErr, ok := err.(*client.APIError); ok && clientErr.StatusCode == 404 {
				return fmt.Errorf("build not found\n\nDetails: Build #%d for pipeline '%s' does not exist\nSuggestion: Use 'jctl builds list %s' to see available builds", buildNumber, pipelineName, pipelineName)
			}
			return fmt.Errorf("failed to retrieve build log: %w", err)
		}

		// Display new log content if any
		if len(logResp.Content) > 0 {
			fmt.Print(logResp.Content)
		}

		// Update offset for next request
		offset = logResp.NextOffset

		// Check for pending input steps on every iteration
		inputSteps, err := apiClient.GetPendingInputs(ctx, pipelineName, buildNumber)
		if err != nil {
			// Log error but continue - input detection is best-effort
			if verbose, _ := rootCmd.Flags().GetBool("verbose"); verbose {
				fmt.Fprintf(os.Stderr, "[DEBUG] Failed to check for pending inputs: %v\n", err)
			}
		} else {
			if verbose, _ := rootCmd.Flags().GetBool("verbose"); verbose {
				fmt.Fprintf(os.Stderr, "[DEBUG] Checked for pending inputs, found %d\n", len(inputSteps))
			}
			if len(inputSteps) > 0 {
				// Handle the first pending input step
				inputStep := inputSteps[0]

				err := handleInputStep(ctx, apiClient, pipelineName, buildNumber, inputStep)
				if err != nil {
					return fmt.Errorf("failed to handle input step: %w", err)
				}
			}
		}

		// If build is complete, exit
		if !logResp.HasMoreData {
			break
		}

		// Wait before polling again
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(pollInterval):
			// Continue to next iteration
		}
	}

	return nil
}
