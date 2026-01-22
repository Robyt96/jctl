package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/user/jctl/internal/auth"
	"github.com/user/jctl/internal/client"
	"gopkg.in/yaml.v3"
)

var pipelinesCmd = &cobra.Command{
	Use:   "pipelines",
	Short: "Manage Jenkins pipelines",
	Long:  `Commands for listing and managing Jenkins pipelines.`,
}

var pipelinesListCmd = &cobra.Command{
	Use:   "list [folder]",
	Short: "List all pipelines",
	Long:  `List all available pipelines on the Jenkins server. Optionally specify a folder path to list pipelines within that folder.`,
	Example: `  jctl pipelines list
  jctl pipelines list --output json
  jctl pipelines list my-folder
  jctl pipelines list my-folder/sub-folder`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPipelinesList(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(pipelinesCmd)
	pipelinesCmd.AddCommand(pipelinesListCmd)
}

// runPipelinesList executes the pipelines list command
func runPipelinesList(cmd *cobra.Command, args []string) error {
	// Validate Jenkins URL is configured
	if profile.Jenkins.URL == "" {
		return fmt.Errorf("Jenkins URL not configured. Set it via --jenkins-url flag or config file")
	}

	// Get folder path from arguments (if provided)
	folderPath := ""
	if len(args) > 0 {
		folderPath = args[0]
	}

	// Get verbose flag
	verbose, _ := cmd.Flags().GetBool("verbose")

	// Initialize auth manager
	authMgr := auth.NewManager(profile.Auth.TokenFile)

	// Create Jenkins API client
	apiClient := client.NewClient(profile.Jenkins.URL, profile.Jenkins.Timeout, authMgr, profile.Name, verbose)

	// Call API to list jobs
	ctx := context.Background()
	jobs, err := apiClient.ListJobs(ctx, folderPath)
	if err != nil {
		return fmt.Errorf("failed to list pipelines: %w", err)
	}

	// Handle empty pipeline list
	if len(jobs) == 0 {
		if folderPath == "" {
			fmt.Println("No pipelines found on the Jenkins server")
		} else {
			fmt.Printf("No pipelines found in folder: %s\n", folderPath)
		}
		return nil
	}

	// Format and display output based on --output flag
	if err := formatPipelinesOutput(jobs, profile.Output.Format, folderPath); err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	return nil
}

// formatPipelinesOutput formats the pipeline list according to the specified format
func formatPipelinesOutput(jobs []client.Job, format string, folderPath string) error {
	switch format {
	case "json":
		return formatPipelinesJSON(jobs)
	case "yaml":
		return formatPipelinesYAML(jobs)
	case "text":
		return formatPipelinesText(jobs, folderPath)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// formatPipelinesJSON outputs pipelines in JSON format
func formatPipelinesJSON(jobs []client.Job) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(jobs)
}

// formatPipelinesYAML outputs pipelines in YAML format
func formatPipelinesYAML(jobs []client.Job) error {
	encoder := yaml.NewEncoder(os.Stdout)
	defer encoder.Close()
	return encoder.Encode(jobs)
}

// formatPipelinesText outputs pipelines in human-readable text format
func formatPipelinesText(jobs []client.Job, folderPath string) error {
	// Print location header
	if folderPath == "" {
		fmt.Println("Pipelines (root):")
	} else {
		fmt.Printf("Pipelines in folder: %s\n", folderPath)
	}
	fmt.Println()

	// Create tabwriter for aligned columns
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	defer w.Flush()

	// Print header
	fmt.Fprintln(w, "TYPE\tNAME\tSTATUS\tLAST BUILD\tDESCRIPTION")
	fmt.Fprintln(w, "----\t----\t------\t----------\t-----------")

	// Print each pipeline/folder
	for _, job := range jobs {
		itemType := "Pipeline"
		status := getStatusFromColor(job.Color)
		lastBuild := "N/A"

		if job.IsFolder() {
			itemType = "Folder"
			status = "-"
			lastBuild = "-"
		} else if job.LastBuild != nil {
			lastBuild = fmt.Sprintf("#%d", job.LastBuild.Number)
		}

		description := job.Description
		if description == "" {
			description = "-"
		}
		// Truncate long descriptions
		if len(description) > 50 {
			description = description[:47] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", itemType, job.Name, status, lastBuild, description)
	}

	// Print helpful hint about folders
	hasFolders := false
	for _, job := range jobs {
		if job.IsFolder() {
			hasFolders = true
			break
		}
	}

	if hasFolders {
		fmt.Println()
		fmt.Println("Tip: To list pipelines in a folder, use: jctl pipelines list <folder-name>")
	}

	return nil
}

// getStatusFromColor converts Jenkins color codes to human-readable status
func getStatusFromColor(color string) string {
	// Jenkins uses color codes like "blue", "red", "yellow", "grey", "disabled", "aborted", "notbuilt"
	// Animated versions have "_anime" suffix (e.g., "blue_anime" for building)
	switch {
	case color == "blue" || color == "blue_anime":
		return "Success"
	case color == "red" || color == "red_anime":
		return "Failed"
	case color == "yellow" || color == "yellow_anime":
		return "Unstable"
	case color == "grey" || color == "grey_anime":
		return "Pending"
	case color == "disabled":
		return "Disabled"
	case color == "aborted" || color == "aborted_anime":
		return "Aborted"
	case color == "notbuilt" || color == "notbuilt_anime":
		return "Not Built"
	default:
		return "Unknown"
	}
}
