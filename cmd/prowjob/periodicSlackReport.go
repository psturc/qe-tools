package prowjob

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// periodicSlackReportCmd returns the periodic-slack-report command
var periodicSlackReportCmd = &cobra.Command{
	Use:   "periodic-slack-report",
	Short: "Analyzes the build log from latest periodic job and sends a summary of detected issues to dedicated Slack channel",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		requiredEnvVars := []string{"slack_token", "channel_id", "url", "prow_url"}

		for _, e := range requiredEnvVars {
			if viper.GetString(e) == "" {
				return fmt.Errorf("%+v env var not set", strings.ToUpper(e))
			}
		}
		return nil
	},
	RunE:         run,
	SilenceUsage: true,
}

func removeANSIEscapeSequences(text string) string {
	regex := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return regex.ReplaceAllString(text, "")
}

func fetchTextContent(url string) (string, error) {
	// #nosec G107
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("error fetching the webpage: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading the webpage content: %w", err)
	}

	return string(bodyBytes), nil
}

func sendMessageToLatestThread(token, channelID, message string) error {
	slackURL := "https://slack.com/api/chat.postMessage"

	payload := url.Values{}
	payload.Set("channel", channelID)
	payload.Set("text", message)

	req, err := http.NewRequest("POST", slackURL, strings.NewReader(payload.Encode()))
	if err != nil {
		return fmt.Errorf("error creating the request: %w", err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending the request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

func constructMessage(content, bodyString string) (string, bool) {
	var message string
	const statePattern = `Reporting job state '(\w+)'`
	const failurePattern = `(?s)(Summarizing.*?Test Suite Failed)`
	const durationPattern = `Ran for ([\dhms]+)`

	stateRegexp := regexp.MustCompile(statePattern)
	stateMatches := stateRegexp.FindStringSubmatch(bodyString)

	hasFailed := len(stateMatches) == 2 && stateMatches[1] == "failed"
	if !hasFailed {
		return "", false
	}

	failureRegexp := regexp.MustCompile(failurePattern)
	failureMatches := failureRegexp.FindStringSubmatch(bodyString)

	failureSummary := ""
	if failureMatches == nil {
		failureSummary = "Infrastructure setup issues or failures unrelated to tests were found. No report of test failures was produced. \n"
	} else {
		failureSummary = removeANSIEscapeSequences(failureMatches[1]) + "\n"
	}

	message += failureSummary
	message += fmt.Sprintf("Reporting job state: %s\n", strings.TrimSpace(stateMatches[1]))

	durationRegexp := regexp.MustCompile(durationPattern)
	durationMatches := durationRegexp.FindStringSubmatch(bodyString)

	if len(durationMatches) >= 2 {
		message += fmt.Sprintf("Ran for %s\n", durationMatches[1])
	}

	return message, true
}

func run(cmd *cobra.Command, args []string) error {
	token := os.Getenv("SLACK_TOKEN")
	channelID := os.Getenv("CHANNEL_ID")

	url := os.Getenv("URL")
	content, err := fetchTextContent(url)
	if err != nil {
		return err
	}

	prowURL := fmt.Sprintf(os.Getenv("PROW_URL"), content)
	bodyString, err := fetchTextContent(prowURL)
	if err != nil {
		return err
	}

	message, sendSlackMessage := constructMessage(content, bodyString)

	fmt.Println(message)

	if sendSlackMessage {
		err = sendMessageToLatestThread(token, channelID, message)
		if err != nil {
			return err
		}

		fmt.Println("Slack message sent successfully!")
	} else {
		fmt.Println("No test failures found. Slack message not sent.")
	}
	return nil
}
