package analyzetestresults

import (
	"fmt"
	"os"

	"github.com/konflux-ci/qe-tools/pkg/oci"
	"github.com/konflux-ci/qe-tools/pkg/testresults"
	"k8s.io/klog/v2"

	"github.com/konflux-ci/qe-tools/pkg/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	ociArtifactRef              string
	clusterProvisionLogFilename string
	jUnitFilename               string
	e2eTestRunLogFilename       string
	outputFilename              string
)

// AnalyzeTestResultsCmd represents the analyze-test-results command
var AnalyzeTestResultsCmd = &cobra.Command{
	Use:   "analyze-test-results",
	Short: "Command for analyzing test results",
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		if viper.GetString(types.OciArtifactRefParamName) == "" {
			_ = cmd.Usage()
			return fmt.Errorf("parameter %q not provided, neither %s env var was set", types.OciArtifactRefParamName, types.OciArtifactRefEnv)
		}
		return nil
	},
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ociArtifactRef = viper.GetString(types.OciArtifactRefParamName)

		cfg := oci.ScannerConfig{
			OciArtifactReference: ociArtifactRef,
			FileNameFilter:       []string{jUnitFilename, clusterProvisionLogFilename, e2eTestRunLogFilename},
		}

		scanner, err := oci.NewArtifactScanner(cfg)
		if err != nil {
			return fmt.Errorf("failed to initialize artifact scanner: %+v", err)
		}

		if err := scanner.Run(); err != nil {
			return fmt.Errorf("failed to scan artifact from %s: %+v", ociArtifactRef, err)
		}

		failedTCReport := testresults.FailedTestCasesReport{}
		failedTCReport.CollectTestFilesData(scanner.FilesPathMap, jUnitFilename, e2eTestRunLogFilename, clusterProvisionLogFilename)

		if err := os.WriteFile(outputFilename, []byte(failedTCReport.GetFormattedReport()), 0o600); err != nil {
			return fmt.Errorf("failed to create a file with the test result analysis: %+v", err)
		}
		klog.Infof("analysis saved to %s", outputFilename)

		return nil
	},
}

func init() {
	AnalyzeTestResultsCmd.Flags().StringVar(&ociArtifactRef, types.OciArtifactRefParamName, "", "OCI artifact reference (e.g. \"quay.io/org/repo:oci-artifact-tag\")")
	AnalyzeTestResultsCmd.Flags().StringVar(&jUnitFilename, types.JUnitFilenameParamName, "e2e-report.xml", "A name of the file containing JUnit report")
	AnalyzeTestResultsCmd.Flags().StringVar(&clusterProvisionLogFilename, types.ClusterProvisionLogFileParamName, "cluster-provision.log", "A name of the file containing log from provisioning a testing cluster")
	AnalyzeTestResultsCmd.Flags().StringVar(&e2eTestRunLogFilename, types.E2ETestRunLogFileParamName, "e2e-tests.log", "A name of the file containing log from running tests")
	AnalyzeTestResultsCmd.Flags().StringVar(&outputFilename, types.OutputFilenameParamName, "analysis.md", "A name of the file to store the analysis output in")

	_ = viper.BindPFlag(types.OciArtifactRefParamName, AnalyzeTestResultsCmd.Flags().Lookup(types.OciArtifactRefParamName))
	_ = viper.BindEnv(types.OciArtifactRefParamName, types.OciArtifactRefEnv)
}
