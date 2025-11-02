package api

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
	"github.com/zalando/go-keyring"
)

const (
	keyringService = "pangolin-cli"
	keyringUser    = "session-token"
)

var GlobalClient *Client

// InitGlobalClient initializes the global API client with stored credentials.
// This function always succeeds in creating a client, even if no token is available.
// The client will be created without authentication if no token is found.
func InitGlobalClient() error {
	// Get hostname from viper config
	hostname := viper.GetString("hostname")
	if hostname == "" {
		hostname = "app.pangolin.net"
	}

	// Get session token from keyring (ignore errors - just use empty token if not found)
	token, _ := keyring.Get(keyringService, keyringUser)

	// Build base URL (hostname should already include protocol from login)
	baseURL := hostname
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		// If no protocol, default to https
		baseURL = "https://" + baseURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/") + "/api/v1"

	// Create API client (this should never fail, but handle it just in case)
	client, err := NewClient(ClientConfig{
		BaseURL:           baseURL,
		AgentName:         "pangolin-cli",
		Token:             token,
		SessionCookieName: "p_session_token",
		CSRFToken:         "x-csrf-protection",
	})
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	GlobalClient = client
	return nil
}

// SaveSessionToken saves the session token to the OS keyring
func SaveSessionToken(token string) error {
	return keyring.Set(keyringService, keyringUser, token)
}

// GetSessionToken retrieves the session token from the OS keyring
func GetSessionToken() (string, error) {
	return keyring.Get(keyringService, keyringUser)
}

// DeleteSessionToken deletes the session token from the OS keyring
func DeleteSessionToken() error {
	return keyring.Delete(keyringService, keyringUser)
}
