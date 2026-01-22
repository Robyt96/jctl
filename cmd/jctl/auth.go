package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/jctl/internal/auth"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
	Long:  `Commands for managing authentication with Jenkins.`,
}

var (
	authMethod        string
	oauthClientID     string
	oauthClientSecret string
)

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Jenkins",
	Long:  `Initiate browser-based authentication or store an API token.`,
	Example: `  jctl auth login
  jctl auth login --method token
  jctl auth login --method oauth --client-id <id> --client-secret <secret>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAuthLogin()
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd)

	// Add flags for auth login
	authLoginCmd.Flags().StringVar(&authMethod, "method", "", "Authentication method: 'token' or 'oauth' (default: prompt)")
	authLoginCmd.Flags().StringVar(&oauthClientID, "client-id", "", "OAuth client ID (required for OAuth)")
	authLoginCmd.Flags().StringVar(&oauthClientSecret, "client-secret", "", "OAuth client secret (required for OAuth)")
}

func runAuthLogin() error {
	// Check if Jenkins URL is configured
	if profile.Jenkins.URL == "" {
		return fmt.Errorf("Jenkins URL not configured. Set it via:\n  - Config file: ~/.jctl/config.yaml\n  - Environment variable: JCTL_JENKINS_URL\n  - Command flag: --jenkins-url")
	}

	// Create auth manager with credentials file
	authMgr := auth.NewManager(profile.Auth.TokenFile)

	// Display which profile is being authenticated
	fmt.Printf("Authenticating profile: %s\n", profile.Name)
	fmt.Printf("Jenkins URL: %s\n\n", profile.Jenkins.URL)

	// Determine authentication method
	method := authMethod
	if method == "" {
		// Use configured method if available
		if profile.Auth.Method != "" {
			method = profile.Auth.Method
		} else {
			// Prompt user to choose
			var err error
			method, err = promptAuthMethod()
			if err != nil {
				return err
			}
		}
	}

	// Validate method
	if method != "token" && method != "oauth" {
		return fmt.Errorf("invalid authentication method: %s (must be 'token' or 'oauth')", method)
	}

	// Execute authentication based on method
	switch method {
	case "token":
		return authenticateWithToken(authMgr, profile.Name)
	case "oauth":
		return authenticateWithOAuth(authMgr, profile.Name)
	default:
		return fmt.Errorf("unsupported authentication method: %s", method)
	}
}

func promptAuthMethod() (string, error) {
	fmt.Println("Choose authentication method:")
	fmt.Println("  1. API Token (recommended)")
	fmt.Println("  2. OAuth (browser-based)")
	fmt.Print("\nEnter choice (1 or 2): ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)
	switch input {
	case "1":
		return "token", nil
	case "2":
		return "oauth", nil
	default:
		return "", fmt.Errorf("invalid choice: %s", input)
	}
}

func authenticateWithToken(authMgr *auth.Manager, profileName string) error {
	fmt.Println("API Token Authentication")
	fmt.Println("------------------------")
	fmt.Printf("Generate an API token from Jenkins: %s/user/<username>/configure\n\n", profile.Jenkins.URL)

	// Read username
	fmt.Print("Enter your Jenkins username: ")
	reader := bufio.NewReader(os.Stdin)
	username, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read username: %w", err)
	}
	username = strings.TrimSpace(username)
	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	// Read token
	fmt.Print("Enter your API token: ")
	tokenValue, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read token: %w", err)
	}

	tokenValue = strings.TrimSpace(tokenValue)
	if tokenValue == "" {
		return fmt.Errorf("token cannot be empty")
	}

	// Create token object
	token := &auth.Token{
		Value:     tokenValue,
		Type:      "api_token",
		Username:  username,
		ExpiresAt: 0, // API tokens don't expire
	}

	// Validate token against Jenkins API
	fmt.Println("\nValidating token...")
	if err := authMgr.ValidateToken(token, profile.Jenkins.URL); err != nil {
		return fmt.Errorf("token validation failed: %w", err)
	}

	// Store token for the current profile
	if err := authMgr.StoreToken(profileName, token); err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}

	fmt.Println("✓ Authentication successful!")
	fmt.Printf("Profile '%s' authenticated\n", profileName)
	fmt.Printf("Credentials stored in: %s\n", profile.Auth.TokenFile)
	return nil
}

func authenticateWithOAuth(authMgr *auth.Manager, profileName string) error {
	fmt.Println("OAuth Authentication")
	fmt.Println("-------------------")

	// Check if OAuth credentials are provided
	clientID := oauthClientID
	clientSecret := oauthClientSecret

	if clientID == "" {
		fmt.Print("Enter OAuth Client ID: ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read client ID: %w", err)
		}
		clientID = strings.TrimSpace(input)
	}

	if clientSecret == "" {
		fmt.Print("Enter OAuth Client Secret: ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read client secret: %w", err)
		}
		clientSecret = strings.TrimSpace(input)
	}

	if clientID == "" || clientSecret == "" {
		return fmt.Errorf("OAuth client ID and secret are required")
	}

	// Configure OAuth
	oauthConfig := &auth.OAuthConfig{
		JenkinsURL:   profile.Jenkins.URL,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{"read", "write"},
	}

	// Initiate OAuth flow for the current profile
	fmt.Println("\nInitiating OAuth flow...")
	token, err := authMgr.Login(profileName, oauthConfig)
	if err != nil {
		return fmt.Errorf("OAuth authentication failed: %w", err)
	}

	fmt.Println("✓ Authentication successful!")
	fmt.Printf("Profile '%s' authenticated\n", profileName)
	fmt.Printf("Credentials stored in: %s\n", profile.Auth.TokenFile)

	// Display token expiration if applicable
	if token.ExpiresAt > 0 {
		fmt.Printf("Token expires at: %s\n", formatTimestamp(token.ExpiresAt))
	}

	return nil
}

func formatTimestamp(timestamp int64) string {
	// Convert Unix timestamp to human-readable format
	// This is a simple implementation
	return fmt.Sprintf("%d", timestamp)
}
