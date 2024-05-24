package sendslackmessage

import (
	"fmt"
	"os"
	"strings"

	"github.com/slack-go/slack"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/klog/v2"
)

var messageText string

// SendSlackMessageCmd defines to send messages to a Slack channel.
var SendSlackMessageCmd = &cobra.Command{
	Use:   "send-slack-message",
	Short: "This command will send message to any slack channel.",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		requiredEnvVars := []string{"slack_token", "channel_id"}
		for _, e := range requiredEnvVars {
			if viper.GetString(e) == "" {
				return fmt.Errorf("%+v env var not set", strings.ToUpper(e))
			}
		}
		return nil
	},
	Run: run,
}

func sendSlackMessage(token, channelID, message string) error {
	api := slack.New(token)

	_, _, err := api.PostMessage(
		channelID,
		slack.MsgOptionText(message, false),
		slack.MsgOptionAsUser(true),
	)
	return err
}

func run(cmd *cobra.Command, args []string) {
	slackToken := os.Getenv("SLACK_TOKEN")
	slackChannelID := os.Getenv("CHANNEL_ID")

	err := sendSlackMessage(slackToken, slackChannelID, messageText)
	if err != nil {
		fmt.Printf("Error sending message to Slack: %v\n", err)
	}
	klog.Info("message was delivered successfully")
}

func init() {
	SendSlackMessageCmd.Flags().StringVarP(&messageText, "message", "m", "", "Message body of the Slack message")

	if err := SendSlackMessageCmd.MarkFlagRequired("message"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking 'message' flag as required: %v\n", err)
		os.Exit(1)
	}

	if err := viper.BindPFlag("message", SendSlackMessageCmd.Flags().Lookup("message")); err != nil {
		fmt.Fprintf(os.Stderr, "Error binding message flag to viper: %v\n", err)
		os.Exit(1)
	}
}
