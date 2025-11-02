package selectcmd

import (
	"github.com/fosrl/cli/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var orgCmd = &cobra.Command{
	Use:   "org",
	Short: "Select an organization",
	Long:  "List your organizations and select one to use",
	Run: func(cmd *cobra.Command, args []string) {
		// Check if user is logged in
		if err := utils.EnsureLoggedIn(); err != nil {
			utils.Error("%v", err)
			return
		}

		// Get userId from config
		userID := viper.GetString("userId")

		// Select organization
		orgID, err := utils.SelectOrg(userID)
		if err != nil {
			utils.Error("%v", err)
			return
		}

		utils.Success("Successfully selected organization: %s", orgID)
	},
}

func init() {
	SelectCmd.AddCommand(orgCmd)
}
