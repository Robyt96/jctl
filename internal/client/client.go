package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/user/jctl/internal/auth"
)

// Client is the Jenkins API client
type Client struct {
	baseURL    string
	httpClient *http.Client
	authMgr    *auth.Manager
	profile    string
	verbose    bool
}

// NewClient creates a new Jenkins API client
func NewClient(baseURL string, timeout time.Duration, authMgr *auth.Manager, profile string, verbose bool) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
		authMgr: authMgr,
		profile: profile,
		verbose: verbose,
	}
}

// doRequest performs an HTTP request with authentication and logging
func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	// Build full URL
	fullURL := c.baseURL + path

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header
	token, err := c.authMgr.GetToken(c.profile)
	if err != nil {
		return nil, fmt.Errorf("authentication required: %w", err)
	}

	// Set authorization header based on token type
	if token.Type == "oauth" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.Value))
	} else {
		// API token authentication uses Basic Auth with username:token
		if token.Username == "" {
			return nil, fmt.Errorf("username is required for API token authentication")
		}
		req.SetBasicAuth(token.Username, token.Value)
	}

	// Set content type for POST requests
	if method == http.MethodPost && body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	// Log request if verbose mode is enabled
	if c.verbose {
		c.logRequest(req)
	}

	// Perform request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Log response if verbose mode is enabled
	if c.verbose {
		c.logResponse(resp)
	}

	return resp, nil
}

// logRequest logs HTTP request details
func (c *Client) logRequest(req *http.Request) {
	fmt.Printf("[DEBUG] Request: %s %s\n", req.Method, req.URL.String())
	fmt.Printf("[DEBUG] Headers:\n")
	for key, values := range req.Header {
		// Mask authorization header for security
		if key == "Authorization" {
			fmt.Printf("[DEBUG]   %s: [REDACTED]\n", key)
		} else {
			for _, value := range values {
				fmt.Printf("[DEBUG]   %s: %s\n", key, value)
			}
		}
	}
}

// logResponse logs HTTP response details
func (c *Client) logResponse(resp *http.Response) {
	fmt.Printf("[DEBUG] Response: %d %s\n", resp.StatusCode, resp.Status)
	fmt.Printf("[DEBUG] Headers:\n")
	for key, values := range resp.Header {
		for _, value := range values {
			fmt.Printf("[DEBUG]   %s: %s\n", key, value)
		}
	}
}

// get performs a GET request
func (c *Client) get(ctx context.Context, path string) (*http.Response, error) {
	return c.doRequest(ctx, http.MethodGet, path, nil)
}

// post performs a POST request
func (c *Client) post(ctx context.Context, path string, data url.Values) (*http.Response, error) {
	var body io.Reader
	if data != nil {
		body = strings.NewReader(data.Encode())
	}
	return c.doRequest(ctx, http.MethodPost, path, body)
}

// parseJSON parses JSON response body into the provided interface
func parseJSON(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return nil
}

// readBody reads the entire response body as a string
func readBody(resp *http.Response) (string, error) {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

// buildQueryString builds a query string from parameters
func buildQueryString(params map[string]string) string {
	if len(params) == 0 {
		return ""
	}

	values := url.Values{}
	for key, value := range params {
		values.Set(key, value)
	}

	return "?" + values.Encode()
}

// ListJobs retrieves all jobs from Jenkins, optionally within a specific folder
// folderPath should be in the format "folder1/folder2" or empty string for root
func (c *Client) ListJobs(ctx context.Context, folderPath string) ([]Job, error) {
	// Build the path based on folder
	var path string
	if folderPath == "" {
		// Root level
		path = "/api/json?tree=jobs[name,url,color,description,buildable,_class,lastBuild[number,url]]"
	} else {
		// Inside a folder - need to URL encode each folder segment
		segments := strings.Split(folderPath, "/")
		encodedSegments := make([]string, len(segments))
		for i, segment := range segments {
			encodedSegments[i] = url.PathEscape(segment)
		}
		folderURL := "/job/" + strings.Join(encodedSegments, "/job/")
		path = folderURL + "/api/json?tree=jobs[name,url,color,description,buildable,_class,lastBuild[number,url]]"
	}

	resp, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}

	var result JobsResponse
	if err := parseJSON(resp, &result); err != nil {
		return nil, err
	}

	return result.Jobs, nil
}

// GetJob retrieves details for a specific job
// name can be a simple name or a folder path like "folder1/folder2/jobname"
func (c *Client) GetJob(ctx context.Context, name string) (*Job, error) {
	// Build the path - handle folder paths by encoding each segment
	segments := strings.Split(name, "/")
	encodedSegments := make([]string, len(segments))
	for i, segment := range segments {
		encodedSegments[i] = url.PathEscape(segment)
	}
	jobPath := "/job/" + strings.Join(encodedSegments, "/job/")
	// Include property information to get parameter definitions
	path := jobPath + "/api/json?tree=name,url,description,color,buildable,_class,lastBuild[number,url],property[parameterDefinitions[name,type,description,defaultParameterValue[value]]]"

	resp, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}

	var job Job
	if err := parseJSON(resp, &job); err != nil {
		return nil, err
	}

	return &job, nil
}

// ListBuilds retrieves all builds for a specific job
// jobName can be a simple name or a folder path like "folder1/folder2/jobname"
func (c *Client) ListBuilds(ctx context.Context, jobName string) ([]Build, error) {
	// Build the path - handle folder paths by encoding each segment
	segments := strings.Split(jobName, "/")
	encodedSegments := make([]string, len(segments))
	for i, segment := range segments {
		encodedSegments[i] = url.PathEscape(segment)
	}
	jobPath := "/job/" + strings.Join(encodedSegments, "/job/")
	path := jobPath + "/api/json?tree=builds[number,result,timestamp,duration,building,url]"

	resp, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}

	var job Job
	if err := parseJSON(resp, &job); err != nil {
		return nil, err
	}

	return job.Builds, nil
}

// GetBuildLog retrieves the console log for a specific build
// jobName can be a simple name or a folder path like "folder1/folder2/jobname"
func (c *Client) GetBuildLog(ctx context.Context, jobName string, buildNumber int) (string, error) {
	// Build the path - handle folder paths by encoding each segment
	segments := strings.Split(jobName, "/")
	encodedSegments := make([]string, len(segments))
	for i, segment := range segments {
		encodedSegments[i] = url.PathEscape(segment)
	}
	jobPath := "/job/" + strings.Join(encodedSegments, "/job/")
	path := fmt.Sprintf("%s/%d/consoleText", jobPath, buildNumber)

	resp, err := c.get(ctx, path)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", parseError(resp)
	}

	log, err := readBody(resp)
	if err != nil {
		return "", err
	}

	return log, nil
}

// TriggerBuild triggers a new build for a job, optionally with parameters
// jobName can be a simple name or a folder path like "folder1/folder2/jobname"
func (c *Client) TriggerBuild(ctx context.Context, jobName string, params map[string]string) (*QueueItem, error) {
	// Build the path - handle folder paths by encoding each segment
	segments := strings.Split(jobName, "/")
	encodedSegments := make([]string, len(segments))
	for i, segment := range segments {
		encodedSegments[i] = url.PathEscape(segment)
	}
	jobPath := "/job/" + strings.Join(encodedSegments, "/job/")

	var path string
	var data url.Values

	if len(params) > 0 {
		// Build with parameters
		path = jobPath + "/buildWithParameters"
		data = url.Values{}
		for key, value := range params {
			data.Set(key, value)
		}
	} else {
		// Build without parameters
		path = jobPath + "/build"
	}

	resp, err := c.post(ctx, path, data)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}

	// Jenkins returns the queue item location in the Location header
	location := resp.Header.Get("Location")
	if location == "" {
		return &QueueItem{}, nil
	}

	// Extract queue ID from location
	// Location format: http://jenkins/queue/item/{id}/
	parts := strings.Split(strings.TrimSuffix(location, "/"), "/")
	if len(parts) > 0 {
		queueIDStr := parts[len(parts)-1]
		queueID, err := strconv.Atoi(queueIDStr)
		if err != nil {
			return &QueueItem{
				Task: TaskRef{
					Name: jobName,
				},
				Why: fmt.Sprintf("Build queued (ID: %s)", queueIDStr),
			}, nil
		}
		return &QueueItem{
			ID: queueID,
			Task: TaskRef{
				Name: jobName,
			},
			Why: fmt.Sprintf("Build queued (ID: %d)", queueID),
		}, nil
	}

	return &QueueItem{}, nil
}

// GetBuildInfo retrieves detailed information about a specific build
// jobName can be a simple name or a folder path like "folder1/folder2/jobname"
func (c *Client) GetBuildInfo(ctx context.Context, jobName string, buildNumber int) (*Build, error) {
	// Build the path - handle folder paths by encoding each segment
	segments := strings.Split(jobName, "/")
	encodedSegments := make([]string, len(segments))
	for i, segment := range segments {
		encodedSegments[i] = url.PathEscape(segment)
	}
	jobPath := "/job/" + strings.Join(encodedSegments, "/job/")
	path := fmt.Sprintf("%s/%d/api/json", jobPath, buildNumber)

	resp, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}

	var build Build
	if err := parseJSON(resp, &build); err != nil {
		return nil, err
	}

	return &build, nil
}

// GetProgressiveLog retrieves progressive log content from a specific byte offset
// This is used for streaming logs as a build progresses
// jobName can be a simple name or a folder path like "folder1/folder2/jobname"
func (c *Client) GetProgressiveLog(ctx context.Context, jobName string, buildNumber int, startByte int64) (*ProgressiveLogResponse, error) {
	// Build the path - handle folder paths by encoding each segment
	segments := strings.Split(jobName, "/")
	encodedSegments := make([]string, len(segments))
	for i, segment := range segments {
		encodedSegments[i] = url.PathEscape(segment)
	}
	jobPath := "/job/" + strings.Join(encodedSegments, "/job/")
	path := fmt.Sprintf("%s/%d/logText/progressiveText?start=%d", jobPath, buildNumber, startByte)

	resp, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}

	// Read the log content
	content, err := readBody(resp)
	if err != nil {
		return nil, err
	}

	// Parse X-Text-Size header to get next offset
	textSizeStr := resp.Header.Get("X-Text-Size")
	var nextOffset int64
	if textSizeStr != "" {
		fmt.Sscanf(textSizeStr, "%d", &nextOffset)
	} else {
		// If header is missing, calculate based on start + content length
		nextOffset = startByte + int64(len(content))
	}

	// Parse X-More-Data header to check if build is still running
	moreDataStr := resp.Header.Get("X-More-Data")
	hasMoreData := moreDataStr == "true"

	return &ProgressiveLogResponse{
		Content:     content,
		NextOffset:  nextOffset,
		HasMoreData: hasMoreData,
	}, nil
}

// GetQueueItem retrieves information about a queued build
// This is used to check if a queued build has started and get its build number
func (c *Client) GetQueueItem(ctx context.Context, queueID int) (*QueueItem, error) {
	path := fmt.Sprintf("/queue/item/%d/api/json", queueID)

	resp, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}

	var queueItem QueueItem
	if err := parseJSON(resp, &queueItem); err != nil {
		return nil, err
	}

	return &queueItem, nil
}

// GetPendingInputs retrieves pending input steps for a build
// This uses the workflow API to detect when a build is waiting for user input
// jobName can be a simple name or a folder path like "folder1/folder2/jobname"
func (c *Client) GetPendingInputs(ctx context.Context, jobName string, buildNumber int) ([]InputStep, error) {
	// Build the path - handle folder paths by encoding each segment
	segments := strings.Split(jobName, "/")
	encodedSegments := make([]string, len(segments))
	for i, segment := range segments {
		encodedSegments[i] = url.PathEscape(segment)
	}
	jobPath := "/job/" + strings.Join(encodedSegments, "/job/")

	// First, try the workflow API
	path := fmt.Sprintf("%s/%d/wfapi/describe", jobPath, buildNumber)

	resp, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}

	// If the workflow API is not available (404), try alternative method
	if resp.StatusCode == http.StatusNotFound {
		return c.getPendingInputsFromBuildAPI(ctx, jobPath, buildNumber)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}

	var workflow WorkflowDescription
	if err := parseJSON(resp, &workflow); err != nil {
		return nil, err
	}

	// Debug logging to see what we got
	if c.verbose {
		fmt.Fprintf(os.Stderr, "[DEBUG] Workflow status: %s, PendingInputActions count: %d\n", workflow.Status, len(workflow.PendingInputActions))
	}

	// If status is PAUSED_PENDING_INPUT but no actions in the response,
	// try the pendingInputActions endpoint directly
	if workflow.Status == "PAUSED_PENDING_INPUT" && len(workflow.PendingInputActions) == 0 {
		return c.getPendingInputsFromWorkflowAPI(ctx, jobPath, buildNumber)
	}

	// If no pending inputs found via workflow API, try the build API as fallback
	if len(workflow.PendingInputActions) == 0 {
		return c.getPendingInputsFromBuildAPI(ctx, jobPath, buildNumber)
	}

	// Convert InputActions to InputSteps
	inputSteps := make([]InputStep, 0, len(workflow.PendingInputActions))
	for _, action := range workflow.PendingInputActions {
		step := InputStep{
			ID:      action.ID,
			Message: action.Message,
			OK:      action.ProceedText,
			Abort:   action.AbortText,
		}

		// Convert input parameters
		if len(action.Inputs) > 0 {
			step.Parameters = make([]InputParameter, 0, len(action.Inputs))
			for _, input := range action.Inputs {
				param := InputParameter{
					Name:        input.Name,
					Description: input.Description,
					Type:        input.Type,
				}

				// Convert default value to string
				if input.DefaultValue != nil {
					param.DefaultValue = fmt.Sprintf("%v", input.DefaultValue)
				}

				step.Parameters = append(step.Parameters, param)
			}
		}

		inputSteps = append(inputSteps, step)
	}

	return inputSteps, nil
}

// getPendingInputsFromBuildAPI tries to get pending inputs from the build API
// This is a fallback when the workflow API doesn't return pending inputs
func (c *Client) getPendingInputsFromBuildAPI(ctx context.Context, jobPath string, buildNumber int) ([]InputStep, error) {
	// Try to get build info with actions that might contain input requests
	path := fmt.Sprintf("%s/%d/api/json?tree=actions[*[*]]", jobPath, buildNumber)

	resp, err := c.get(ctx, path)
	if err != nil {
		return []InputStep{}, nil // Return empty list on error
	}

	if resp.StatusCode != http.StatusOK {
		return []InputStep{}, nil // Return empty list on error
	}

	var buildInfo map[string]interface{}
	if err := parseJSON(resp, &buildInfo); err != nil {
		return []InputStep{}, nil // Return empty list on parse error
	}

	if c.verbose {
		fmt.Fprintf(os.Stderr, "[DEBUG] Checking build API for pending inputs\n")
	}

	// Look for InputRequiredAction in the actions array
	actions, ok := buildInfo["actions"].([]interface{})
	if !ok {
		return []InputStep{}, nil
	}

	var inputSteps []InputStep
	for _, action := range actions {
		actionMap, ok := action.(map[string]interface{})
		if !ok {
			continue
		}

		// Check if this is an input action
		class, _ := actionMap["_class"].(string)
		if !strings.Contains(class, "InputAction") && !strings.Contains(class, "InputRequiredAction") {
			continue
		}

		// Extract input information
		id, _ := actionMap["id"].(string)
		message, _ := actionMap["message"].(string)
		proceedText, _ := actionMap["proceedText"].(string)
		abortText, _ := actionMap["abortText"].(string)

		if id != "" {
			step := InputStep{
				ID:      id,
				Message: message,
				OK:      proceedText,
				Abort:   abortText,
			}

			// Try to extract parameters if present
			if params, ok := actionMap["parameters"].([]interface{}); ok {
				step.Parameters = make([]InputParameter, 0, len(params))
				for _, p := range params {
					paramMap, ok := p.(map[string]interface{})
					if !ok {
						continue
					}

					param := InputParameter{
						Name:        getString(paramMap, "name"),
						Description: getString(paramMap, "description"),
						Type:        getString(paramMap, "type"),
					}

					if defVal := paramMap["defaultValue"]; defVal != nil {
						param.DefaultValue = fmt.Sprintf("%v", defVal)
					}

					step.Parameters = append(step.Parameters, param)
				}
			}

			inputSteps = append(inputSteps, step)

			if c.verbose {
				fmt.Fprintf(os.Stderr, "[DEBUG] Found input step via build API: ID=%s, Message=%s\n", id, message)
			}
		}
	}

	return inputSteps, nil
}

// getPendingInputsFromWorkflowAPI gets pending inputs from the workflow pendingInputActions endpoint
func (c *Client) getPendingInputsFromWorkflowAPI(ctx context.Context, jobPath string, buildNumber int) ([]InputStep, error) {
	path := fmt.Sprintf("%s/%d/wfapi/pendingInputActions", jobPath, buildNumber)

	if c.verbose {
		fmt.Fprintf(os.Stderr, "[DEBUG] Fetching pending inputs from workflow API endpoint\n")
	}

	resp, err := c.get(ctx, path)
	if err != nil {
		return []InputStep{}, nil // Return empty list on error
	}

	if resp.StatusCode != http.StatusOK {
		return []InputStep{}, nil // Return empty list on error
	}

	var actions []InputAction
	if err := parseJSON(resp, &actions); err != nil {
		return []InputStep{}, nil // Return empty list on parse error
	}

	if c.verbose {
		fmt.Fprintf(os.Stderr, "[DEBUG] Found %d pending input actions from workflow API\n", len(actions))
	}

	// Convert InputActions to InputSteps
	inputSteps := make([]InputStep, 0, len(actions))
	for _, action := range actions {
		step := InputStep{
			ID:      action.ID,
			Message: action.Message,
			OK:      action.ProceedText,
			Abort:   action.AbortText,
		}

		// Convert input parameters
		if len(action.Inputs) > 0 {
			step.Parameters = make([]InputParameter, 0, len(action.Inputs))
			for _, input := range action.Inputs {
				param := InputParameter{
					Name:        input.Name,
					Description: input.Description,
					Type:        input.Type,
				}

				// Convert default value to string
				if input.DefaultValue != nil {
					param.DefaultValue = fmt.Sprintf("%v", input.DefaultValue)
				}

				step.Parameters = append(step.Parameters, param)
			}
		}

		inputSteps = append(inputSteps, step)
	}

	return inputSteps, nil
}

// getString safely extracts a string from a map
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// SubmitInput submits user input for a pending input step
// This allows the build to continue after receiving the required input
// jobName can be a simple name or a folder path like "folder1/folder2/jobname"
// params should contain values for any parameters required by the input step
func (c *Client) SubmitInput(ctx context.Context, jobName string, buildNumber int, inputID string, params map[string]string) error {
	// Build the path - handle folder paths by encoding each segment
	segments := strings.Split(jobName, "/")
	encodedSegments := make([]string, len(segments))
	for i, segment := range segments {
		encodedSegments[i] = url.PathEscape(segment)
	}
	jobPath := "/job/" + strings.Join(encodedSegments, "/job/")

	var path string
	var data url.Values

	// If no parameters, use proceedEmpty endpoint for simple approval
	if len(params) == 0 {
		path = fmt.Sprintf("%s/%d/input/%s/proceedEmpty", jobPath, buildNumber, url.PathEscape(inputID))
		data = nil
	} else {
		// With parameters, use submit endpoint
		path = fmt.Sprintf("%s/%d/input/%s/submit", jobPath, buildNumber, url.PathEscape(inputID))
		// Prepare form data with parameters
		data = url.Values{}
		for key, value := range params {
			data.Set(key, value)
		}
	}

	resp, err := c.post(ctx, path, data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Jenkins returns 200 or 302 on successful input submission
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
		return parseError(resp)
	}

	return nil
}

// AbortInput aborts a pending input step, causing the build to be aborted
// jobName can be a simple name or a folder path like "folder1/folder2/jobname"
func (c *Client) AbortInput(ctx context.Context, jobName string, buildNumber int, inputID string) error {
	// Build the path - handle folder paths by encoding each segment
	segments := strings.Split(jobName, "/")
	encodedSegments := make([]string, len(segments))
	for i, segment := range segments {
		encodedSegments[i] = url.PathEscape(segment)
	}
	jobPath := "/job/" + strings.Join(encodedSegments, "/job/")
	path := fmt.Sprintf("%s/%d/input/%s/abort", jobPath, buildNumber, url.PathEscape(inputID))

	resp, err := c.post(ctx, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Jenkins returns 200 or 302 on successful abort
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
		return parseError(resp)
	}

	return nil
}
