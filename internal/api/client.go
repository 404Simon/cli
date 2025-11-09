package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ClientConfig holds configuration for creating a new client
type ClientConfig struct {
	BaseURL           string
	AgentName         string
	Token             string
	APIKey            string
	SessionCookieName string
	CSRFToken         string
}

// NewClient creates a new API client with the provided configuration
func NewClient(config ClientConfig) (*Client, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://app.pangolin.net"
	} else if !strings.HasPrefix(baseURL, "http") {
		baseURL = "https://" + baseURL
	}

	// Default session cookie name
	sessionCookieName := config.SessionCookieName
	if sessionCookieName == "" {
		sessionCookieName = "p_session_token"
	}

	client := &Client{
		BaseURL:           strings.TrimSuffix(baseURL, "/"),
		AgentName:         config.AgentName,
		APIKey:            config.APIKey,
		Token:             config.Token,
		SessionCookieName: sessionCookieName,
		CSRFToken:         config.CSRFToken,
		HTTPClient: &HTTPClient{
			Timeout: 30 * time.Second,
		},
	}

	return client, nil
}

// Get performs a GET request to the API
func (c *Client) Get(endpoint string, result interface{}, opts ...RequestOptions) error {
	return c.request(http.MethodGet, endpoint, nil, result, opts...)
}

// Post performs a POST request to the API
func (c *Client) Post(endpoint string, payload interface{}, result interface{}, opts ...RequestOptions) error {
	return c.request(http.MethodPost, endpoint, payload, result, opts...)
}

// Put performs a PUT request to the API
func (c *Client) Put(endpoint string, payload interface{}, result interface{}, opts ...RequestOptions) error {
	return c.request(http.MethodPut, endpoint, payload, result, opts...)
}

// Patch performs a PATCH request to the API
func (c *Client) Patch(endpoint string, payload interface{}, result interface{}, opts ...RequestOptions) error {
	return c.request(http.MethodPatch, endpoint, payload, result, opts...)
}

// Delete performs a DELETE request to the API
func (c *Client) Delete(endpoint string, result interface{}, opts ...RequestOptions) error {
	return c.request(http.MethodDelete, endpoint, nil, result, opts...)
}

// request is the core method that handles all HTTP requests
func (c *Client) request(method, endpoint string, payload interface{}, result interface{}, opts ...RequestOptions) error {
	// Build URL
	requestURL, err := c.buildURL(endpoint, opts...)
	if err != nil {
		return fmt.Errorf("failed to build URL: %w", err)
	}

	// Prepare request body
	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	// Create HTTP request
	req, err := http.NewRequest(method, requestURL, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Set User-Agent with agent name
	userAgent := c.AgentName
	if userAgent == "" {
		userAgent = "pangolin-cli"
	}
	req.Header.Set("User-Agent", userAgent)

	// Set CSRF header if provided
	if c.CSRFToken != "" {
		req.Header.Set("X-CSRF-Token", c.CSRFToken)
	}

	// Set authentication
	if c.Token != "" {
		// Token is sent as a cookie
		cookie := &http.Cookie{
			Name:  c.SessionCookieName,
			Value: c.Token,
		}
		req.AddCookie(cookie)
	} else if c.APIKey != "" {
		// API key is sent as Bearer token
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	// Apply custom headers from options
	if len(opts) > 0 && opts[0].Headers != nil {
		for key, value := range opts[0].Headers {
			req.Header.Set(key, value)
		}
	}

	// Create HTTP client and execute request
	httpClient := &http.Client{
		Timeout: c.HTTPClient.Timeout,
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if len(bodyBytes) == 0 {
		return nil
	}

	// Parse the API response structure
	var apiResp APIResponse
	if err := json.Unmarshal(bodyBytes, &apiResp); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check if the response indicates an error (based on error/success fields)
	if apiResp.Error.Bool() || !apiResp.Success {
		errorResp := ErrorResponse{
			Message: apiResp.Message,
			Status:  apiResp.Status,
			Stack:   apiResp.Stack,
		}
		// Use HTTP status code if status field is not set
		if errorResp.Status == 0 {
			errorResp.Status = resp.StatusCode
		}
		// If message is empty, try to provide a default based on status code
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
		return &errorResp
	}

	// Parse successful response
	if result != nil && apiResp.Data != nil {
		if err := json.Unmarshal(apiResp.Data, result); err != nil {
			return fmt.Errorf("failed to unmarshal response data: %w", err)
		}
	}

	return nil
}

// buildURL constructs the full URL for the request
func (c *Client) buildURL(endpoint string, opts ...RequestOptions) (string, error) {
	// Ensure endpoint starts with /
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}

	baseURL := strings.TrimSuffix(c.BaseURL, "/")
	fullURL := baseURL + endpoint

	// Add query parameters if provided
	if len(opts) > 0 && opts[0].Query != nil && len(opts[0].Query) > 0 {
		u, err := url.Parse(fullURL)
		if err != nil {
			return "", err
		}

		q := u.Query()
		for key, value := range opts[0].Query {
			q.Set(key, value)
		}
		u.RawQuery = q.Encode()
		fullURL = u.String()
	}

	return fullURL, nil
}

// SetBaseURL updates the base URL for the client
func (c *Client) SetBaseURL(baseURL string) {
	if !strings.HasPrefix(baseURL, "http") {
		baseURL = "https://" + baseURL
	}
	c.BaseURL = strings.TrimSuffix(baseURL, "/")
}

// SetToken updates the token for the client
func (c *Client) SetToken(token string) {
	c.Token = token
}
