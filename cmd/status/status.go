package status

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var StatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Status commands",
	Long:  "View status information.",
	Run: func(cmd *cobra.Command, args []string) {
		// Default to client subcommand if no subcommand is provided
		// This makes "pangolin status" equivalent to "pangolin status client"
		cmd.Flags().VisitAll(func(flag *pflag.Flag) {
			if cmd.Flags().Changed(flag.Name) {
				ClientCmd.Flags().Set(flag.Name, flag.Value.String())
			}
		})
		ClientCmd.Run(ClientCmd, args)
	},
}

func init() {
	addStatusClientFlags(StatusCmd)
	StatusCmd.AddCommand(ClientCmd)
}
