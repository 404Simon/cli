package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// LoginWithCookie performs a login request and returns the session cookie
// This is a lower-level function that handles cookie extraction
func LoginWithCookie(client *Client, req LoginRequest) (*LoginResponse, string, error) {
	var response LoginResponse
	sessionToken := ""

	// Use a custom request that captures cookies
	baseURL := client.BaseURL
	if baseURL == "" {
		baseURL = "https://app.pangolin.net"
	}
	if !strings.HasPrefix(baseURL, "http") {
		baseURL = "https://" + baseURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	endpoint := "/api/v1/auth/login"
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}

	url := baseURL + endpoint

	// Create HTTP client
	httpClient := &http.Client{
		Timeout: client.HTTPClient.Timeout,
	}

	// Marshal request body
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create request
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	userAgent := client.AgentName
	if userAgent == "" {
		userAgent = "pangolin-cli"
	}
	httpReq.Header.Set("User-Agent", userAgent)

	// Set CSRF token header
	if client.CSRFToken != "" {
		httpReq.Header.Set("X-CSRF-Token", client.CSRFToken)
	} else {
		// Default CSRF token value
		httpReq.Header.Set("X-CSRF-Token", "x-csrf-protection")
	}

	// Execute request
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Extract session cookie
	for _, cookie := range resp.Cookies() {
		if cookie.Name == client.SessionCookieName || cookie.Name == "p_session" {
			sessionToken = cookie.Value
			break
		}
	}

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response: %w", err)
	}

	if len(bodyBytes) == 0 {
		// Return empty response but with token if available
		return &response, sessionToken, nil
	}

	// Parse the API response structure
	var apiResp APIResponse
	if err := json.Unmarshal(bodyBytes, &apiResp); err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check if the response indicates an error
	if apiResp.Error.Bool() || !apiResp.Success {
		errorResp := ErrorResponse{
			Message: apiResp.Message,
			Status:  apiResp.Status,
			Stack:   apiResp.Stack,
		}
		if errorResp.Status == 0 {
			errorResp.Status = resp.StatusCode
		}
		// If message is empty, try to provide a default based on status code
		// But also check if maybe the message is in a different format
		if errorResp.Message == "" {
			// Try to extract message from raw response if it exists in a different format
			var rawResp map[string]interface{}
			if json.Unmarshal(bodyBytes, &rawResp) == nil {
				if msg, ok := rawResp["message"].(string); ok && msg != "" {
					errorResp.Message = msg
				}
			}
			// If still empty, use default based on status code
			if errorResp.Message == "" {
				switch errorResp.Status {
				case 401, 403:
					errorResp.Message = "Unauthorized"
				case 404:
					errorResp.Message = "Not found"
				case 500:
					errorResp.Message = "Internal server error"
				default:
					errorResp.Message = "An error occurred"
				}
			}
		}
		return nil, "", &errorResp
	}

	// Parse successful response data
	if apiResp.Data != nil {
		if err := json.Unmarshal(apiResp.Data, &response); err != nil {
			return nil, "", fmt.Errorf("failed to unmarshal response data: %w", err)
		}
	}

	return &response, sessionToken, nil
}

// Logout performs a logout request
func (c *Client) Logout() error {
	var result interface{}
	err := c.Post("/auth/logout", nil, &result)
	if err != nil {
		return err
	}
	return nil
}
