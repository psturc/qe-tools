package testresults

import (
	"fmt"
)

const dropdownSummaryString = "Click to view logs"

// extractFailedTestCasesBody initialises the FailedTestCasesReport struct's
// 'failedTestCaseNames' field with the names of failed test cases
// within given JUnitTestSuites -- if the given JUnitTestSuites is !nil.
func extractFailedTestCasesBody(f FailedTestCasesReport) (failedTestCasesBody []string) {
	switch f.FailureType {
	case OtherFailure:
		return
	case ClusterCreationFailure:
		return []string{returnContentWrappedInDropdown(dropdownSummaryString, f.ClusterProvisionLog)}
	case TestRunFailure:
		return []string{returnContentWrappedInDropdown(dropdownSummaryString, f.E2ETestLog)}
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
		failedTestCasesBody = append(failedTestCasesBody, testCaseEntry)
	}
	return
}

// getHeaderStringForFailureType returns 'headerString' for the report summary
// based on phase at which PipelineRun failed
func getHeaderStringForFailureType(ft FailureType) string {
	switch ft {
	case OtherFailure:
		return ":rotating_light: **Couldn't detect a specific failure, see the related PipelineRun for more details or consult with Konflux DevProd team.**\n"
	case TestRunFailure:
		return ":rotating_light: **No JUnit file found, see the log from running tests**: \n"
	case ClusterCreationFailure:
		return ":rotating_light: **Failed to provision a cluster, see the log for more details**: \n"
	case TestCaseFailure:
		return ":rotating_light: **Error occurred while running the E2E tests, list of failed Spec(s)**: \n"
	}
	return ""
}

// GetFormattedReport returns the full report (test run analysis) as a string
func GetFormattedReport(report FailedTestCasesReport) (formattedReport string) {
	formattedReport = getHeaderStringForFailureType(report.FailureType)

	for _, failedTCName := range extractFailedTestCasesBody(report) {
		formattedReport += fmt.Sprintf("\n %s\n", failedTCName)
	}

	return
}

func returnContentWrappedInDropdown(summary, content string) string {
	return "<details><summary>" + summary + "</summary><br><pre>" + content + "</pre></details>"
}
