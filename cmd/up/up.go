package up

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var UpCmd = &cobra.Command{
	Use:   "up",
	Short: "Start a client or site",
	Long:  "Bring up a client or site tunneled connection",
	Run: func(cmd *cobra.Command, args []string) {
		// Default to client subcommand if no subcommand is provided
		// This makes "pangolin up" equivalent to "pangolin up client"
		cmd.Flags().VisitAll(func(flag *pflag.Flag) {
			if cmd.Flags().Changed(flag.Name) {
				ClientCmd.Flags().Set(flag.Name, flag.Value.String())
			}
		})
		ClientCmd.Run(ClientCmd, args)
	},
}

func init() {
	addClientFlags(UpCmd)
}
