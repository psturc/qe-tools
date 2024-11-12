package testresults

import (
	"encoding/xml"

	"github.com/bsm/ginkgo/v2/reporters"
	"github.com/konflux-ci/qe-tools/pkg/oci"
	"k8s.io/klog/v2"
)

// FailureType collects the types of failures met in the PipelineRun
type FailureType string

const (
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
