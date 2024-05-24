package webhook

import "github.com/spf13/cobra"

// WebhookCmd is a cobra command for triggering webhooks
var WebhookCmd = &cobra.Command{
	Use:   "webhook",
	Short: "Command for triggering webhooks",
}

func init() {
	WebhookCmd.AddCommand(reportPortalWebhookCmd)
}
