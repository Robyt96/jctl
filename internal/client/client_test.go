package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/user/jctl/internal/auth"
)

// TestGetBuildInfo_IncludesParametersInAPIRequest verifies that the API request
// includes actions[parameters[name,value]] in the tree parameter
func TestGetBuildInfo_IncludesParametersInAPIRequest(t *testing.T) {
	// Create a test server that captures the request
	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path + "?" + r.URL.RawQuery

		// Return a mock build response with parameters
		response := Build{
			Number:    42,
			Result:    "SUCCESS",
			Timestamp: 1234567890,
			Duration:  5000,
			Building:  false,
			URL:       "http://jenkins/job/test-job/42/",
			Actions: []BuildAction{
				{
					Parameters: []Parameter{
						{Name: "ENV", Value: "production"},
						{Name: "VERSION", Value: "1.2.3"},
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with auth manager
	// Use a temporary file for credentials
	tmpDir := t.TempDir()
	credsFile := tmpDir + "/credentials.yaml"
	authMgr := auth.NewManager(credsFile)
	authMgr.StoreToken("test", &auth.Token{
		Type:     "api_token",
		Username: "testuser",
		Value:    "testtoken",
	})

	client := NewClient(server.URL, 30*time.Second, authMgr, "test", false)

	// Call GetBuildInfo
	build, err := client.GetBuildInfo(context.Background(), "test-job", 42)
	if err != nil {
		t.Fatalf("GetBuildInfo failed: %v", err)
	}

	// Verify the API request included the tree parameter with actions[parameters[name,value]]
	expectedTreeParam := "tree=number,result,timestamp,duration,building,url,actions[parameters[name,value]]"
	if !strings.Contains(capturedPath, expectedTreeParam) {
		t.Errorf("API request did not include expected tree parameter.\nExpected to contain: %s\nActual path: %s", expectedTreeParam, capturedPath)
	}

	// Verify the response includes parameters
	if len(build.Actions) == 0 {
		t.Error("Expected build to have actions, got none")
	}

	if len(build.Actions[0].Parameters) != 2 {
		t.Errorf("Expected 2 parameters, got %d", len(build.Actions[0].Parameters))
	}
}

// TestGetBuildInfo_URLEncodesfolderPaths verifies that folder paths are correctly
// URL-encoded in the API request
func TestGetBuildInfo_URLEncodesFolderPaths(t *testing.T) {
	testCases := []struct {
		name         string
		jobName      string
		expectedPath string
	}{
		{
			name:         "simple job name",
			jobName:      "my-job",
			expectedPath: "/job/my-job/42/api/json",
		},
		{
			name:         "job in folder",
			jobName:      "folder/my-job",
			expectedPath: "/job/folder/job/my-job/42/api/json",
		},
		{
			name:         "job in nested folders",
			jobName:      "folder1/folder2/my-job",
			expectedPath: "/job/folder1/job/folder2/job/my-job/42/api/json",
		},
		{
			name:         "job with spaces",
			jobName:      "my folder/my job",
			expectedPath: "/job/my%20folder/job/my%20job/42/api/json",
		},
		{
			name:         "job with special characters",
			jobName:      "folder-1/job_name",
			expectedPath: "/job/folder-1/job/job_name/42/api/json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test server that captures the request
			var capturedPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedPath = r.URL.Path

				// Return a minimal build response
				response := Build{
					Number:   42,
					Result:   "SUCCESS",
					Building: false,
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			// Create client with auth manager
			// Use a temporary file for credentials
			tmpDir := t.TempDir()
			credsFile := tmpDir + "/credentials.yaml"
			authMgr := auth.NewManager(credsFile)
			authMgr.StoreToken("test", &auth.Token{
				Type:     "api_token",
				Username: "testuser",
				Value:    "testtoken",
			})

			client := NewClient(server.URL, 30*time.Second, authMgr, "test", false)

			// Call GetBuildInfo
			_, err := client.GetBuildInfo(context.Background(), tc.jobName, 42)
			if err != nil {
				t.Fatalf("GetBuildInfo failed: %v", err)
			}

			// Verify the path was correctly encoded
			// Decode the captured path to compare
			decodedPath, err := url.PathUnescape(capturedPath)
			if err != nil {
				t.Fatalf("Failed to decode path: %v", err)
			}

			expectedDecoded, err := url.PathUnescape(tc.expectedPath)
			if err != nil {
				t.Fatalf("Failed to decode expected path: %v", err)
			}

			if decodedPath != expectedDecoded {
				t.Errorf("Path mismatch.\nExpected: %s\nGot: %s", tc.expectedPath, capturedPath)
			}
		})
	}
}

// TestGetBuildInfo_ExtractParameters verifies that parameters can be extracted
// from the build's actions array
func TestGetBuildInfo_ExtractParameters(t *testing.T) {
	testCases := []struct {
		name           string
		actions        []BuildAction
		expectedParams int
	}{
		{
			name:           "no actions",
			actions:        []BuildAction{},
			expectedParams: 0,
		},
		{
			name: "single action with parameters",
			actions: []BuildAction{
				{
					Parameters: []Parameter{
						{Name: "ENV", Value: "prod"},
						{Name: "VERSION", Value: "1.0"},
					},
				},
			},
			expectedParams: 2,
		},
		{
			name: "multiple actions with parameters",
			actions: []BuildAction{
				{
					Parameters: []Parameter{
						{Name: "ENV", Value: "prod"},
					},
				},
				{
					Parameters: []Parameter{
						{Name: "VERSION", Value: "1.0"},
						{Name: "REGION", Value: "us-east-1"},
					},
				},
			},
			expectedParams: 3,
		},
		{
			name: "action without parameters",
			actions: []BuildAction{
				{
					Parameters: []Parameter{},
				},
			},
			expectedParams: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			build := &Build{
				Number:  1,
				Actions: tc.actions,
			}

			params := build.ExtractParameters()

			if len(params) != tc.expectedParams {
				t.Errorf("Expected %d parameters, got %d", tc.expectedParams, len(params))
			}
		})
	}
}
