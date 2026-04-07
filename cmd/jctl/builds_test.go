package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/user/jctl/internal/client"
	"github.com/user/jctl/internal/config"
	"gopkg.in/yaml.v3"
)

// TestFormatParamsJSON tests JSON output formatting
func TestFormatParamsJSON(t *testing.T) {
	params := []client.Parameter{
		{Name: "ENVIRONMENT", Value: "production"},
		{Name: "VERSION", Value: "1.2.3"},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := formatParamsJSON(params)
	if err != nil {
		t.Fatalf("formatParamsJSON failed: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)

	// Verify JSON is valid
	var result []client.Parameter
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	// Verify content
	if len(result) != 2 {
		t.Errorf("Expected 2 parameters, got %d", len(result))
	}
	if result[0].Name != "ENVIRONMENT" || result[0].Value != "production" {
		t.Errorf("First parameter mismatch: got %+v", result[0])
	}
	if result[1].Name != "VERSION" || result[1].Value != "1.2.3" {
		t.Errorf("Second parameter mismatch: got %+v", result[1])
	}
}

// TestFormatParamsYAML tests YAML output formatting
func TestFormatParamsYAML(t *testing.T) {
	params := []client.Parameter{
		{Name: "ENVIRONMENT", Value: "production"},
		{Name: "VERSION", Value: "1.2.3"},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := formatParamsYAML(params)
	if err != nil {
		t.Fatalf("formatParamsYAML failed: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)

	// Verify YAML is valid
	var result []client.Parameter
	if err := yaml.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Invalid YAML output: %v", err)
	}

	// Verify content
	if len(result) != 2 {
		t.Errorf("Expected 2 parameters, got %d", len(result))
	}
	if result[0].Name != "ENVIRONMENT" || result[0].Value != "production" {
		t.Errorf("First parameter mismatch: got %+v", result[0])
	}
	if result[1].Name != "VERSION" || result[1].Value != "1.2.3" {
		t.Errorf("Second parameter mismatch: got %+v", result[1])
	}
}

// TestFormatParamsText tests text output formatting
func TestFormatParamsText(t *testing.T) {
	params := []client.Parameter{
		{Name: "ENVIRONMENT", Value: "production"},
		{Name: "VERSION", Value: "1.2.3"},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := formatParamsText(params, "my-pipeline", 42)
	if err != nil {
		t.Fatalf("formatParamsText failed: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify output contains expected elements
	expectedStrings := []string{
		"Parameters for build #42 of pipeline my-pipeline:",
		"NAME",
		"VALUE",
		"ENVIRONMENT",
		"production",
		"VERSION",
		"1.2.3",
	}

	for _, expected := range expectedStrings {
		if !bytes.Contains([]byte(output), []byte(expected)) {
			t.Errorf("Output missing expected string: %s\nGot: %s", expected, output)
		}
	}
}

// TestExtractParameters tests the ExtractParameters method
func TestExtractParameters(t *testing.T) {
	tests := []struct {
		name     string
		build    client.Build
		expected []client.Parameter
	}{
		{
			name: "build with parameters",
			build: client.Build{
				Number: 42,
				Actions: []client.BuildAction{
					{
						Parameters: []client.Parameter{
							{Name: "ENV", Value: "prod"},
							{Name: "VERSION", Value: "1.0.0"},
						},
					},
				},
			},
			expected: []client.Parameter{
				{Name: "ENV", Value: "prod"},
				{Name: "VERSION", Value: "1.0.0"},
			},
		},
		{
			name: "build with no parameters",
			build: client.Build{
				Number:  42,
				Actions: []client.BuildAction{},
			},
			expected: []client.Parameter{},
		},
		{
			name: "build with multiple actions",
			build: client.Build{
				Number: 42,
				Actions: []client.BuildAction{
					{
						Parameters: []client.Parameter{
							{Name: "PARAM1", Value: "value1"},
						},
					},
					{
						Parameters: []client.Parameter{
							{Name: "PARAM2", Value: "value2"},
						},
					},
				},
			},
			expected: []client.Parameter{
				{Name: "PARAM1", Value: "value1"},
				{Name: "PARAM2", Value: "value2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.build.ExtractParameters()

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d parameters, got %d", len(tt.expected), len(result))
				return
			}

			for i, param := range result {
				if param.Name != tt.expected[i].Name || param.Value != tt.expected[i].Value {
					t.Errorf("Parameter %d mismatch: expected %+v, got %+v", i, tt.expected[i], param)
				}
			}
		})
	}
}

// MockClient is a mock implementation of the Jenkins API client for testing
type MockClient struct {
	GetBuildInfoFunc func(ctx context.Context, jobName string, buildNumber int) (*client.Build, error)
}

func (m *MockClient) GetBuildInfo(ctx context.Context, jobName string, buildNumber int) (*client.Build, error) {
	if m.GetBuildInfoFunc != nil {
		return m.GetBuildInfoFunc(ctx, jobName, buildNumber)
	}
	return nil, nil
}

// TestBuildsParamsArgValidation tests argument validation for the builds params command
func TestBuildsParamsArgValidation(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid arguments",
			args:        []string{"my-pipeline", "42"},
			expectError: false,
		},
		{
			name:        "no arguments",
			args:        []string{},
			expectError: true,
			errorMsg:    "requires exactly 2 arguments",
		},
		{
			name:        "one argument",
			args:        []string{"my-pipeline"},
			expectError: true,
			errorMsg:    "requires exactly 2 arguments",
		},
		{
			name:        "too many arguments",
			args:        []string{"my-pipeline", "42", "extra"},
			expectError: true,
			errorMsg:    "too many arguments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the Args validation function
			err := buildsParamsCmd.Args(buildsParamsCmd, tt.args)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && !bytes.Contains([]byte(err.Error()), []byte(tt.errorMsg)) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestBuildNumberValidation tests build number validation in runBuildsParams
func TestBuildNumberValidation(t *testing.T) {
	// Save original profile and restore after test
	originalProfile := profile
	defer func() { profile = originalProfile }()

	// Set up a test profile with Jenkins URL
	profile = &config.Profile{
		Name: "test",
		Jenkins: config.JenkinsConfig{
			URL:     "https://jenkins.example.com",
			Timeout: 30,
		},
		Auth: config.AuthConfig{
			TokenFile: "/tmp/test-token",
		},
		Output: config.OutputConfig{
			Format: "text",
		},
	}

	tests := []struct {
		name        string
		buildNumber string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid positive integer",
			buildNumber: "42",
			expectError: false,
		},
		{
			name:        "zero build number",
			buildNumber: "0",
			expectError: true,
			errorMsg:    "Build number must be a positive integer",
		},
		{
			name:        "negative build number",
			buildNumber: "-5",
			expectError: true,
			errorMsg:    "Build number must be a positive integer",
		},
		{
			name:        "non-numeric build number",
			buildNumber: "abc",
			expectError: true,
			errorMsg:    "Build number must be a positive integer",
		},
		{
			name:        "float build number",
			buildNumber: "42.5",
			expectError: true,
			errorMsg:    "Build number must be a positive integer",
		},
		{
			name:        "build number with spaces",
			buildNumber: "42 ",
			expectError: true,
			errorMsg:    "Build number must be a positive integer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock command
			cmd := buildsParamsCmd

			// Test with the build number
			args := []string{"test-pipeline", tt.buildNumber}

			// Call runBuildsParams - it will fail at API call but we're testing validation
			err := runBuildsParams(cmd, args)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && !bytes.Contains([]byte(err.Error()), []byte(tt.errorMsg)) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorMsg, err.Error())
				}
			} else {
				// For valid cases, we expect an error from the API call (since we're not mocking it)
				// but NOT a validation error
				if err != nil && bytes.Contains([]byte(err.Error()), []byte("Build number must be a positive integer")) {
					t.Errorf("Got validation error for valid input: %v", err)
				}
			}
		})
	}
}

// TestParameterStringValue tests the StringValue method with different types
func TestParameterStringValue(t *testing.T) {
	tests := []struct {
		name     string
		param    client.Parameter
		expected string
	}{
		{
			name:     "string value",
			param:    client.Parameter{Name: "ENV", Value: "production"},
			expected: "production",
		},
		{
			name:     "boolean true",
			param:    client.Parameter{Name: "ENABLED", Value: true},
			expected: "true",
		},
		{
			name:     "boolean false",
			param:    client.Parameter{Name: "DISABLED", Value: false},
			expected: "false",
		},
		{
			name:     "integer",
			param:    client.Parameter{Name: "COUNT", Value: 42},
			expected: "42",
		},
		{
			name:     "float",
			param:    client.Parameter{Name: "VERSION", Value: 1.5},
			expected: "1.5",
		},
		{
			name:     "nil value",
			param:    client.Parameter{Name: "EMPTY", Value: nil},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.param.StringValue()
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestFormatParamsWithMixedTypes tests formatting with mixed parameter types
func TestFormatParamsWithMixedTypes(t *testing.T) {
	params := []client.Parameter{
		{Name: "ENVIRONMENT", Value: "production"},
		{Name: "ENABLED", Value: true},
		{Name: "COUNT", Value: 5},
		{Name: "VERSION", Value: 1.2},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := formatParamsText(params, "test-pipeline", 42)
	if err != nil {
		t.Fatalf("formatParamsText failed: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify output contains expected values
	expectedStrings := []string{
		"ENVIRONMENT",
		"production",
		"ENABLED",
		"true",
		"COUNT",
		"5",
		"VERSION",
		"1.2",
	}

	for _, expected := range expectedStrings {
		if !bytes.Contains([]byte(output), []byte(expected)) {
			t.Errorf("Output missing expected string: %s\nGot: %s", expected, output)
		}
	}
}
