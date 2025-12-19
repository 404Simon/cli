package status

import (
	"fmt"
	"os"

	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/cli/internal/logger"
	"github.com/spf13/cobra"
)

func StatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check authentication status",
		Long:  "Check if you are logged in and view your account information",
		Run: func(cmd *cobra.Command, args []string) {
			if err := statusMain(cmd); err != nil {
				os.Exit(1)
			}
		},
	}

	return cmd
}

func statusMain(cmd *cobra.Command) error {
	apiClient := api.FromContext(cmd.Context())
	accountStore := config.AccountStoreFromContext(cmd.Context())

	account, err := accountStore.ActiveAccount()
	if err != nil {
		logger.Info("Status: %s", err)
		logger.Info("Run 'pangolin login' to authenticate")
		return err
	}

	// User info exists in config, try to get user from API
	user, err := apiClient.GetUser()
	if err != nil {
		// Unable to get user - consider logged out (previously logged in but now not)
		logger.Info("Status: logged out: %v", err)
		logger.Info("Your session has expired or is invalid")
		logger.Info("Run 'pangolin login' to authenticate again")
		return err
	}

	// Successfully got user - logged in
	logger.Success("Status: logged in")
	// Show hostname if available
	logger.Info("@ %s", account.Host)
	fmt.Println()

	// Display user information
	displayName := user.Email
	if user.Username != nil && *user.Username != "" {
		displayName = *user.Username
	} else if user.Name != nil && *user.Name != "" {
		displayName = *user.Name
	}
	if displayName != "" {
		logger.Info("User: %s", displayName)
	}
	if user.UserID != "" {
		logger.Info("User ID: %s", user.UserID)
	}

	// Display organization information
	logger.Info("Org ID: %s", account.OrgID)

	return nil
}
