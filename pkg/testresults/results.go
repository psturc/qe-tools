package testresults

import (
	"encoding/xml"
	"fmt"

	"github.com/bsm/ginkgo/v2/reporters"
	"github.com/konflux-ci/qe-tools/pkg/oci"
	"k8s.io/klog/v2"
)

// FailureType collects the types of failures met in the PipelineRun
type FailureType string

const (
	dropdownSummaryString = "Click to view logs"

	// OtherFailure represents the type failure that hasn't been identified by the analyzer
	OtherFailure FailureType = "otherFailure"
	// ClusterCreationFailure represents the issue with provisioning a test cluster
	ClusterCreationFailure FailureType = "clusterCreationFailure"
	// TestRunFailure represents the issue with running the test command
	TestRunFailure FailureType = "testRunFailure"
	// TestCaseFailure represents the issue in the test suite (i.e. test case has failed)
	TestCaseFailure FailureType = "testCaseFailure"
)

// FailedTestCasesReport collects the data about failures
type FailedTestCasesReport struct {
	JUnitTestSuites *reporters.JUnitTestSuites

	HeaderString        string
	FailedTestCaseNames []string

	ClusterProvisionLog string
	E2ETestLog          string

	FailureType FailureType
}

// CollectTestFilesData inspects the FilesPathMap data and based on the supplied
// names for JUnit, E2E run log and cluster provision log file names determines
// the cause of the PipelineRun failure
func (f *FailedTestCasesReport) CollectTestFilesData(fpm oci.FilesPathMap, junitFilename, e2eTestRunLogFilename, clusterProvisionLogFilename string) {
	f.JUnitTestSuites = &reporters.JUnitTestSuites{}

	for _, file := range fpm {
		if string(file.Filename) == junitFilename {
			if err := xml.Unmarshal([]byte(file.Content), f.JUnitTestSuites); err != nil {
				klog.Warningf("cannot decode JUnit suite into xml: %+v", err)
			}
			f.FailureType = TestCaseFailure
			klog.Info("the given PipelineRun failed on a test case failure")
			return
		}
	}

	for _, file := range fpm {
		if file.Filename == e2eTestRunLogFilename {
			f.E2ETestLog = file.Content
		}
		if file.Filename == clusterProvisionLogFilename {
			f.ClusterProvisionLog = file.Content
		}
	}

	if f.E2ETestLog != "" {
		klog.Info("no JUnit file found - PipelineRun failed during running tests")
		f.FailureType = TestRunFailure
		return
	}

	if f.ClusterProvisionLog != "" {
		klog.Info("failed to provision a cluster")
		f.FailureType = ClusterCreationFailure
		return
	}

	klog.Info("could not find any related artifacts")
	f.FailureType = OtherFailure
}

// setHeaderString initialises sets 'headerString' field for the report summary
// based on phase at which PipelineRun failed
func (f *FailedTestCasesReport) setHeaderString() {
	switch f.FailureType {
	case OtherFailure:
		f.HeaderString = ":rotating_light: **Couldn't detect a specific failure, see the related PipelineRun for more details or consult with Konflux DevProd team.**\n"
		return
	case TestRunFailure:
		f.HeaderString = ":rotating_light: **No JUnit file found, see the log from running tests**: \n"
		return
	case ClusterCreationFailure:
		f.HeaderString = ":rotating_light: **Failed to provision a cluster, see the log for more details**: \n"
		return
	case TestCaseFailure:
		f.HeaderString = ":rotating_light: **Error occurred while running the E2E tests, list of failed Spec(s)**: \n"
	}
}

// extractFailedTestCasesBody initialises the FailedTestCasesReport struct's
// 'failedTestCaseNames' field with the names of failed test cases
// within given JUnitTestSuites -- if the given JUnitTestSuites is !nil.
func (f *FailedTestCasesReport) extractFailedTestCasesBody() {
	switch f.FailureType {
	case OtherFailure:
		return
	case ClusterCreationFailure:
		testCaseEntry := returnContentWrappedInDropdown(dropdownSummaryString, f.ClusterProvisionLog)
		f.FailedTestCaseNames = append(f.FailedTestCaseNames, testCaseEntry)
		return
	case TestRunFailure:
		testCaseEntry := returnContentWrappedInDropdown(dropdownSummaryString, f.E2ETestLog)
		f.FailedTestCaseNames = append(f.FailedTestCaseNames, testCaseEntry)
		return
	}
	ftc := f.GetFailedTestCases()
	for _, tc := range ftc {
		var tcMessage string
		switch {
		case tc.Status == "timedout":
			tcMessage = returnContentWrappedInDropdown(dropdownSummaryString, tc.SystemErr)
		case tc.Failure != nil:
			tcMessage = "```\n" + tc.Failure.Message + "\n```"
		default:
			tcMessage = "```\n" + tc.Error.Message + "\n```"
		}

		testCaseEntry := "* :arrow_right: " + "[**`" + tc.Status + "`**] " + tc.Name + "\n" + tcMessage
		f.FailedTestCaseNames = append(f.FailedTestCaseNames, testCaseEntry)
	}
}

// GetFormattedReport returns the full report (test run analysis) as a string
func (f *FailedTestCasesReport) GetFormattedReport() (report string) {
	f.setHeaderString()
	f.extractFailedTestCasesBody()

	report = f.HeaderString
	for _, failedTCName := range f.FailedTestCaseNames {
		report += fmt.Sprintf("\n %s\n", failedTCName)
	}

	return
}

// GetFailedTestCases returns the list of JUnit test cases that failed
func (f *FailedTestCasesReport) GetFailedTestCases() (ftc []reporters.JUnitTestCase) {
	for _, testSuite := range f.JUnitTestSuites.TestSuites {
		if testSuite.Failures > 0 || testSuite.Errors > 0 {
			for _, tc := range testSuite.TestCases {
				if tc.Failure != nil || tc.Error != nil {
					klog.Infof("Found a Test Case (suiteName/testCaseName): %s/%s, that didn't pass", testSuite.Name, tc.Name)
					ftc = append(ftc, tc)
				}
			}
		}
	}
	return
}

func returnContentWrappedInDropdown(summary, content string) string {
	return "<details><summary>" + summary + "</summary><br><pre>" + content + "</pre></details>"
}
