package down

import (
	"github.com/spf13/cobra"
)

var DownCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop a client or site",
	Long:  "Stop a client or site tunneled connection",
}

