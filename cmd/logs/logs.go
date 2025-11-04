package logs

import (
	"github.com/spf13/cobra"
)

var LogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View client logs",
	Long:  "View and follow client logs",
}

