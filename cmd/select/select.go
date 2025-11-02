package selectcmd

import (
	"github.com/spf13/cobra"
)

var SelectCmd = &cobra.Command{
	Use:   "select",
	Short: "Select resources",
	Long:  "Select resources such as organizations",
}
