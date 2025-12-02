package utils

import (
	"errors"
	"fmt"
	"os"
	"os/user"

	"github.com/fosrl/cli/internal/api"
	"github.com/spf13/viper"
)

// GetOriginalUserHomeDir returns the home directory of the original user
// (the user who invoked the command, not the effective user when running with sudo).
// This ensures that config files and keyring access work both with and without sudo.
func GetOriginalUserHomeDir() (string, error) {
	// Check if we're running under sudo - SUDO_USER contains the original user
	sudoUser := os.Getenv("SUDO_USER")
	if sudoUser != "" {
		// We're running with sudo, get the original user's home directory
		u, err := user.Lookup(sudoUser)
		if err != nil {
			return "", fmt.Errorf("failed to lookup original user %s: %w", sudoUser, err)
		}
		return u.HomeDir, nil
	}

	// Not running with sudo, use current user's home directory
	return os.UserHomeDir()
}

// EnsureLoggedIn checks if the user is logged in by verifying:
// 1. A userId exists in the viper config
// 2. A session token exists in the key store
// Returns an error if the user is not logged in, nil otherwise.
func EnsureLoggedIn() error {
	// Check for userId in config
	userID := viper.GetString("userId")
	if userID == "" {
		return errors.New("Please login first")
	}

	// Check for session token in keyring
	_, err := api.GetSessionToken()
	if err != nil {
		return fmt.Errorf("Please login first")
	}

	return nil
}
