package prowjob

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"golang.org/x/exp/slices"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/klog/v2"
)

var healthCheckConfig HealthCheckConfig

type HealthCheckConfig struct {
	ExternalServices []Service `json:"externalServices"`
}

type HealthCheckStatus struct {
	ExternalServices            []Service `json:"externalServices"`
	UnhealthyCriticalComponents []string  `json:"unhealthyCriticalComponents"`
}

type Service struct {
	Name               string   `json:"name"`
	CriticalComponents []string `json:"criticalComponents"`
	StatusPageURL      string   `json:"statusPageURL"`
	CurrentStatus      Summary  `json:"currentStatus"`
}

// Summary is the Statuspage API component representation
type Summary struct {
	Components []Component `json:"components"`
	Incidents  []Incident  `json:"incidents"`
	Status     Status      `json:"status"`
}

// Component is the Statuspage API component representation
type Component struct {
	ID                 string    `json:"id,omitempty"`
	PageID             string    `json:"page_id,omitempty"`
	GroupID            string    `json:"group_id,omitempty"`
	CreatedAt          time.Time `json:"created_at,omitempty"`
	UpdatedAt          time.Time `json:"updated_at,omitempty"`
	Group              bool      `json:"group,omitempty"`
	Name               string    `json:"name,omitempty"`
	Description        string    `json:"description,omitempty"`
	Position           int32     `json:"position,omitempty"`
	Status             string    `json:"status,omitempty"`
	Showcase           bool      `json:"showcase,omitempty"`
	OnlyShowIfDegraded bool      `json:"only_show_if_degraded,omitempty"`
	AutomationEmail    string    `json:"automation_email,omitempty"`
}

// Incident entity reflects one single incident
type Incident struct {
	ID                string           `json:"id,omitempty"`
	Name              string           `json:"name,omitempty"`
	Status            string           `json:"status,omitempty"`
	Message           string           `json:"message,omitempty"`
	Visible           int              `json:"visible,omitempty"`
	ComponentID       int              `json:"component_id,omitempty"`
	ComponentStatus   int              `json:"component_status,omitempty"`
	Notify            bool             `json:"notify,omitempty"`
	Stickied          bool             `json:"stickied,omitempty"`
	OccurredAt        string           `json:"occurred_at,omitempty"`
	Template          string           `json:"template,omitempty"`
	Vars              []string         `json:"vars,omitempty"`
	CreatedAt         string           `json:"created_at,omitempty"`
	UpdatedAt         string           `json:"updated_at,omitempty"`
	DeletedAt         string           `json:"deleted_at,omitempty"`
	IsResolved        bool             `json:"is_resolved,omitempty"`
	Updates           []IncidentUpdate `json:"incident_updates,omitempty"`
	HumanStatus       string           `json:"human_status,omitempty"`
	LatestUpdateID    int              `json:"latest_update_id,omitempty"`
	LatestStatus      int              `json:"latest_status,omitempty"`
	LatestHumanStatus string           `json:"latest_human_status,omitempty"`
	LatestIcon        string           `json:"latest_icon,omitempty"`
	Permalink         string           `json:"permalink,omitempty"`
	Duration          int              `json:"duration,omitempty"`
}

// IncidentUpdate entity reflects one single incident update
type IncidentUpdate struct {
	ID              string `json:"id,omitempty"`
	Body            string `json:"body,omitempty"`
	IncidentID      string `json:"incident_id,omitempty"`
	ComponentID     int    `json:"component_id,omitempty"`
	ComponentStatus int    `json:"component_status,omitempty"`
	Status          string `json:"status,omitempty"`
	Message         string `json:"message,omitempty"`
	UserID          int    `json:"user_id,omitempty"`
	CreatedAt       string `json:"created_at,omitempty"`
	UpdatedAt       string `json:"updated_at,omitempty"`
	HumanStatus     string `json:"human_status,omitempty"`
	Permalink       string `json:"permalink,omitempty"`
}

// Status entity contains the contents of API Response of a /status call.
type Status struct {
	Indicator   string `json:"indicator,omitempty"`
	Description string `json:"description,omitempty"`
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
		status := &HealthCheckStatus{}
		status.ExternalServices = healthCheckConfig.ExternalServices
		for i, service := range status.ExternalServices {
			r, err := http.Get(service.StatusPageURL)
			if err != nil {
				return fmt.Errorf("failed to get service %s status page: %+v", service.Name, err)
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				return fmt.Errorf("failed to read response body for a service %s: %+v", service.Name, err)
			}
			v := Summary{}
			if err := json.Unmarshal(body, &v); err != nil {
				return fmt.Errorf("failed to unmarshal response body from a service %s: %+v", service.Name, err)
			}
			status.ExternalServices[i].CurrentStatus = v

			for _, c := range v.Components {
				if c.Status == "major_outage" && isCriticalComponent(service, c) {
					desc := fmt.Sprintf("%s: %s", service.Name, c.Name)
					status.UnhealthyCriticalComponents = append(status.UnhealthyCriticalComponents, desc)
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
		o, err := json.MarshalIndent(status, "", "    ")
		if err != nil {
			return fmt.Errorf("failed to marshal services status: %+v", err)
		}
		reportFilePath := artifactDir + "/services-status.json"
		if err := os.WriteFile(reportFilePath, []byte(o), 0o600); err != nil {
			return fmt.Errorf("failed to create file with the status of dependant services: %+v", err)
		}
		klog.Infof("health check report saved to %s", reportFilePath)

		if viper.GetBool(failIfUnhealthyParamName) {
			// for s := range status.Services.Github.Components {

			// }
			return fmt.Errorf("TESTING UNHEALTHY!")
		}

		return nil
	},
}

func isCriticalComponent(service Service, c Component) bool {
	return slices.Contains(service.CriticalComponents, c.Name)
}

func init() {
	healthCheckCmd.Flags().BoolVar(&failIfUnhealthy, failIfUnhealthyParamName, false, "Exit with non-zero code if health check fails")

	_ = viper.BindPFlag(artifactDirParamName, healthCheckCmd.Flags().Lookup(artifactDirParamName))
	_ = viper.BindPFlag(failIfUnhealthyParamName, healthCheckCmd.Flags().Lookup(failIfUnhealthyParamName))
	// Bind environment variables to viper (in case the associated command's parameter is not provided)
	_ = viper.BindEnv(artifactDirParamName, artifactDirEnv)
}
