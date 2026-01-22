package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/user/jctl/internal/auth"
	"github.com/user/jctl/internal/client"
)

var logsCmd = &cobra.Command{
	Use:   "logs <pipeline> <build-number>",
	Short: "View build logs",
	Long:  `View the console logs for a specific build. Pipeline can be a simple name or a folder path (e.g., folder/subfolder/pipeline).`,
	Example: `  jctl logs my-pipeline 42
  jctl logs Odino/OdinoInfrastructureProducts 123
  jctl logs backend-service 456
  jctl logs my-pipeline 42 --follow`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("pipeline name and build number are required\n\nUsage:\n  %s\n\nExample:\n%s", cmd.Use, cmd.Example)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLogs(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(logsCmd)
	logsCmd.Flags().BoolP("follow", "f", false, "Follow log output (stream logs as build progresses)")
}

// runLogs executes the logs command
func runLogs(cmd *cobra.Command, args []string) error {
	// Validate Jenkins URL is configured
	if profile.Jenkins.URL == "" {
		return fmt.Errorf("Jenkins URL not configured. Set it via --jenkins-url flag or config file")
	}

	// Get pipeline name and build number from arguments (already validated by Args function)
	pipelineName := args[0]
	buildNumberStr := args[1]

	// Parse build number
	buildNumber, err := strconv.Atoi(buildNumberStr)
	if err != nil {
		return fmt.Errorf("invalid build number '%s': must be a positive integer\n\nUsage:\n  %s\n\nExample:\n%s", buildNumberStr, cmd.Use, cmd.Example)
	}

	if buildNumber <= 0 {
		return fmt.Errorf("invalid build number '%d': must be a positive integer\n\nUsage:\n  %s\n\nExample:\n%s", buildNumber, cmd.Use, cmd.Example)
	}

	// Get flags
	verbose, _ := cmd.Flags().GetBool("verbose")
	follow, _ := cmd.Flags().GetBool("follow")

	// Initialize auth manager
	authMgr := auth.NewManager(profile.Auth.TokenFile)

	// Create Jenkins API client
	apiClient := client.NewClient(profile.Jenkins.URL, profile.Jenkins.Timeout, authMgr, profile.Name, verbose)

	ctx := context.Background()

	// If follow flag is set, use progressive log streaming
	if follow {
		return streamLogsProgressively(ctx, apiClient, pipelineName, buildNumber)
	}

	// Otherwise, get the complete log
	log, err := apiClient.GetBuildLog(ctx, pipelineName, buildNumber)
	if err != nil {
		// Check if it's a 404 error (non-existent build or pipeline)
		if clientErr, ok := err.(*client.APIError); ok && clientErr.StatusCode == 404 {
			return fmt.Errorf("build not found\n\nDetails: Build #%d for pipeline '%s' does not exist\nSuggestion: Use 'jctl builds list %s' to see available builds", buildNumber, pipelineName, pipelineName)
		}
		return fmt.Errorf("failed to retrieve build log: %w", err)
	}

	// Stream log output to stdout
	fmt.Print(log)

	return nil
}

// streamLogsProgressively streams logs progressively as the build runs
func streamLogsProgressively(ctx context.Context, apiClient *client.Client, pipelineName string, buildNumber int) error {
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

// handleInputStep handles a pending input step by prompting the user and submitting their response
func handleInputStep(ctx context.Context, apiClient *client.Client, pipelineName string, buildNumber int, inputStep client.InputStep) error {
	// Display the input prompt message
	fmt.Fprintf(os.Stderr, "\n\n=== Input Required ===\n")
	fmt.Fprintf(os.Stderr, "%s\n\n", inputStep.Message)

	// Create a scanner for reading user input
	scanner := bufio.NewScanner(os.Stdin)

	// Check if this is a simple approval or parameterized input
	if len(inputStep.Parameters) == 0 {
		// Simple approval - prompt for yes/no
		okText := inputStep.OK
		if okText == "" {
			okText = "Proceed"
		}
		abortText := inputStep.Abort
		if abortText == "" {
			abortText = "Abort"
		}

		fmt.Fprintf(os.Stderr, "%s? (y/n): ", okText)

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("failed to read user input: %w", err)
			}
			return fmt.Errorf("failed to read user input: EOF")
		}

		response := strings.ToLower(strings.TrimSpace(scanner.Text()))

		if verbose, _ := rootCmd.Flags().GetBool("verbose"); verbose {
			fmt.Fprintf(os.Stderr, "[DEBUG] User response: '%s'\n", response)
		}

		if response == "y" || response == "yes" {
			// Submit approval
			if verbose, _ := rootCmd.Flags().GetBool("verbose"); verbose {
				fmt.Fprintf(os.Stderr, "[DEBUG] Submitting approval for input ID: %s\n", inputStep.ID)
			}
			err := apiClient.SubmitInput(ctx, pipelineName, buildNumber, inputStep.ID, nil)
			if err != nil {
				return fmt.Errorf("failed to submit input: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Input submitted. Continuing...\n\n")
		} else if response == "n" || response == "no" || response == "abort" {
			// Abort the input
			if verbose, _ := rootCmd.Flags().GetBool("verbose"); verbose {
				fmt.Fprintf(os.Stderr, "[DEBUG] Aborting input ID: %s\n", inputStep.ID)
			}
			err := apiClient.AbortInput(ctx, pipelineName, buildNumber, inputStep.ID)
			if err != nil {
				return fmt.Errorf("failed to abort input: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Input aborted. Build will be aborted.\n\n")
		} else {
			return fmt.Errorf("invalid response '%s'. Expected 'y' or 'n'", response)
		}
	} else {
		// Parameterized input - prompt for each parameter
		params := make(map[string]string)

		fmt.Fprintf(os.Stderr, "Please provide values for the following parameters:\n\n")

		for _, param := range inputStep.Parameters {
			// Display parameter description
			if param.Description != "" {
				fmt.Fprintf(os.Stderr, "%s: %s\n", param.Name, param.Description)
			} else {
				fmt.Fprintf(os.Stderr, "%s:\n", param.Name)
			}

			// Show default value if available
			if param.DefaultValue != "" {
				fmt.Fprintf(os.Stderr, "  (default: %s)\n", param.DefaultValue)
			}

			// Show choices for choice parameters
			if len(param.Choices) > 0 {
				fmt.Fprintf(os.Stderr, "  Choices: %s\n", strings.Join(param.Choices, ", "))
			}

			fmt.Fprintf(os.Stderr, "Value: ")

			// Read user input
			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					return fmt.Errorf("failed to read parameter value: %w", err)
				}
				return fmt.Errorf("failed to read parameter value: EOF")
			}

			value := strings.TrimSpace(scanner.Text())

			// Use default value if user provided empty input
			if value == "" && param.DefaultValue != "" {
				value = param.DefaultValue
			}

			params[param.Name] = value
			fmt.Fprintf(os.Stderr, "\n")
		}

		// Ask for final confirmation
		fmt.Fprintf(os.Stderr, "Submit these values? (y/n/abort): ")
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("failed to read confirmation: %w", err)
			}
			return fmt.Errorf("failed to read confirmation: EOF")
		}

		confirm := strings.ToLower(strings.TrimSpace(scanner.Text()))

		if confirm == "y" || confirm == "yes" {
			// Submit the parameters
			err := apiClient.SubmitInput(ctx, pipelineName, buildNumber, inputStep.ID, params)
			if err != nil {
				return fmt.Errorf("failed to submit input: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Input submitted. Continuing...\n\n")
		} else if confirm == "n" || confirm == "no" || confirm == "abort" {
			// Abort the input
			err := apiClient.AbortInput(ctx, pipelineName, buildNumber, inputStep.ID)
			if err != nil {
				return fmt.Errorf("failed to abort input: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Input aborted. Build will be aborted.\n\n")
		} else {
			return fmt.Errorf("invalid confirmation '%s'. Expected 'y', 'n', or 'abort'", confirm)
		}
	}

	return nil
}
