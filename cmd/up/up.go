package up

import (
	"github.com/spf13/cobra"
)

var UpCmd = &cobra.Command{
	Use:   "up",
	Short: "Start a client or site",
	Long:  "Bring up a client or site tunneled connection",
}
