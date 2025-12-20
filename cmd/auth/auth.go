package auth

import (
	"github.com/fosrl/cli/cmd/auth/login"
	"github.com/fosrl/cli/cmd/auth/logout"
	"github.com/fosrl/cli/cmd/auth/status"
	"github.com/spf13/cobra"
)

func AuthCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication commands",
		Long:  "Manage authentication and sessions",
	}

	cmd.AddCommand(login.LoginCmd())
	cmd.AddCommand(logout.LogoutCmd())
	cmd.AddCommand(status.StatusCmd())

	return cmd
}
