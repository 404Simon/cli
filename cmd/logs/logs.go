package logs

import (
	"github.com/spf13/cobra"
)

func LogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "View client logs",
		Long:  "View and follow client logs",
	}

	cmd.AddCommand(ClientLogsCmd())

	return cmd
}
