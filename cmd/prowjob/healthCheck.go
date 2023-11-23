package prowjob

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"golang.org/x/exp/slices"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/klog/v2"
	prowUtils "k8s.io/test-infra/prow/pod-utils/downwardapi"

	"github.com/google/go-github/v56/github"
	"github.com/redhat-appstudio/qe-tools/pkg/status"
)

var healthCheckConfig HealthCheckConfig

type HealthCheckConfig struct {
	ExternalServices []Service `json:"externalServices"`
}

type HealthCheckStatus struct {
	ExternalServices            []Service           `json:"externalServices"`
	UnhealthyCriticalComponents map[string][]string `json:"unhealthyCriticalComponents"`
}

type Service struct {
	Name               string         `json:"name"`
	CriticalComponents []string       `json:"criticalComponents"`
	StatusPageURL      string         `json:"statusPageURL"`
	CurrentStatus      status.Summary `json:"currentStatus"`
}

// healthCheckCmd represents the createReport command
var healthCheckCmd = &cobra.Command{
	Use:   "health-check",
	Short: "Perform a health check on dependant services",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		viper.AddConfigPath("./config/health-check")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		if err := viper.ReadInConfig(); err != nil {
			return fmt.Errorf("err readinconfig: %+v", err)
		}
		if err := viper.Unmarshal(&healthCheckConfig); err != nil {
			return fmt.Errorf("failed to parse config: %+v", err)
		}
		return nil
	},
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		hcStatus := &HealthCheckStatus{}
		hcStatus.ExternalServices = healthCheckConfig.ExternalServices
		hcStatus.UnhealthyCriticalComponents = make(map[string][]string)

		for i, service := range hcStatus.ExternalServices {
			r, err := http.Get(service.StatusPageURL)
			if err != nil {
				return fmt.Errorf("failed to get service %s status page: %+v", service.Name, err)
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				return fmt.Errorf("failed to read response body for a service %s: %+v", service.Name, err)
			}
			v := status.Summary{}
			if err := json.Unmarshal(body, &v); err != nil {
				return fmt.Errorf("failed to unmarshal response body from a service %s: %+v", service.Name, err)
			}
			hcStatus.ExternalServices[i].CurrentStatus = v

			for _, c := range v.Components {
				if c.Status == "major_outage" && isCriticalComponent(service, c) {
					hcStatus.UnhealthyCriticalComponents[service.Name] = append(hcStatus.UnhealthyCriticalComponents[service.Name], c.Name)
				}
				// DELETE ME
				if service.Name == "quay" && c.Name == "Registry" {
					hcStatus.UnhealthyCriticalComponents[service.Name] = []string{c.Name, "Test"}
				}

			}

		}

		artifactDir := viper.GetString(artifactDirParamName)
		if artifactDir == "" {
			artifactDir = "./tmp"
			klog.Warningf("path to artifact dir was not provided - using default %q\n", artifactDir)
		}
		if err := os.MkdirAll(artifactDir, 0o750); err != nil {
			return fmt.Errorf("failed to create directory for results '%s': %+v", artifactDir, err)
		}
		o, err := json.MarshalIndent(hcStatus, "", "    ")
		if err != nil {
			return fmt.Errorf("failed to marshal services status: %+v", err)
		}
		reportFilePath := artifactDir + "/services-status.json"
		if err := os.WriteFile(reportFilePath, []byte(o), 0o600); err != nil {
			return fmt.Errorf("failed to create file with the status of dependant services: %+v", err)
		}
		klog.Infof("health check report saved to %s", reportFilePath)

		if len(hcStatus.UnhealthyCriticalComponents) > 0 {

			if viper.GetBool(notifyOnPRParamName) {
				githubToken := viper.GetString(githubTokenEnv)

				if githubToken == "" {
					klog.Warning("GITHUB_TOKEN env var not exported - the health check report on a PR won't be created")
				} else {
					client := github.NewClient(http.DefaultClient).WithAuthToken(githubToken)
					owner := viper.GetString(prowUtils.RepoOwnerEnv)
					repo := viper.GetString(prowUtils.RepoNameEnv)
					prNumber := viper.GetString(prowUtils.PullNumberEnv)
					prNumberInt, _ := strconv.Atoi(prNumber)
					prMessage := "❗ Detected an outage of the following critical component(s)❗\n"
					for s, components := range hcStatus.UnhealthyCriticalComponents {
						prMessage += fmt.Sprintf("- %s: %s\n", s, strings.Join(components, ", "))
					}
					prMessage += "\nDue to this issue **E2E tests won't run on your PR**. Please keep an eye on the following status pages:\n"
					for _, s := range healthCheckConfig.ExternalServices {
						if _, ok := hcStatus.UnhealthyCriticalComponents[s.Name]; ok {
							u, err := url.Parse(s.StatusPageURL)
							if err != nil {
								klog.Errorf("could not parse status page URL %s: %+v", s.StatusPageURL, err)
								continue
							}
							prMessage += fmt.Sprintf("- %s://%s\n", u.Scheme, u.Host)
						}
					}
					prMessage += "\nand add a comment `/retest-required` once the reported issues are solved"
					comment, _, err := client.Issues.CreateComment(context.Background(), owner, repo, prNumberInt, &github.IssueComment{
						Body: github.String(prMessage),
					})
					if err != nil {
						klog.Errorf("couldn't report an issue on a PR: %+v", err)
					}
					klog.Infof("added a report about an outage to %s", comment.GetHTMLURL())

				}
			}

			if viper.GetBool(failIfUnhealthyParamName) {
				return fmt.Errorf("detected unhealthy critical components: %+v - see %s for more info", hcStatus.UnhealthyCriticalComponents, reportFilePath)
			}
		}

		return nil
	},
}

func isCriticalComponent(service Service, c status.Component) bool {
	return slices.Contains(service.CriticalComponents, c.Name)
}

func init() {
	healthCheckCmd.Flags().BoolVar(&failIfUnhealthy, failIfUnhealthyParamName, false, "Exit with non-zero code if critical issues were found")
	healthCheckCmd.Flags().BoolVar(&notifyOnPR, notifyOnPRParamName, false, "Create a comment in a related PR if critical issues were found")

	_ = viper.BindPFlag(artifactDirParamName, healthCheckCmd.Flags().Lookup(artifactDirParamName))
	_ = viper.BindPFlag(failIfUnhealthyParamName, healthCheckCmd.Flags().Lookup(failIfUnhealthyParamName))
	_ = viper.BindPFlag(notifyOnPRParamName, healthCheckCmd.Flags().Lookup(notifyOnPRParamName))
	// Bind environment variables to viper (in case the associated command's parameter is not provided)
	_ = viper.BindEnv(artifactDirParamName, artifactDirEnv)
}
