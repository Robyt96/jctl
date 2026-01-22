package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/jctl/internal/auth"
	"github.com/user/jctl/internal/config"
	"gopkg.in/yaml.v3"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage profiles",
	Long:  `Commands for managing jctl profiles. Profiles allow you to configure multiple Jenkins instances and switch between them easily.`,
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured profiles",
	Long:  `Display all configured profiles with their Jenkins URLs and indicate which is the default.`,
	Example: `  jctl profile list
  jctl profile list --output json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runProfileList()
	},
}

func runProfileList() error {
	// Get all profiles from config
	profileNames := cfg.ListProfiles()

	// Sort profile names for consistent output
	sort.Strings(profileNames)

	// Get default profile name
	defaultProfile := cfg.GetDefaultProfile()

	// Check which profiles have credentials
	authMgr := auth.NewManager(profile.Auth.TokenFile)
	credentialedProfiles, err := authMgr.ListProfiles()
	if err != nil {
		// Non-fatal error - just means we can't show credential status
		credentialedProfiles = []string{}
	}

	// Create a map for quick lookup
	hasCredentials := make(map[string]bool)
	for _, p := range credentialedProfiles {
		hasCredentials[p] = true
	}

	// Format output based on --output flag
	outputFormat := profile.Output.Format

	switch outputFormat {
	case "json":
		return outputProfileListJSON(profileNames, defaultProfile, hasCredentials)
	case "yaml":
		return outputProfileListYAML(profileNames, defaultProfile, hasCredentials)
	default: // "text"
		return outputProfileListText(profileNames, defaultProfile, hasCredentials)
	}
}

func outputProfileListText(profileNames []string, defaultProfile string, hasCredentials map[string]bool) error {
	if len(profileNames) == 0 {
		fmt.Println("No profiles configured")
		return nil
	}

	fmt.Println("Configured Profiles:")
	fmt.Println()

	for _, name := range profileNames {
		prof, err := cfg.GetProfile(name)
		if err != nil {
			continue
		}

		// Build status indicators
		indicators := []string{}
		if name == defaultProfile {
			indicators = append(indicators, "default")
		}
		if hasCredentials[name] {
			indicators = append(indicators, "authenticated")
		}

		statusStr := ""
		if len(indicators) > 0 {
			statusStr = fmt.Sprintf(" [%s]", strings.Join(indicators, ", "))
		}

		fmt.Printf("  %s%s\n", name, statusStr)
		fmt.Printf("    URL: %s\n", prof.Jenkins.URL)
		fmt.Printf("    Timeout: %s\n", prof.Jenkins.Timeout)
		fmt.Printf("    Auth Method: %s\n", prof.Auth.Method)
		fmt.Println()
	}

	return nil
}

func outputProfileListJSON(profileNames []string, defaultProfile string, hasCredentials map[string]bool) error {
	type ProfileInfo struct {
		Name          string `json:"name"`
		URL           string `json:"url"`
		Timeout       string `json:"timeout"`
		AuthMethod    string `json:"auth_method"`
		IsDefault     bool   `json:"is_default"`
		Authenticated bool   `json:"authenticated"`
	}

	profiles := []ProfileInfo{}

	for _, name := range profileNames {
		prof, err := cfg.GetProfile(name)
		if err != nil {
			continue
		}

		profiles = append(profiles, ProfileInfo{
			Name:          name,
			URL:           prof.Jenkins.URL,
			Timeout:       prof.Jenkins.Timeout.String(),
			AuthMethod:    prof.Auth.Method,
			IsDefault:     name == defaultProfile,
			Authenticated: hasCredentials[name],
		})
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(profiles)
}

func outputProfileListYAML(profileNames []string, defaultProfile string, hasCredentials map[string]bool) error {
	type ProfileInfo struct {
		Name          string `yaml:"name"`
		URL           string `yaml:"url"`
		Timeout       string `yaml:"timeout"`
		AuthMethod    string `yaml:"auth_method"`
		IsDefault     bool   `yaml:"is_default"`
		Authenticated bool   `yaml:"authenticated"`
	}

	profiles := []ProfileInfo{}

	for _, name := range profileNames {
		prof, err := cfg.GetProfile(name)
		if err != nil {
			continue
		}

		profiles = append(profiles, ProfileInfo{
			Name:          name,
			URL:           prof.Jenkins.URL,
			Timeout:       prof.Jenkins.Timeout.String(),
			AuthMethod:    prof.Auth.Method,
			IsDefault:     name == defaultProfile,
			Authenticated: hasCredentials[name],
		})
	}

	encoder := yaml.NewEncoder(os.Stdout)
	return encoder.Encode(profiles)
}

var profileShowCmd = &cobra.Command{
	Use:   "show [profile-name]",
	Short: "Show profile configuration",
	Long:  `Display the configuration for a specific profile. If no profile name is provided, shows the current profile.`,
	Example: `  jctl profile show
  jctl profile show production
  jctl profile show --output json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName := ""
		if len(args) > 0 {
			profileName = args[0]
		}
		return runProfileShow(profileName)
	},
}

func runProfileShow(profileName string) error {
	// If no profile name provided, use current profile
	if profileName == "" {
		profileName = profile.Name
	}

	// Get the profile
	prof, err := cfg.GetProfile(profileName)
	if err != nil {
		return fmt.Errorf("Error: Profile '%s' not found\nSuggestion: Use 'jctl profile list' to see available profiles", profileName)
	}

	// Check if credentials are configured
	authMgr := auth.NewManager(profile.Auth.TokenFile)
	_, err = authMgr.GetToken(profileName)
	hasCredentials := err == nil

	// Check if this is the default profile
	isDefault := profileName == cfg.GetDefaultProfile()

	// Format output based on --output flag
	outputFormat := profile.Output.Format

	switch outputFormat {
	case "json":
		return outputProfileShowJSON(prof, isDefault, hasCredentials)
	case "yaml":
		return outputProfileShowYAML(prof, isDefault, hasCredentials)
	default: // "text"
		return outputProfileShowText(prof, isDefault, hasCredentials)
	}
}

func outputProfileShowText(prof *config.Profile, isDefault bool, hasCredentials bool) error {
	fmt.Printf("Profile: %s\n", prof.Name)

	if isDefault {
		fmt.Println("Status: Default profile")
	}

	fmt.Println()
	fmt.Println("Jenkins Configuration:")
	fmt.Printf("  URL: %s\n", prof.Jenkins.URL)
	fmt.Printf("  Timeout: %s\n", prof.Jenkins.Timeout)

	fmt.Println()
	fmt.Println("Authentication:")
	fmt.Printf("  Method: %s\n", prof.Auth.Method)
	fmt.Printf("  Token File: %s\n", prof.Auth.TokenFile)
	if hasCredentials {
		fmt.Println("  Credentials: Configured ✓")
	} else {
		fmt.Println("  Credentials: Not configured")
		fmt.Printf("  Suggestion: Run 'jctl auth login --profile %s' to authenticate\n", prof.Name)
	}

	fmt.Println()
	fmt.Println("Output Settings:")
	fmt.Printf("  Format: %s\n", prof.Output.Format)
	fmt.Printf("  Color: %t\n", prof.Output.Color)

	if prof.Defaults.Pipeline != "" {
		fmt.Println()
		fmt.Println("Defaults:")
		fmt.Printf("  Pipeline: %s\n", prof.Defaults.Pipeline)
	}

	return nil
}

func outputProfileShowJSON(prof *config.Profile, isDefault bool, hasCredentials bool) error {
	type ProfileDetails struct {
		Name          string `json:"name"`
		IsDefault     bool   `json:"is_default"`
		Authenticated bool   `json:"authenticated"`
		Jenkins       struct {
			URL     string `json:"url"`
			Timeout string `json:"timeout"`
		} `json:"jenkins"`
		Auth struct {
			Method    string `json:"method"`
			TokenFile string `json:"token_file"`
		} `json:"auth"`
		Output struct {
			Format string `json:"format"`
			Color  bool   `json:"color"`
		} `json:"output"`
		Defaults struct {
			Pipeline string `json:"pipeline,omitempty"`
		} `json:"defaults,omitempty"`
	}

	details := ProfileDetails{
		Name:          prof.Name,
		IsDefault:     isDefault,
		Authenticated: hasCredentials,
	}
	details.Jenkins.URL = prof.Jenkins.URL
	details.Jenkins.Timeout = prof.Jenkins.Timeout.String()
	details.Auth.Method = prof.Auth.Method
	details.Auth.TokenFile = prof.Auth.TokenFile
	details.Output.Format = prof.Output.Format
	details.Output.Color = prof.Output.Color
	details.Defaults.Pipeline = prof.Defaults.Pipeline

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(details)
}

func outputProfileShowYAML(prof *config.Profile, isDefault bool, hasCredentials bool) error {
	type ProfileDetails struct {
		Name          string `yaml:"name"`
		IsDefault     bool   `yaml:"is_default"`
		Authenticated bool   `yaml:"authenticated"`
		Jenkins       struct {
			URL     string `yaml:"url"`
			Timeout string `yaml:"timeout"`
		} `yaml:"jenkins"`
		Auth struct {
			Method    string `yaml:"method"`
			TokenFile string `yaml:"token_file"`
		} `yaml:"auth"`
		Output struct {
			Format string `yaml:"format"`
			Color  bool   `yaml:"color"`
		} `yaml:"output"`
		Defaults struct {
			Pipeline string `yaml:"pipeline,omitempty"`
		} `yaml:"defaults,omitempty"`
	}

	details := ProfileDetails{
		Name:          prof.Name,
		IsDefault:     isDefault,
		Authenticated: hasCredentials,
	}
	details.Jenkins.URL = prof.Jenkins.URL
	details.Jenkins.Timeout = prof.Jenkins.Timeout.String()
	details.Auth.Method = prof.Auth.Method
	details.Auth.TokenFile = prof.Auth.TokenFile
	details.Output.Format = prof.Output.Format
	details.Output.Color = prof.Output.Color
	details.Defaults.Pipeline = prof.Defaults.Pipeline

	encoder := yaml.NewEncoder(os.Stdout)
	return encoder.Encode(details)
}

var profileSetDefaultCmd = &cobra.Command{
	Use:   "set-default <profile-name>",
	Short: "Set the default profile",
	Long:  `Set the specified profile as the default profile. The default profile is used when no --profile flag is provided.`,
	Example: `  jctl profile set-default production
  jctl profile set-default staging`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName := args[0]
		return runProfileSetDefault(profileName)
	},
}

func runProfileSetDefault(profileName string) error {
	// Validate profile exists
	_, err := cfg.GetProfile(profileName)
	if err != nil {
		return fmt.Errorf("Error: Profile '%s' not found\nSuggestion: Use 'jctl profile list' to see available profiles", profileName)
	}

	// Set as default
	if err := cfg.SetDefaultProfile(profileName); err != nil {
		return fmt.Errorf("Error: Failed to set default profile: %v", err)
	}

	// Get config file path
	configPath, _ := rootCmd.PersistentFlags().GetString("config")
	if configPath == "" {
		configPath = "~/.jctl/config.yaml"
	}

	// Save updated config
	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("Error: Failed to save configuration: %v", err)
	}

	fmt.Printf("✓ Default profile set to '%s'\n", profileName)
	return nil
}

func init() {
	rootCmd.AddCommand(profileCmd)
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileShowCmd)
	profileCmd.AddCommand(profileSetDefaultCmd)
}
