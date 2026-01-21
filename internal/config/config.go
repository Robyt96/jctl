package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Config represents the complete configuration for jctl with profile support
type Config struct {
	DefaultProfile string              `yaml:"default_profile" mapstructure:"default_profile"`
	Profiles       map[string]*Profile `yaml:"profiles" mapstructure:"profiles"`
}

// Profile represents a named configuration set for a specific Jenkins instance
type Profile struct {
	Name     string         `yaml:"-"` // Not stored in YAML, set from map key
	Jenkins  JenkinsConfig  `yaml:"jenkins"`
	Auth     AuthConfig     `yaml:"auth"`
	Output   OutputConfig   `yaml:"output"`
	Defaults DefaultsConfig `yaml:"defaults"`
}

// JenkinsConfig contains Jenkins server connection settings
type JenkinsConfig struct {
	URL     string        `yaml:"url"`
	Timeout time.Duration `yaml:"timeout"`
}

// AuthConfig contains authentication settings
type AuthConfig struct {
	Method    string `yaml:"method"` // "token" or "oauth"
	TokenFile string `yaml:"token_file"`
}

// OutputConfig contains output formatting settings
type OutputConfig struct {
	Format string `yaml:"format"` // "text", "json", or "yaml"
	Color  bool   `yaml:"color"`
}

// DefaultsConfig contains default values for commands
type DefaultsConfig struct {
	Pipeline string `yaml:"pipeline"`
}

// NewDefaultConfig returns a Config with sensible defaults
func NewDefaultConfig() *Config {
	defaultProfile := &Profile{
		Name: "default",
		Jenkins: JenkinsConfig{
			URL:     "",
			Timeout: 30 * time.Second,
		},
		Auth: AuthConfig{
			Method:    "token",
			TokenFile: "~/.jctl/credentials",
		},
		Output: OutputConfig{
			Format: "text",
			Color:  true,
		},
		Defaults: DefaultsConfig{
			Pipeline: "",
		},
	}

	return &Config{
		DefaultProfile: "default",
		Profiles: map[string]*Profile{
			"default": defaultProfile,
		},
	}
}

// Loader handles loading and validating configuration
type Loader struct {
	configPath string
}

// NewLoader creates a new configuration loader
func NewLoader(configPath string) *Loader {
	return &Loader{
		configPath: configPath,
	}
}

// Load reads and parses the configuration file
// Returns default config if file doesn't exist
func (l *Loader) Load() (*Config, error) {
	// If no config path specified, try default location
	if l.configPath == "" {
		l.configPath = expandPath("~/.jctl/config.yaml")
	} else {
		l.configPath = expandPath(l.configPath)
	}

	// Check if config file exists
	if !fileExists(l.configPath) {
		// Return default config if file doesn't exist
		return NewDefaultConfig(), nil
	}

	// Load config using viper
	v := viper.New()
	v.SetConfigFile(l.configPath)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Create empty config to unmarshal into
	cfg := &Config{}

	// Unmarshal into config struct
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set profile names from map keys
	for name, profile := range cfg.Profiles {
		profile.Name = name
	}

	// Validate the configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks if the configuration values are valid
func (c *Config) Validate() error {
	var errors []string

	// Validate that at least one profile exists
	if len(c.Profiles) == 0 {
		errors = append(errors, "at least one profile must be defined")
	}

	// Validate default profile exists
	if c.DefaultProfile != "" {
		if _, exists := c.Profiles[c.DefaultProfile]; !exists {
			errors = append(errors, fmt.Sprintf("default_profile '%s' does not exist in profiles", c.DefaultProfile))
		}
	}

	// Validate each profile
	for name, profile := range c.Profiles {
		profileErrors := profile.Validate()
		for _, err := range profileErrors {
			errors = append(errors, fmt.Sprintf("profile '%s': %s", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// Validate checks if a profile's configuration values are valid
func (p *Profile) Validate() []string {
	var errors []string

	// Validate Jenkins URL if provided
	if p.Jenkins.URL != "" {
		if !isValidURL(p.Jenkins.URL) {
			errors = append(errors, "jenkins.url: invalid URL format")
		}
	}

	// Validate timeout
	if p.Jenkins.Timeout < 0 {
		errors = append(errors, "jenkins.timeout: timeout cannot be negative")
	}
	if p.Jenkins.Timeout == 0 {
		p.Jenkins.Timeout = 30 * time.Second
	}

	// Set default token file if not specified
	if p.Auth.TokenFile == "" {
		p.Auth.TokenFile = "~/.jctl/credentials"
	}

	// Validate auth method
	if p.Auth.Method != "" && p.Auth.Method != "token" && p.Auth.Method != "oauth" {
		errors = append(errors, "auth.method: must be 'token' or 'oauth'")
	}

	// Validate output format
	if p.Output.Format != "" && p.Output.Format != "text" && p.Output.Format != "json" && p.Output.Format != "yaml" {
		errors = append(errors, "output.format: must be 'text', 'json', or 'yaml'")
	}

	return errors
}

// GetProfile retrieves a specific profile by name
func (c *Config) GetProfile(name string) (*Profile, error) {
	profile, exists := c.Profiles[name]
	if !exists {
		return nil, fmt.Errorf("profile '%s' not found", name)
	}
	return profile, nil
}

// ListProfiles returns a list of all profile names
func (c *Config) ListProfiles() []string {
	names := make([]string, 0, len(c.Profiles))
	for name := range c.Profiles {
		names = append(names, name)
	}
	return names
}

// GetDefaultProfile returns the name of the default profile
func (c *Config) GetDefaultProfile() string {
	return c.DefaultProfile
}

// SetDefaultProfile sets the default profile name
func (c *Config) SetDefaultProfile(name string) error {
	if _, exists := c.Profiles[name]; !exists {
		return fmt.Errorf("profile '%s' does not exist", name)
	}
	c.DefaultProfile = name
	return nil
}

// Save writes the configuration to a file
func (c *Config) Save(path string) error {
	// Expand path
	expandedPath := expandPath(path)

	// Ensure directory exists
	dir := filepath.Dir(expandedPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal config to YAML
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(expandedPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Helper functions

func isValidURL(urlStr string) bool {
	if urlStr == "" {
		return false
	}
	u, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	// Must have http or https scheme and host
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// MergeProfile merges profile settings with overrides
// This is used to implement configuration precedence: CLI flags > env vars > file > defaults
func (p *Profile) MergeProfile(other *Profile) {
	// Merge Jenkins config
	if other.Jenkins.URL != "" {
		p.Jenkins.URL = other.Jenkins.URL
	}
	if other.Jenkins.Timeout != 0 {
		p.Jenkins.Timeout = other.Jenkins.Timeout
	}

	// Merge Auth config
	if other.Auth.Method != "" {
		p.Auth.Method = other.Auth.Method
	}
	if other.Auth.TokenFile != "" {
		p.Auth.TokenFile = other.Auth.TokenFile
	}

	// Merge Output config
	if other.Output.Format != "" {
		p.Output.Format = other.Output.Format
	}
	// For boolean, we need to check if it was explicitly set
	// Since we can't distinguish between false and unset, we always take the other value
	p.Output.Color = other.Output.Color

	// Merge Defaults config
	if other.Defaults.Pipeline != "" {
		p.Defaults.Pipeline = other.Defaults.Pipeline
	}
}

// LoadWithOverrides loads configuration from file and applies overrides from CLI flags and environment
func LoadWithOverrides(configPath string, profileName string, profileOverrides *Profile) (*Config, *Profile, error) {
	loader := NewLoader(configPath)

	// Load base config from file (or defaults if file doesn't exist)
	cfg, err := loader.Load()
	if err != nil {
		return nil, nil, err
	}

	// Determine which profile to use
	selectedProfileName := profileName
	if selectedProfileName == "" {
		// Check environment variable
		if envProfile := os.Getenv("JCTL_PROFILE"); envProfile != "" {
			selectedProfileName = envProfile
		} else if cfg.DefaultProfile != "" {
			// Use default profile from config
			selectedProfileName = cfg.DefaultProfile
		} else {
			// Fall back to "default" profile
			selectedProfileName = "default"
		}
	}

	// Get the selected profile
	profile, exists := cfg.Profiles[selectedProfileName]
	if !exists {
		return nil, nil, fmt.Errorf("profile '%s' not found in configuration", selectedProfileName)
	}

	// Apply environment variable overrides
	envOverrides := loadProfileFromEnv()
	profile.MergeProfile(envOverrides)

	// Apply CLI flag overrides (highest precedence)
	if profileOverrides != nil {
		profile.MergeProfile(profileOverrides)
	}

	// Validate final merged profile
	if errs := profile.Validate(); len(errs) > 0 {
		return nil, nil, fmt.Errorf("profile validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}

	return cfg, profile, nil
}

// loadProfileFromEnv loads profile configuration values from environment variables
func loadProfileFromEnv() *Profile {
	profile := &Profile{}

	// Check for JCTL_JENKINS_URL
	if url := os.Getenv("JCTL_JENKINS_URL"); url != "" {
		profile.Jenkins.URL = url
	}

	// Check for JCTL_JENKINS_TIMEOUT
	if timeoutStr := os.Getenv("JCTL_JENKINS_TIMEOUT"); timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			profile.Jenkins.Timeout = timeout
		}
	}

	// Check for JCTL_AUTH_METHOD
	if method := os.Getenv("JCTL_AUTH_METHOD"); method != "" {
		profile.Auth.Method = method
	}

	// Check for JCTL_AUTH_TOKEN_FILE
	if tokenFile := os.Getenv("JCTL_AUTH_TOKEN_FILE"); tokenFile != "" {
		profile.Auth.TokenFile = tokenFile
	}

	// Check for JCTL_OUTPUT_FORMAT
	if format := os.Getenv("JCTL_OUTPUT_FORMAT"); format != "" {
		profile.Output.Format = format
	}

	// Check for JCTL_OUTPUT_COLOR
	if colorStr := os.Getenv("JCTL_OUTPUT_COLOR"); colorStr != "" {
		profile.Output.Color = colorStr == "true" || colorStr == "1"
	}

	// Check for JCTL_DEFAULT_PIPELINE
	if pipeline := os.Getenv("JCTL_DEFAULT_PIPELINE"); pipeline != "" {
		profile.Defaults.Pipeline = pipeline
	}

	return profile
}
