package utils

import (
	"fmt"
	"os"

	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/config"
)

// GetDeviceName returns a human-readable device name
func GetDeviceName() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "Unknown Device"
	}
	return hostname
}

// EnsureOlmCredentials ensures that OLM credentials exist and are valid.
// It checks if OLM credentials exist locally, verifies them on the server,
// and creates new ones if they don't exist or are invalid.
//
// If new ones are created, a "true" is returned to indicate we need to
// save the new credentials to disk.
func EnsureOlmCredentials(client *api.Client, account *config.Account) (bool, error) {
	userID := account.UserID

	if account.OlmCredentials != nil {
		serverCreds, err := client.GetUserOlm(userID, account.OlmCredentials.ID)
		if err == nil && serverCreds != nil {
			return false, nil
		}

		// If getting OLM fails, the OLM might not exist.
		// This requires regeneration; in case of any errors
		// that are not API-related, these are likely not
		// related to the credentials and should be bubbled up.
		if _, ok := err.(*api.ErrorResponse); !ok {
			return false, fmt.Errorf("failed to get OLM: %w", err)
		}

		// Clear invalid credentials so we can try to create new ones
		account.OlmCredentials = nil
	}

	newOlm, err := client.CreateOlm(userID, GetDeviceName())
	if err != nil {
		return false, fmt.Errorf("failed to create OLM: %w", err)
	}

	account.OlmCredentials = &config.OlmCredentials{
		ID:     newOlm.OlmID,
		Secret: newOlm.Secret,
	}

	return true, nil
}

// EnsureOrgAccess ensures that the user has access to the organization
func EnsureOrgAccess(client *api.Client, account *config.Account) error {
	// Get org via API to ensure it exists
	_, err := client.GetOrg(account.OrgID)
	if err != nil {
		return err
	}

	// Check org user access and policies
	accessResponse, err := client.CheckOrgUserAccess(account.OrgID, account.UserID)
	if err != nil {
		return err
	}

	// Check if user is allowed access
	if !accessResponse.Allowed {
		// Get hostname base URL for constructing the web URL
		url := fmt.Sprintf("%s/%s", FormatHostnameBaseURL(account.Host), account.OrgID)
		return fmt.Errorf("Organization policy is preventing you from connecting. Please visit %s to complete required steps", url)
	}

	return nil
}
