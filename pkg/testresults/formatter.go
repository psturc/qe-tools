package testresults

import (
	"fmt"
)

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

func returnContentWrappedInDropdown(summary, content string) string {
	return "<details><summary>" + summary + "</summary><br><pre>" + content + "</pre></details>"
}
