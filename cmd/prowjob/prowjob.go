package prowjob

import (
	"github.com/spf13/cobra"
)

const (
	artifactDirEnv       string = "ARTIFACT_DIR"
	artifactDirParamName string = "artifact-dir"

	failIfUnhealthyParamName string = "fail-if-unhealthy"

	prowJobIDEnv       string = "PROW_JOB_ID"
	prowJobIDParamName string = "prow-job-id"
)

var (
	artifactDir     string
	failIfUnhealthy bool
	prowJobID       string
)

// ProwjobCmd represents the prowjob command
var ProwjobCmd = &cobra.Command{
	Use:   "prowjob",
	Short: "Commands for processing Prow jobs",
}

func init() {
	ProwjobCmd.AddCommand(periodicSlackReportCmd)
	ProwjobCmd.AddCommand(createReportCmd)
	ProwjobCmd.AddCommand(healthCheckCmd)

	createReportCmd.Flags().StringVar(&artifactDir, artifactDirParamName, "", "Path to the folder where to store produced files")
	healthCheckCmd.Flags().StringVar(&artifactDir, artifactDirParamName, "", "Path to the folder where to store produced files")
}
