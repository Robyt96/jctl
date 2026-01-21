package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Token represents an authentication token
type Token struct {
	Value     string `json:"value"`
	Type      string `json:"type"`       // "api_token" or "oauth"
	Username  string `json:"username"`   // Jenkins username (required for api_token)
	ExpiresAt int64  `json:"expires_at"` // Unix timestamp, 0 for non-expiring tokens
}

// ProfileCredentials represents credentials for a single profile
type ProfileCredentials struct {
	Token     string `yaml:"token"`
	Username  string `yaml:"username"`
	ExpiresAt int64  `yaml:"expires_at"`
}

// Credentials represents all stored credentials organized by profile
type Credentials struct {
	Profiles map[string]*ProfileCredentials `yaml:"profiles"`
}

// Manager handles authentication operations
type Manager struct {
	credentialsFile string
	legacyTokenFile string // For migration from old token file
}

// NewManager creates a new authentication manager
func NewManager(credentialsFile string) *Manager {
	expandedPath := expandPath(credentialsFile)
	return &Manager{
		credentialsFile: expandedPath,
		legacyTokenFile: expandPath("~/.jctl/token"), // For migration
	}
}

// loadCredentials loads all credentials from the credentials file
func (m *Manager) loadCredentials() (*Credentials, error) {
	// Check if credentials file exists
	if _, err := os.Stat(m.credentialsFile); os.IsNotExist(err) {
		// Try to migrate from legacy token file
		if err := m.migrateLegacyToken(); err == nil {
			// Retry loading after migration
			return m.loadCredentials()
		}
		// Return empty credentials if no file exists
		return &Credentials{
			Profiles: make(map[string]*ProfileCredentials),
		}, nil
	}

	// Read credentials file
	data, err := os.ReadFile(m.credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	// Unmarshal credentials
	var creds Credentials
	if err := yaml.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials file: %w", err)
	}

	// Initialize profiles map if nil
	if creds.Profiles == nil {
		creds.Profiles = make(map[string]*ProfileCredentials)
	}

	return &creds, nil
}

// saveCredentials saves all credentials to the credentials file
func (m *Manager) saveCredentials(creds *Credentials) error {
	// Ensure the directory exists
	dir := filepath.Dir(m.credentialsFile)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create credentials directory: %w", err)
	}

	// Marshal credentials to YAML
	data, err := yaml.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	// Write credentials file with secure permissions (0600 - read/write for owner only)
	if err := os.WriteFile(m.credentialsFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}

	return nil
}

// migrateLegacyToken migrates from old single-token file to new credentials file
func (m *Manager) migrateLegacyToken() error {
	// Check if legacy token file exists
	if _, err := os.Stat(m.legacyTokenFile); os.IsNotExist(err) {
		return fmt.Errorf("no legacy token file found")
	}

	// Read legacy token file
	data, err := os.ReadFile(m.legacyTokenFile)
	if err != nil {
		return fmt.Errorf("failed to read legacy token file: %w", err)
	}

	// Unmarshal legacy token
	var token Token
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf("failed to parse legacy token file: %w", err)
	}

	// Create new credentials structure with token in "default" profile
	creds := &Credentials{
		Profiles: map[string]*ProfileCredentials{
			"default": {
				Token:     token.Value,
				Username:  token.Username,
				ExpiresAt: token.ExpiresAt,
			},
		},
	}

	// Save to new credentials file
	if err := m.saveCredentials(creds); err != nil {
		return fmt.Errorf("failed to save migrated credentials: %w", err)
	}

	// Optionally remove legacy token file (commented out for safety)
	// os.Remove(m.legacyTokenFile)

	fmt.Printf("Migrated legacy token to new credentials file: %s\n", m.credentialsFile)
	return nil
}

// StoreToken saves a token for a specific profile
func (m *Manager) StoreToken(profile string, token *Token) error {
	// Load existing credentials
	creds, err := m.loadCredentials()
	if err != nil {
		return err
	}

	// Store token for the specified profile
	creds.Profiles[profile] = &ProfileCredentials{
		Token:     token.Value,
		Username:  token.Username,
		ExpiresAt: token.ExpiresAt,
	}

	// Save credentials
	return m.saveCredentials(creds)
}

// GetToken retrieves the stored token for a specific profile
func (m *Manager) GetToken(profile string) (*Token, error) {
	// Load credentials
	creds, err := m.loadCredentials()
	if err != nil {
		return nil, err
	}

	// Get profile credentials
	profileCreds, exists := creds.Profiles[profile]
	if !exists {
		return nil, fmt.Errorf("no authentication token found for profile '%s': run 'jctl auth login --profile %s' to authenticate", profile, profile)
	}

	// Check if token is expired
	if profileCreds.ExpiresAt > 0 && time.Now().Unix() > profileCreds.ExpiresAt {
		return nil, fmt.Errorf("authentication token for profile '%s' has expired: run 'jctl auth login --profile %s' to re-authenticate", profile, profile)
	}

	// Create token object
	token := &Token{
		Value:     profileCreds.Token,
		Type:      "api_token", // Default to api_token for stored credentials
		Username:  profileCreds.Username,
		ExpiresAt: profileCreds.ExpiresAt,
	}

	return token, nil
}

// ClearToken removes the stored token for a specific profile
func (m *Manager) ClearToken(profile string) error {
	// Load credentials
	creds, err := m.loadCredentials()
	if err != nil {
		return err
	}

	// Remove profile credentials
	delete(creds.Profiles, profile)

	// Save updated credentials
	return m.saveCredentials(creds)
}

// ListProfiles returns a list of all profiles with stored credentials
func (m *Manager) ListProfiles() ([]string, error) {
	// Load credentials
	creds, err := m.loadCredentials()
	if err != nil {
		return nil, err
	}

	// Extract profile names
	profiles := make([]string, 0, len(creds.Profiles))
	for profile := range creds.Profiles {
		profiles = append(profiles, profile)
	}

	return profiles, nil
}

// ValidateToken validates a token against the Jenkins API
func (m *Manager) ValidateToken(token *Token, jenkinsURL string) error {
	if token == nil {
		return fmt.Errorf("token is nil")
	}

	if token.Value == "" {
		return fmt.Errorf("token value is empty")
	}

	// Check if token is expired
	if token.ExpiresAt > 0 && time.Now().Unix() > token.ExpiresAt {
		return fmt.Errorf("token has expired")
	}

	// If Jenkins URL is provided, validate against the API
	if jenkinsURL != "" {
		if err := m.validateAgainstAPI(token, jenkinsURL); err != nil {
			return fmt.Errorf("token validation failed: %w", err)
		}
	}

	return nil
}

// validateAgainstAPI validates the token by making a request to Jenkins API
func (m *Manager) validateAgainstAPI(token *Token, jenkinsURL string) error {
	// Import http package at the top of the file
	// This will make a simple API call to verify the token works
	// We'll use the /api/json endpoint which requires authentication

	// For now, we'll implement a basic check
	// The actual HTTP call will be implemented when we have the API client
	// This is a placeholder that checks token structure

	if token.Type != "api_token" && token.Type != "oauth" {
		return fmt.Errorf("invalid token type: %s", token.Type)
	}

	// TODO: Make actual HTTP request to Jenkins API when API client is available
	// For now, we just validate the token structure

	return nil
}

// expandPath expands ~ to the user's home directory
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// OAuthConfig contains OAuth flow configuration
type OAuthConfig struct {
	JenkinsURL   string
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       []string
}

// OAuthResult contains the result of an OAuth flow
type OAuthResult struct {
	Token        *Token
	Error        error
	AuthCode     string
	CallbackPort int
}

// Login initiates the OAuth browser flow for a specific profile
func (m *Manager) Login(profile string, config *OAuthConfig) (*Token, error) {
	// Generate a random state parameter for CSRF protection
	state, err := generateRandomState()
	if err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	// Start local HTTP server on random port for callback
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to start callback server: %w", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	// Create channel to receive OAuth result
	resultChan := make(chan *OAuthResult, 1)

	// Set up HTTP server for callback
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		handleOAuthCallback(w, r, state, resultChan)
	})

	server := &http.Server{
		Handler: mux,
	}

	// Start server in background
	go func() {
		server.Serve(listener)
	}()

	// Build OAuth authorization URL
	authURL := buildAuthURL(config.JenkinsURL, config.ClientID, redirectURI, state, config.Scopes)

	// Open browser to OAuth URL
	fmt.Printf("Opening browser for authentication...\n")
	fmt.Printf("If the browser doesn't open automatically, visit: %s\n", authURL)

	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Failed to open browser automatically: %v\n", err)
	}

	// Wait for callback with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	select {
	case result := <-resultChan:
		// Shutdown server
		server.Shutdown(context.Background())

		if result.Error != nil {
			return nil, result.Error
		}

		// Exchange authorization code for access token
		token, err := m.exchangeCodeForToken(config, result.AuthCode)
		if err != nil {
			return nil, fmt.Errorf("failed to exchange code for token: %w", err)
		}

		// Store the token for the specified profile
		if err := m.StoreToken(profile, token); err != nil {
			return nil, fmt.Errorf("failed to store token: %w", err)
		}

		return token, nil

	case <-ctx.Done():
		server.Shutdown(context.Background())
		return nil, fmt.Errorf("authentication timeout: no response received within 5 minutes")
	}
}

// handleOAuthCallback handles the OAuth callback request
func handleOAuthCallback(w http.ResponseWriter, r *http.Request, expectedState string, resultChan chan *OAuthResult) {
	// Parse query parameters
	query := r.URL.Query()

	// Check for error
	if errMsg := query.Get("error"); errMsg != "" {
		errDesc := query.Get("error_description")
		result := &OAuthResult{
			Error: fmt.Errorf("OAuth error: %s - %s", errMsg, errDesc),
		}
		resultChan <- result

		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "<html><body><h1>Authentication Failed</h1><p>%s</p></body></html>", errMsg)
		return
	}

	// Verify state parameter
	state := query.Get("state")
	if state != expectedState {
		result := &OAuthResult{
			Error: fmt.Errorf("invalid state parameter: possible CSRF attack"),
		}
		resultChan <- result

		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "<html><body><h1>Authentication Failed</h1><p>Invalid state parameter</p></body></html>")
		return
	}

	// Extract authorization code
	code := query.Get("code")
	if code == "" {
		result := &OAuthResult{
			Error: fmt.Errorf("no authorization code received"),
		}
		resultChan <- result

		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "<html><body><h1>Authentication Failed</h1><p>No authorization code received</p></body></html>")
		return
	}

	// Send success result
	result := &OAuthResult{
		AuthCode: code,
	}
	resultChan <- result

	// Send success response to browser
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "<html><body><h1>Authentication Successful</h1><p>You can close this window and return to the terminal.</p></body></html>")
}

// exchangeCodeForToken exchanges an authorization code for an access token
func (m *Manager) exchangeCodeForToken(config *OAuthConfig, code string) (*Token, error) {
	// Build token endpoint URL
	tokenURL := fmt.Sprintf("%s/oauth/token", strings.TrimSuffix(config.JenkinsURL, "/"))

	// Prepare form data
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", config.RedirectURI)
	data.Set("client_id", config.ClientID)
	data.Set("client_secret", config.ClientSecret)

	// Make POST request to token endpoint
	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("failed to request token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token request failed with status: %d", resp.StatusCode)
	}

	// Parse token response
	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int64  `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	// Calculate expiration time
	var expiresAt int64
	if tokenResp.ExpiresIn > 0 {
		expiresAt = time.Now().Unix() + tokenResp.ExpiresIn
	}

	// Create token
	token := &Token{
		Value:     tokenResp.AccessToken,
		Type:      "oauth",
		ExpiresAt: expiresAt,
	}

	return token, nil
}

// buildAuthURL constructs the OAuth authorization URL
func buildAuthURL(jenkinsURL, clientID, redirectURI, state string, scopes []string) string {
	authURL := fmt.Sprintf("%s/oauth/authorize", strings.TrimSuffix(jenkinsURL, "/"))

	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", clientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("state", state)

	if len(scopes) > 0 {
		params.Set("scope", strings.Join(scopes, " "))
	}

	return fmt.Sprintf("%s?%s", authURL, params.Encode())
}

// generateRandomState generates a random state parameter for CSRF protection
func generateRandomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// openBrowser opens the default browser to the specified URL
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}
