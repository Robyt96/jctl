package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// APIError represents an error from the Jenkins API
type APIError struct {
	StatusCode int
	Status     string
	Message    string
	Details    string
}

// Error implements the error interface
func (e *APIError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s - %s", e.Status, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Status, e.Message)
}

// parseError parses an HTTP error response into a user-friendly error
func parseError(resp *http.Response) error {
	defer resp.Body.Close()

	apiErr := &APIError{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		apiErr.Message = "Failed to read error response"
		return apiErr
	}

	// Try to parse as JSON error response
	var jsonErr struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}

	if err := json.Unmarshal(body, &jsonErr); err == nil {
		if jsonErr.Message != "" {
			apiErr.Details = jsonErr.Message
		} else if jsonErr.Error != "" {
			apiErr.Details = jsonErr.Error
		}
	}

	// Map status codes to user-friendly messages
	switch resp.StatusCode {
	case http.StatusNotFound:
		apiErr.Message = "Resource not found"
		if apiErr.Details == "" {
			apiErr.Details = "The requested pipeline or build does not exist"
		}

	case http.StatusUnauthorized:
		apiErr.Message = "Authentication required"
		if apiErr.Details == "" {
			apiErr.Details = "Your authentication token is invalid or has expired. Run 'jctl auth login' to authenticate"
		}

	case http.StatusForbidden:
		apiErr.Message = "Access denied"
		if apiErr.Details == "" {
			apiErr.Details = "You do not have permission to perform this operation"
		}

	case http.StatusBadRequest:
		apiErr.Message = "Invalid request"
		if apiErr.Details == "" {
			apiErr.Details = "The request parameters are invalid or missing required fields"
		}

	case http.StatusInternalServerError:
		apiErr.Message = "Jenkins server error"
		if apiErr.Details == "" {
			apiErr.Details = "The Jenkins server encountered an internal error. Please try again later"
		}

	case http.StatusServiceUnavailable:
		apiErr.Message = "Jenkins server unavailable"
		if apiErr.Details == "" {
			apiErr.Details = "The Jenkins server is temporarily unavailable. Please try again later"
		}

	case http.StatusTooManyRequests:
		apiErr.Message = "Rate limit exceeded"
		if apiErr.Details == "" {
			apiErr.Details = "Too many requests. Please wait before trying again"
		}
		// Check for Retry-After header
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			apiErr.Details = fmt.Sprintf("%s (retry after %s seconds)", apiErr.Details, retryAfter)
		}

	case http.StatusGatewayTimeout, http.StatusBadGateway:
		apiErr.Message = "Gateway error"
		if apiErr.Details == "" {
			apiErr.Details = "Unable to reach Jenkins server. Check your network connection and Jenkins URL"
		}

	default:
		apiErr.Message = "Request failed"
		if apiErr.Details == "" {
			// Use body as details if we couldn't parse JSON
			bodyStr := string(body)
			if len(bodyStr) > 200 {
				bodyStr = bodyStr[:200] + "..."
			}
			apiErr.Details = bodyStr
		}
	}

	return apiErr
}

// IsNotFoundError checks if an error is a 404 Not Found error
func IsNotFoundError(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}

// IsAuthError checks if an error is an authentication error
func IsAuthError(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden
	}
	return false
}

// IsNetworkError checks if an error is a network-related error
func IsNetworkError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection timeout") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "network is unreachable")
}

// FormatError formats an error into a user-friendly message with suggestions
func FormatError(err error, context string) string {
	if err == nil {
		return ""
	}

	var message strings.Builder
	message.WriteString(fmt.Sprintf("Error: %s\n", context))

	// Check error type and provide specific details and suggestions
	if apiErr, ok := err.(*APIError); ok {
		message.WriteString(fmt.Sprintf("Details: %s\n", apiErr.Error()))

		// Add suggestions based on error type
		switch apiErr.StatusCode {
		case http.StatusNotFound:
			message.WriteString("Suggestion: Use 'jctl pipelines list' to see available pipelines\n")

		case http.StatusUnauthorized, http.StatusForbidden:
			message.WriteString("Suggestion: Run 'jctl auth login' to authenticate with Jenkins\n")

		case http.StatusBadRequest:
			message.WriteString("Suggestion: Check that all required parameters are provided and valid\n")

		case http.StatusInternalServerError, http.StatusServiceUnavailable:
			message.WriteString("Suggestion: Try again in a few moments. If the problem persists, contact your Jenkins administrator\n")

		case http.StatusTooManyRequests:
			message.WriteString("Suggestion: Wait a moment before retrying your request\n")
		}
	} else if IsNetworkError(err) {
		message.WriteString(fmt.Sprintf("Details: %s\n", err.Error()))
		message.WriteString("Suggestion: Check that your Jenkins URL is correct and that you have network connectivity\n")
	} else {
		message.WriteString(fmt.Sprintf("Details: %s\n", err.Error()))
	}

	return message.String()
}
