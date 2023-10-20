package prowjob

import (
	"github.com/spf13/cobra"
)

// ProwjobCmd represents the prowjob command
var ProwjobCmd = &cobra.Command{
	Use:   "prowjob",
	Short: "Commands for processing Prow jobs",
}

func init() {
	ProwjobCmd.AddCommand(analyzeCmd)
	ProwjobCmd.AddCommand(periodicSlackReportCmd)
	ProwjobCmd.AddCommand(reportCmd)
}
