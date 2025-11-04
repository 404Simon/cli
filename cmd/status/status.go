package status

import (
	"github.com/fosrl/cli/cmd/status/client"
	"github.com/spf13/cobra"
)

var StatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Status commands",
	Long:  "View status information",
}

func init() {
	StatusCmd.AddCommand(client.ClientCmd)
}

