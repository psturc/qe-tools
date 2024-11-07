package analyzetestresults

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"

	"github.com/bsm/ginkgo/v2/reporters"
	"github.com/konflux-ci/qe-tools/pkg/oci"
	"k8s.io/klog/v2"

	"github.com/konflux-ci/qe-tools/pkg/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// FailureType collects the types of failures met in the PipelineRun
type FailureType string

const (
	dropdownSummaryString = "Click to view logs"

	// OtherFailure represents the type failure that hasn't been identified by the analyzer
	OtherFailure FailureType = "otherFailure"
	// ClusterCreationFailure represents the issue with provisioning a test cluster
	ClusterCreationFailure FailureType = "clusterCreationFailure"
	// TestRunFailure represents the issue with provisioning a test cluster
	TestRunFailure FailureType = "testRunFailure"
	// TestCaseFailure represents the issue with provisioning a test cluster
	TestCaseFailure FailureType = "testCaseFailure"
)

var (
	ociArtifactRef              string
	clusterProvisionLogFilename string
	jUnitFilename               string
	e2eTestRunLogFilename       string
	outputFilename              string
)

// FailedTestCasesReport collects the data about failures
type FailedTestCasesReport struct {
	headerString        string
	failedTestCaseNames []string

	clusterProvisionLog string
	e2eTestLog          string

	failureType FailureType
}

// getTestSuitesFromXMLFile returns all the JUnitTestSuites
// present within a file with the given name
func getTestSuitesFromXMLFile(scanner *oci.ArtifactScanner, filename string) (*reporters.JUnitTestSuites, error) {
	overallJUnitSuites := &reporters.JUnitTestSuites{}

	for _, file := range scanner.FilesPathMap {
		if string(file.Filename) == filename {
			if err := xml.Unmarshal([]byte(file.Content), overallJUnitSuites); err != nil {
				klog.Warningf("cannot decode JUnit suite into xml: %+v", err)
				return &reporters.JUnitTestSuites{}, err
			}
			return overallJUnitSuites, nil
		}
	}

	return &reporters.JUnitTestSuites{}, fmt.Errorf("couldn't find the %s file", filename)
}

// setHeaderString initialises struct FailedTestCasesReport's
// 'headerString' field based on phase at which PipelineRun failed
func (f *FailedTestCasesReport) setHeaderString(overallJUnitSuites *reporters.JUnitTestSuites) {
	if len(overallJUnitSuites.TestSuites) == 0 {
		if f.clusterProvisionLog == "" && f.e2eTestLog == "" {
			klog.Info("could not find any related artifacts")
			f.failureType = OtherFailure
			f.headerString = ":rotating_light: **Couldn't detect a specific failure, see the related PipelineRun for more details or consult with Konflux DevProd team.**\n"
			return
		}
		if f.e2eTestLog != "" {
			klog.Info("no JUnit file found - PipelineRun failed during running tests")
			f.failureType = TestRunFailure
			f.headerString = ":rotating_light: **No JUnit file found, see the log from running tests**: \n"
			return
		}
		if f.clusterProvisionLog != "" {
			klog.Info("failed to provision a cluster")
			f.failureType = ClusterCreationFailure
			f.headerString = ":rotating_light: **Failed to provision a cluster, see the log for more details**: \n"
			return
		}

	} else {
		klog.Info("The given PipelineRun failed while running tests")
		f.failureType = TestCaseFailure
		f.headerString = ":rotating_light: **Error occurred while running the E2E tests, list of failed Spec(s)**: \n"
	}
}

// extractFailedTestCases initialises the FailedTestCasesReport struct's
// 'failedTestCaseNames' field with the names of failed test cases
// within given JUnitTestSuites -- if the given JUnitTestSuites is !nil.
func (f *FailedTestCasesReport) extractFailedTestCases(overallJUnitSuites *reporters.JUnitTestSuites) {
	switch f.failureType {
	case OtherFailure:
		return
	case ClusterCreationFailure:
		testCaseEntry := returnContentWrappedInDropdown(dropdownSummaryString, f.clusterProvisionLog)
		f.failedTestCaseNames = append(f.failedTestCaseNames, testCaseEntry)
		return
	case TestRunFailure:
		testCaseEntry := returnContentWrappedInDropdown(dropdownSummaryString, f.e2eTestLog)
		f.failedTestCaseNames = append(f.failedTestCaseNames, testCaseEntry)
		return
	}
	for _, testSuite := range overallJUnitSuites.TestSuites {
		if testSuite.Failures > 0 || testSuite.Errors > 0 {
			var tcMessage string
			for _, tc := range testSuite.TestCases {
				if tc.Failure != nil || tc.Error != nil {
					klog.Infof("Found a Test Case (suiteName/testCaseName): %s/%s, that didn't pass", testSuite.Name, tc.Name)
					switch {
					case tc.Status == "timedout":
						tcMessage = returnContentWrappedInDropdown(dropdownSummaryString, tc.SystemErr)
					case tc.Failure != nil:
						tcMessage = "```\n" + tc.Failure.Message + "\n```"
					default:
						tcMessage = "```\n" + tc.Error.Message + "\n```"
					}

					testCaseEntry := "* :arrow_right: " + "[**`" + tc.Status + "`**] " + tc.Name + "\n" + tcMessage
					f.failedTestCaseNames = append(f.failedTestCaseNames, testCaseEntry)
				}
			}
		}
	}
}

// getFormattedReport returns the full report as a string
func (f *FailedTestCasesReport) getFormattedReport() string {
	msg := f.headerString
	for _, failedTCName := range f.failedTestCaseNames {
		msg += fmt.Sprintf("\n %s\n", failedTCName)
	}

	return msg
}

func returnContentWrappedInDropdown(summary, content string) string {
	return "<details><summary>" + summary + "</summary><br><pre>" + content + "</pre></details>"
}

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
			FileNameFilter:       []string{types.JunitFilename, clusterProvisionLogFilename, e2eTestRunLogFilename},
		}

		scanner, err := oci.NewArtifactScanner(cfg)
		if err != nil {
			return fmt.Errorf("failed to initialize artifact scanner: %+v", err)
		}

		if err := scanner.Run(); err != nil {
			return fmt.Errorf("failed to scan artifact from %s: %+v", ociArtifactRef, err)
		}

		overallJUnitSuites, err := getTestSuitesFromXMLFile(scanner, jUnitFilename)
		// make sure that the Prow job didn't fail while creating the cluster
		if err != nil && !strings.Contains(err.Error(), fmt.Sprintf("couldn't find the %s file", jUnitFilename)) {
			return fmt.Errorf("failed to get JUnitTestSuites from the file %s: %+v", jUnitFilename, err)
		}

		failedTCReport := FailedTestCasesReport{}

		for _, file := range scanner.FilesPathMap {
			if file.Filename == clusterProvisionLogFilename {
				failedTCReport.clusterProvisionLog = file.Content
			}
			if file.Filename == e2eTestRunLogFilename {
				failedTCReport.e2eTestLog = file.Content
			}
		}

		failedTCReport.setHeaderString(overallJUnitSuites)
		failedTCReport.extractFailedTestCases(overallJUnitSuites)

		if err := os.WriteFile(outputFilename, []byte(failedTCReport.getFormattedReport()), 0o600); err != nil {
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
