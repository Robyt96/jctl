package config

import (
	"strings"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: jenkins-cli-tool, Property 7: Configuration Validation
// Validates: Requirements 5.4
// For any configuration file with invalid values (malformed URLs, negative timeouts, etc.),
// when jctl loads the configuration, it should report specific validation errors identifying the invalid fields.
func TestProperty_ConfigurationValidation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Generator for invalid URL configurations
	genInvalidURL := gen.OneConstOf(
		"not-a-url",
		"ftp://invalid",
		"://missing-scheme",
		"http://",
		"invalid url with spaces",
		"missing-scheme.com",
	)

	// Generator for invalid auth methods
	genInvalidAuthMethod := gen.OneConstOf(
		"invalid",
		"basic",
		"bearer",
		"random",
		"api-key",
	)

	// Generator for invalid output formats
	genInvalidOutputFormat := gen.OneConstOf(
		"xml",
		"csv",
		"html",
		"invalid",
		"pdf",
	)

	// Generator for negative timeouts
	genNegativeTimeout := gen.Int64Range(-60, -1).Map(func(n int64) time.Duration {
		return time.Duration(n) * time.Second
	})

	// Property 1: Invalid URLs should produce validation errors
	properties.Property("invalid URLs should produce validation errors", prop.ForAll(
		func(invalidURL string) bool {
			cfg := &Config{
				DefaultProfile: "default",
				Profiles: map[string]*Profile{
					"default": {
						Name: "default",
						Jenkins: JenkinsConfig{
							URL:     invalidURL,
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
					},
				},
			}

			err := cfg.Validate()

			// Should return an error for invalid URL
			if err == nil {
				t.Logf("Expected validation error for invalid URL '%s' but got none", invalidURL)
				return false
			}

			// Error message should mention URL
			errMsg := strings.ToLower(err.Error())
			if !strings.Contains(errMsg, "url") && !strings.Contains(errMsg, "invalid") {
				t.Logf("Error message doesn't identify URL field: %s", err.Error())
				return false
			}

			return true
		},
		genInvalidURL,
	))

	// Property 2: Negative timeouts should produce validation errors
	properties.Property("negative timeouts should produce validation errors", prop.ForAll(
		func(negativeTimeout time.Duration) bool {
			cfg := &Config{
				DefaultProfile: "default",
				Profiles: map[string]*Profile{
					"default": {
						Name: "default",
						Jenkins: JenkinsConfig{
							URL:     "https://jenkins.example.com",
							Timeout: negativeTimeout,
						},
						Auth: AuthConfig{
							Method:    "token",
							TokenFile: "~/.jctl/credentials",
						},
						Output: OutputConfig{
							Format: "text",
							Color:  true,
						},
					},
				},
			}

			err := cfg.Validate()

			// Should return an error for negative timeout
			if err == nil {
				t.Logf("Expected validation error for negative timeout %v but got none", negativeTimeout)
				return false
			}

			// Error message should mention timeout
			errMsg := strings.ToLower(err.Error())
			if !strings.Contains(errMsg, "timeout") && !strings.Contains(errMsg, "negative") {
				t.Logf("Error message doesn't identify timeout field: %s", err.Error())
				return false
			}

			return true
		},
		genNegativeTimeout,
	))

	// Property 3: Invalid auth methods should produce validation errors
	properties.Property("invalid auth methods should produce validation errors", prop.ForAll(
		func(invalidMethod string) bool {
			cfg := &Config{
				DefaultProfile: "default",
				Profiles: map[string]*Profile{
					"default": {
						Name: "default",
						Jenkins: JenkinsConfig{
							URL:     "https://jenkins.example.com",
							Timeout: 30 * time.Second,
						},
						Auth: AuthConfig{
							Method:    invalidMethod,
							TokenFile: "~/.jctl/credentials",
						},
						Output: OutputConfig{
							Format: "text",
							Color:  true,
						},
					},
				},
			}

			err := cfg.Validate()

			// Should return an error for invalid auth method
			if err == nil {
				t.Logf("Expected validation error for invalid auth method '%s' but got none", invalidMethod)
				return false
			}

			// Error message should mention auth method
			errMsg := strings.ToLower(err.Error())
			if !strings.Contains(errMsg, "auth") && !strings.Contains(errMsg, "method") {
				t.Logf("Error message doesn't identify auth method field: %s", err.Error())
				return false
			}

			return true
		},
		genInvalidAuthMethod,
	))

	// Property 4: Invalid output formats should produce validation errors
	properties.Property("invalid output formats should produce validation errors", prop.ForAll(
		func(invalidFormat string) bool {
			cfg := &Config{
				DefaultProfile: "default",
				Profiles: map[string]*Profile{
					"default": {
						Name: "default",
						Jenkins: JenkinsConfig{
							URL:     "https://jenkins.example.com",
							Timeout: 30 * time.Second,
						},
						Auth: AuthConfig{
							Method:    "token",
							TokenFile: "~/.jctl/credentials",
						},
						Output: OutputConfig{
							Format: invalidFormat,
							Color:  true,
						},
					},
				},
			}

			err := cfg.Validate()

			// Should return an error for invalid output format
			if err == nil {
				t.Logf("Expected validation error for invalid output format '%s' but got none", invalidFormat)
				return false
			}

			// Error message should mention output format
			errMsg := strings.ToLower(err.Error())
			if !strings.Contains(errMsg, "output") && !strings.Contains(errMsg, "format") {
				t.Logf("Error message doesn't identify output format field: %s", err.Error())
				return false
			}

			return true
		},
		genInvalidOutputFormat,
	))

	// Property 5: Non-existent default profile should produce validation error
	properties.Property("non-existent default profile should produce validation error", prop.ForAll(
		func(profileName string) bool {
			// Skip if profile name happens to be "default"
			if profileName == "default" {
				return true
			}

			cfg := &Config{
				DefaultProfile: profileName,
				Profiles: map[string]*Profile{
					"default": {
						Name: "default",
						Jenkins: JenkinsConfig{
							URL:     "https://jenkins.example.com",
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
					},
				},
			}

			err := cfg.Validate()

			// Should return an error for non-existent default profile
			if err == nil {
				t.Logf("Expected validation error for non-existent default profile '%s' but got none", profileName)
				return false
			}

			// Error message should mention profile
			errMsg := strings.ToLower(err.Error())
			if !strings.Contains(errMsg, "profile") && !strings.Contains(errMsg, "default") {
				t.Logf("Error message doesn't identify profile issue: %s", err.Error())
				return false
			}

			return true
		},
		gen.AlphaString().SuchThat(func(s string) bool {
			return s != "" && s != "default"
		}),
	))

	properties.TestingRun(t)
}

// Feature: jenkins-cli-tool, Property 14: Profile Configuration Retrieval
// Validates: Requirements 9.1, 9.2
// For any valid profile name, when jctl loads that profile's configuration,
// all settings specific to that profile should be applied (Jenkins URL, timeout, auth method, etc.).
func TestProperty_ProfileConfigurationRetrieval(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Generator for valid profile names
	genProfileName := gen.OneConstOf(
		"development",
		"staging",
		"production",
		"test",
		"ci",
		"local",
		"qa",
		"demo",
	)

	// Generator for valid URLs
	genValidURL := gen.OneConstOf(
		"https://jenkins1.example.com",
		"https://jenkins2.example.com",
		"http://localhost:8080",
		"https://ci.company.com",
		"https://build.server.net",
		"https://jenkins-dev.internal",
		"https://jenkins-prod.cloud",
	)

	// Generator for valid timeouts
	genValidTimeout := gen.Int64Range(1, 300).Map(func(n int64) time.Duration {
		return time.Duration(n) * time.Second
	})

	// Generator for valid auth methods
	genValidAuthMethod := gen.OneConstOf("token", "oauth")

	// Generator for valid output formats
	genValidOutputFormat := gen.OneConstOf("text", "json", "yaml")

	// Generator for boolean values
	genBool := gen.Bool()

	// Generator for pipeline names
	genPipelineName := gen.OneConstOf(
		"main-pipeline",
		"build-pipeline",
		"deploy-pipeline",
		"test-pipeline",
		"",
	)

	// Property 1: GetProfile should retrieve profile with all settings intact
	properties.Property("GetProfile should retrieve profile with all settings intact", prop.ForAll(
		func(profileName string, url string, timeout time.Duration, authMethod string, outputFormat string, color bool, pipeline string) bool {
			// Create a config with the profile
			cfg := &Config{
				DefaultProfile: "default",
				Profiles: map[string]*Profile{
					profileName: {
						Name: profileName,
						Jenkins: JenkinsConfig{
							URL:     url,
							Timeout: timeout,
						},
						Auth: AuthConfig{
							Method:    authMethod,
							TokenFile: "~/.jctl/credentials",
						},
						Output: OutputConfig{
							Format: outputFormat,
							Color:  color,
						},
						Defaults: DefaultsConfig{
							Pipeline: pipeline,
						},
					},
				},
			}

			// Retrieve the profile
			retrievedProfile, err := cfg.GetProfile(profileName)
			if err != nil {
				t.Logf("Failed to retrieve profile '%s': %v", profileName, err)
				return false
			}

			// Verify all settings match
			if retrievedProfile.Name != profileName {
				t.Logf("Profile name mismatch: expected '%s', got '%s'", profileName, retrievedProfile.Name)
				return false
			}
			if retrievedProfile.Jenkins.URL != url {
				t.Logf("Jenkins URL mismatch: expected '%s', got '%s'", url, retrievedProfile.Jenkins.URL)
				return false
			}
			if retrievedProfile.Jenkins.Timeout != timeout {
				t.Logf("Timeout mismatch: expected %v, got %v", timeout, retrievedProfile.Jenkins.Timeout)
				return false
			}
			if retrievedProfile.Auth.Method != authMethod {
				t.Logf("Auth method mismatch: expected '%s', got '%s'", authMethod, retrievedProfile.Auth.Method)
				return false
			}
			if retrievedProfile.Output.Format != outputFormat {
				t.Logf("Output format mismatch: expected '%s', got '%s'", outputFormat, retrievedProfile.Output.Format)
				return false
			}
			if retrievedProfile.Output.Color != color {
				t.Logf("Color setting mismatch: expected %v, got %v", color, retrievedProfile.Output.Color)
				return false
			}
			if retrievedProfile.Defaults.Pipeline != pipeline {
				t.Logf("Default pipeline mismatch: expected '%s', got '%s'", pipeline, retrievedProfile.Defaults.Pipeline)
				return false
			}

			return true
		},
		genProfileName,
		genValidURL,
		genValidTimeout,
		genValidAuthMethod,
		genValidOutputFormat,
		genBool,
		genPipelineName,
	))

	// Property 2: GetProfile should fail for non-existent profiles
	properties.Property("GetProfile should fail for non-existent profiles", prop.ForAll(
		func(existingProfile string, nonExistentProfile string) bool {
			// Skip if profile names are the same
			if existingProfile == nonExistentProfile {
				return true
			}

			// Create a config with only one profile
			cfg := &Config{
				DefaultProfile: existingProfile,
				Profiles: map[string]*Profile{
					existingProfile: {
						Name: existingProfile,
						Jenkins: JenkinsConfig{
							URL:     "https://jenkins.example.com",
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
					},
				},
			}

			// Try to retrieve non-existent profile
			_, err := cfg.GetProfile(nonExistentProfile)

			// Should return an error
			if err == nil {
				t.Logf("Expected error when retrieving non-existent profile '%s', but got none", nonExistentProfile)
				return false
			}

			// Error message should mention the profile name
			errMsg := strings.ToLower(err.Error())
			if !strings.Contains(errMsg, "not found") && !strings.Contains(errMsg, "profile") {
				t.Logf("Error message doesn't clearly indicate profile not found: %s", err.Error())
				return false
			}

			return true
		},
		genProfileName,
		genProfileName,
	))

	// Property 3: Multiple profiles should maintain independent settings
	properties.Property("multiple profiles should maintain independent settings", prop.ForAll(
		func(profile1Name string, profile2Name string, url1 string, url2 string, timeout1 time.Duration, timeout2 time.Duration) bool {
			// Skip if profile names are the same or URLs/timeouts are the same
			if profile1Name == profile2Name || (url1 == url2 && timeout1 == timeout2) {
				return true
			}

			// Create a config with two profiles with different settings
			cfg := &Config{
				DefaultProfile: profile1Name,
				Profiles: map[string]*Profile{
					profile1Name: {
						Name: profile1Name,
						Jenkins: JenkinsConfig{
							URL:     url1,
							Timeout: timeout1,
						},
						Auth: AuthConfig{
							Method:    "token",
							TokenFile: "~/.jctl/credentials",
						},
						Output: OutputConfig{
							Format: "text",
							Color:  true,
						},
					},
					profile2Name: {
						Name: profile2Name,
						Jenkins: JenkinsConfig{
							URL:     url2,
							Timeout: timeout2,
						},
						Auth: AuthConfig{
							Method:    "oauth",
							TokenFile: "~/.jctl/credentials",
						},
						Output: OutputConfig{
							Format: "json",
							Color:  false,
						},
					},
				},
			}

			// Retrieve both profiles
			retrievedProfile1, err1 := cfg.GetProfile(profile1Name)
			if err1 != nil {
				t.Logf("Failed to retrieve profile '%s': %v", profile1Name, err1)
				return false
			}

			retrievedProfile2, err2 := cfg.GetProfile(profile2Name)
			if err2 != nil {
				t.Logf("Failed to retrieve profile '%s': %v", profile2Name, err2)
				return false
			}

			// Verify profile 1 settings
			if retrievedProfile1.Jenkins.URL != url1 {
				t.Logf("Profile 1 URL mismatch: expected '%s', got '%s'", url1, retrievedProfile1.Jenkins.URL)
				return false
			}
			if retrievedProfile1.Jenkins.Timeout != timeout1 {
				t.Logf("Profile 1 timeout mismatch: expected %v, got %v", timeout1, retrievedProfile1.Jenkins.Timeout)
				return false
			}
			if retrievedProfile1.Auth.Method != "token" {
				t.Logf("Profile 1 auth method mismatch: expected 'token', got '%s'", retrievedProfile1.Auth.Method)
				return false
			}

			// Verify profile 2 settings
			if retrievedProfile2.Jenkins.URL != url2 {
				t.Logf("Profile 2 URL mismatch: expected '%s', got '%s'", url2, retrievedProfile2.Jenkins.URL)
				return false
			}
			if retrievedProfile2.Jenkins.Timeout != timeout2 {
				t.Logf("Profile 2 timeout mismatch: expected %v, got %v", timeout2, retrievedProfile2.Jenkins.Timeout)
				return false
			}
			if retrievedProfile2.Auth.Method != "oauth" {
				t.Logf("Profile 2 auth method mismatch: expected 'oauth', got '%s'", retrievedProfile2.Auth.Method)
				return false
			}

			return true
		},
		genProfileName,
		genProfileName,
		genValidURL,
		genValidURL,
		genValidTimeout,
		genValidTimeout,
	))

	// Property 4: Profile settings should not be affected by other profiles
	properties.Property("profile settings should not be affected by other profiles", prop.ForAll(
		func(targetProfile string, otherProfile string, targetURL string, otherURL string) bool {
			// Skip if profile names are the same
			if targetProfile == otherProfile {
				return true
			}

			// Create a config with two profiles
			cfg := &Config{
				DefaultProfile: targetProfile,
				Profiles: map[string]*Profile{
					targetProfile: {
						Name: targetProfile,
						Jenkins: JenkinsConfig{
							URL:     targetURL,
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
					},
					otherProfile: {
						Name: otherProfile,
						Jenkins: JenkinsConfig{
							URL:     otherURL,
							Timeout: 60 * time.Second,
						},
						Auth: AuthConfig{
							Method:    "oauth",
							TokenFile: "~/.jctl/credentials",
						},
						Output: OutputConfig{
							Format: "json",
							Color:  false,
						},
					},
				},
			}

			// Retrieve target profile
			retrievedProfile, err := cfg.GetProfile(targetProfile)
			if err != nil {
				t.Logf("Failed to retrieve profile '%s': %v", targetProfile, err)
				return false
			}

			// Verify target profile has its own settings, not the other profile's
			if retrievedProfile.Jenkins.URL != targetURL {
				t.Logf("Target profile URL should be '%s', got '%s'", targetURL, retrievedProfile.Jenkins.URL)
				return false
			}
			if retrievedProfile.Jenkins.Timeout != 30*time.Second {
				t.Logf("Target profile timeout should be 30s, got %v", retrievedProfile.Jenkins.Timeout)
				return false
			}
			if retrievedProfile.Auth.Method != "token" {
				t.Logf("Target profile auth method should be 'token', got '%s'", retrievedProfile.Auth.Method)
				return false
			}
			if retrievedProfile.Output.Format != "text" {
				t.Logf("Target profile output format should be 'text', got '%s'", retrievedProfile.Output.Format)
				return false
			}

			return true
		},
		genProfileName,
		genProfileName,
		genValidURL,
		genValidURL,
	))

	properties.TestingRun(t)
}

// Feature: jenkins-cli-tool, Property 6: Configuration Precedence
// Validates: Requirements 5.5
// For any configuration key that appears in both the config file and command-line flags,
// the command-line flag value should take precedence in the effective configuration.
func TestProperty_ConfigurationPrecedence(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Generator for valid URLs
	genValidURL := gen.OneConstOf(
		"https://jenkins1.example.com",
		"https://jenkins2.example.com",
		"http://localhost:8080",
		"https://ci.company.com",
		"https://build.server.net",
	)

	// Generator for valid timeouts
	genValidTimeout := gen.Int64Range(1, 300).Map(func(n int64) time.Duration {
		return time.Duration(n) * time.Second
	})

	// Generator for valid auth methods
	genValidAuthMethod := gen.OneConstOf("token", "oauth")

	// Generator for valid output formats
	genValidOutputFormat := gen.OneConstOf("text", "json", "yaml")

	// Generator for boolean values
	genBool := gen.Bool()

	// Property 1: CLI flag URL should override file URL
	properties.Property("CLI flag URL should override file URL", prop.ForAll(
		func(fileURL string, cliURL string) bool {
			// Skip if URLs are the same
			if fileURL == cliURL {
				return true
			}

			// Create base profile from file
			baseProfile := &Profile{
				Name: "test",
				Jenkins: JenkinsConfig{
					URL:     fileURL,
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
			}

			// Create CLI override profile
			cliOverride := &Profile{
				Jenkins: JenkinsConfig{
					URL: cliURL,
				},
			}

			// Merge CLI override into base profile
			baseProfile.MergeProfile(cliOverride)

			// Verify CLI URL took precedence
			if baseProfile.Jenkins.URL != cliURL {
				t.Logf("Expected URL to be '%s' (CLI), but got '%s'", cliURL, baseProfile.Jenkins.URL)
				return false
			}

			return true
		},
		genValidURL,
		genValidURL,
	))

	// Property 2: CLI flag timeout should override file timeout
	properties.Property("CLI flag timeout should override file timeout", prop.ForAll(
		func(fileTimeout time.Duration, cliTimeout time.Duration) bool {
			// Skip if timeouts are the same
			if fileTimeout == cliTimeout {
				return true
			}

			// Create base profile from file
			baseProfile := &Profile{
				Name: "test",
				Jenkins: JenkinsConfig{
					URL:     "https://jenkins.example.com",
					Timeout: fileTimeout,
				},
				Auth: AuthConfig{
					Method:    "token",
					TokenFile: "~/.jctl/credentials",
				},
				Output: OutputConfig{
					Format: "text",
					Color:  true,
				},
			}

			// Create CLI override profile
			cliOverride := &Profile{
				Jenkins: JenkinsConfig{
					Timeout: cliTimeout,
				},
			}

			// Merge CLI override into base profile
			baseProfile.MergeProfile(cliOverride)

			// Verify CLI timeout took precedence
			if baseProfile.Jenkins.Timeout != cliTimeout {
				t.Logf("Expected timeout to be %v (CLI), but got %v", cliTimeout, baseProfile.Jenkins.Timeout)
				return false
			}

			return true
		},
		genValidTimeout,
		genValidTimeout,
	))

	// Property 3: CLI flag auth method should override file auth method
	properties.Property("CLI flag auth method should override file auth method", prop.ForAll(
		func(fileMethod string, cliMethod string) bool {
			// Skip if methods are the same
			if fileMethod == cliMethod {
				return true
			}

			// Create base profile from file
			baseProfile := &Profile{
				Name: "test",
				Jenkins: JenkinsConfig{
					URL:     "https://jenkins.example.com",
					Timeout: 30 * time.Second,
				},
				Auth: AuthConfig{
					Method:    fileMethod,
					TokenFile: "~/.jctl/credentials",
				},
				Output: OutputConfig{
					Format: "text",
					Color:  true,
				},
			}

			// Create CLI override profile
			cliOverride := &Profile{
				Auth: AuthConfig{
					Method: cliMethod,
				},
			}

			// Merge CLI override into base profile
			baseProfile.MergeProfile(cliOverride)

			// Verify CLI auth method took precedence
			if baseProfile.Auth.Method != cliMethod {
				t.Logf("Expected auth method to be '%s' (CLI), but got '%s'", cliMethod, baseProfile.Auth.Method)
				return false
			}

			return true
		},
		genValidAuthMethod,
		genValidAuthMethod,
	))

	// Property 4: CLI flag output format should override file output format
	properties.Property("CLI flag output format should override file output format", prop.ForAll(
		func(fileFormat string, cliFormat string) bool {
			// Skip if formats are the same
			if fileFormat == cliFormat {
				return true
			}

			// Create base profile from file
			baseProfile := &Profile{
				Name: "test",
				Jenkins: JenkinsConfig{
					URL:     "https://jenkins.example.com",
					Timeout: 30 * time.Second,
				},
				Auth: AuthConfig{
					Method:    "token",
					TokenFile: "~/.jctl/credentials",
				},
				Output: OutputConfig{
					Format: fileFormat,
					Color:  true,
				},
			}

			// Create CLI override profile
			cliOverride := &Profile{
				Output: OutputConfig{
					Format: cliFormat,
				},
			}

			// Merge CLI override into base profile
			baseProfile.MergeProfile(cliOverride)

			// Verify CLI output format took precedence
			if baseProfile.Output.Format != cliFormat {
				t.Logf("Expected output format to be '%s' (CLI), but got '%s'", cliFormat, baseProfile.Output.Format)
				return false
			}

			return true
		},
		genValidOutputFormat,
		genValidOutputFormat,
	))

	// Property 5: CLI flag color setting should override file color setting
	properties.Property("CLI flag color setting should override file color setting", prop.ForAll(
		func(fileColor bool, cliColor bool) bool {
			// Skip if color settings are the same
			if fileColor == cliColor {
				return true
			}

			// Create base profile from file
			baseProfile := &Profile{
				Name: "test",
				Jenkins: JenkinsConfig{
					URL:     "https://jenkins.example.com",
					Timeout: 30 * time.Second,
				},
				Auth: AuthConfig{
					Method:    "token",
					TokenFile: "~/.jctl/credentials",
				},
				Output: OutputConfig{
					Format: "text",
					Color:  fileColor,
				},
			}

			// Create CLI override profile
			cliOverride := &Profile{
				Output: OutputConfig{
					Color: cliColor,
				},
			}

			// Merge CLI override into base profile
			baseProfile.MergeProfile(cliOverride)

			// Verify CLI color setting took precedence
			if baseProfile.Output.Color != cliColor {
				t.Logf("Expected color to be %v (CLI), but got %v", cliColor, baseProfile.Output.Color)
				return false
			}

			return true
		},
		genBool,
		genBool,
	))

	// Property 6: Multiple CLI flags should all override file values
	properties.Property("multiple CLI flags should all override file values", prop.ForAll(
		func(fileURL string, cliURL string, fileTimeout time.Duration, cliTimeout time.Duration, fileFormat string, cliFormat string) bool {
			// Skip if any values are the same
			if fileURL == cliURL || fileTimeout == cliTimeout || fileFormat == cliFormat {
				return true
			}

			// Create base profile from file
			baseProfile := &Profile{
				Name: "test",
				Jenkins: JenkinsConfig{
					URL:     fileURL,
					Timeout: fileTimeout,
				},
				Auth: AuthConfig{
					Method:    "token",
					TokenFile: "~/.jctl/credentials",
				},
				Output: OutputConfig{
					Format: fileFormat,
					Color:  true,
				},
			}

			// Create CLI override profile with multiple overrides
			cliOverride := &Profile{
				Jenkins: JenkinsConfig{
					URL:     cliURL,
					Timeout: cliTimeout,
				},
				Output: OutputConfig{
					Format: cliFormat,
				},
			}

			// Merge CLI override into base profile
			baseProfile.MergeProfile(cliOverride)

			// Verify all CLI values took precedence
			if baseProfile.Jenkins.URL != cliURL {
				t.Logf("Expected URL to be '%s' (CLI), but got '%s'", cliURL, baseProfile.Jenkins.URL)
				return false
			}
			if baseProfile.Jenkins.Timeout != cliTimeout {
				t.Logf("Expected timeout to be %v (CLI), but got %v", cliTimeout, baseProfile.Jenkins.Timeout)
				return false
			}
			if baseProfile.Output.Format != cliFormat {
				t.Logf("Expected output format to be '%s' (CLI), but got '%s'", cliFormat, baseProfile.Output.Format)
				return false
			}

			return true
		},
		genValidURL,
		genValidURL,
		genValidTimeout,
		genValidTimeout,
		genValidOutputFormat,
		genValidOutputFormat,
	))

	// Property 7: Empty CLI override should not change file values
	properties.Property("empty CLI override should not change file values", prop.ForAll(
		func(fileURL string, fileTimeout time.Duration, fileFormat string) bool {
			// Create base profile from file
			baseProfile := &Profile{
				Name: "test",
				Jenkins: JenkinsConfig{
					URL:     fileURL,
					Timeout: fileTimeout,
				},
				Auth: AuthConfig{
					Method:    "token",
					TokenFile: "~/.jctl/credentials",
				},
				Output: OutputConfig{
					Format: fileFormat,
					Color:  true,
				},
			}

			// Store original values
			originalURL := baseProfile.Jenkins.URL
			originalTimeout := baseProfile.Jenkins.Timeout
			originalFormat := baseProfile.Output.Format

			// Create empty CLI override profile (no values set)
			cliOverride := &Profile{
				Jenkins: JenkinsConfig{
					URL:     "", // Empty means no override
					Timeout: 0,  // Zero means no override
				},
				Output: OutputConfig{
					Format: "", // Empty means no override
				},
			}

			// Merge empty CLI override into base profile
			baseProfile.MergeProfile(cliOverride)

			// Verify file values were preserved
			if baseProfile.Jenkins.URL != originalURL {
				t.Logf("Expected URL to remain '%s', but got '%s'", originalURL, baseProfile.Jenkins.URL)
				return false
			}
			if baseProfile.Jenkins.Timeout != originalTimeout {
				t.Logf("Expected timeout to remain %v, but got %v", originalTimeout, baseProfile.Jenkins.Timeout)
				return false
			}
			if baseProfile.Output.Format != originalFormat {
				t.Logf("Expected output format to remain '%s', but got '%s'", originalFormat, baseProfile.Output.Format)
				return false
			}

			return true
		},
		genValidURL,
		genValidTimeout,
		genValidOutputFormat,
	))

	properties.TestingRun(t)
}
