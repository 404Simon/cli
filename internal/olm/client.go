package olm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/spf13/viper"
)

const (
	defaultSocketPath = "/var/run/olm.sock"
)

// Client handles communication with the OLM process via Unix socket
type Client struct {
	socketPath string
	httpClient *http.Client
}

// StatusResponse represents the status response from OLM
type StatusResponse struct {
	Status    string             `json:"status"`
	Connected bool               `json:"connected"`
	TunnelIP  string             `json:"tunnelIP"`
	Version   string             `json:"version"`
	Peers     map[string]Peer    `json:"peers"`
	Registered bool               `json:"registered"` // whether the wireguard interface is created
}

// Peer represents a peer in the status response
type Peer struct {
	SiteID    int    `json:"siteId"`
	Connected bool   `json:"connected"`
	RTT       int64  `json:"rtt"` // nanoseconds
	LastSeen  string `json:"lastSeen"`
	Endpoint  string `json:"endpoint"`
	IsRelay   bool   `json:"isRelay"`
}

// ExitResponse represents the exit/shutdown response
type ExitResponse struct {
	Status string `json:"status"`
}

// NewClient creates a new OLM socket client
func NewClient(socketPath string) *Client {
	if socketPath == "" {
		socketPath = getDefaultSocketPath()
	}

	return &Client{
		socketPath: socketPath,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return net.Dial("unix", socketPath)
				},
			},
		},
	}
}

// getDefaultSocketPath returns the default socket path
// Checks config first, then falls back to default
func getDefaultSocketPath() string {
	if socketPath := viper.GetString("olm_defaults.socket_path"); socketPath != "" {
		return socketPath
	}
	return defaultSocketPath
}

// GetDefaultSocketPath returns the default socket path (exported for use in other packages)
func GetDefaultSocketPath() string {
	return getDefaultSocketPath()
}

// GetStatus retrieves the current status from the OLM process
func (c *Client) GetStatus() (*StatusResponse, error) {
	req, err := http.NewRequest("GET", "http://localhost/status", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Check if socket file exists
		if _, statErr := os.Stat(c.socketPath); os.IsNotExist(statErr) {
			return nil, fmt.Errorf("socket does not exist: %s (is the client running?)", c.socketPath)
		}
		return nil, fmt.Errorf("failed to connect to socket: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var status StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &status, nil
}

// Exit sends a shutdown signal to the OLM process
func (c *Client) Exit() (*ExitResponse, error) {
	req, err := http.NewRequest("POST", "http://localhost/exit", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Check if socket file exists
		if _, statErr := os.Stat(c.socketPath); os.IsNotExist(statErr) {
			return nil, fmt.Errorf("socket does not exist: %s (is the client running?)", c.socketPath)
		}
		return nil, fmt.Errorf("failed to connect to socket: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var exitResp ExitResponse
	if err := json.NewDecoder(resp.Body).Decode(&exitResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &exitResp, nil
}

// IsRunning checks if the OLM process is running by checking if the socket exists
func (c *Client) IsRunning() bool {
	_, err := os.Stat(c.socketPath)
	return err == nil
}

