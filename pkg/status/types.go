package status

import "time"

type ServicesStatus struct {
	Services Services `json:"services"`
}

type Services struct {
	Github Summary `json:"github"`
	Quay   Summary `json:"quay"`
	RedHat Summary `json:"redhat"`
}

// Summary is the Statuspage API component representation
type Summary struct {
	Components []Component `json:"components"`
	Incidents  []Incident  `json:"incidents"`
	Status     Status      `json:"status"`
}

// Status entity contains the contents of API Response of a /status call.
type Status struct {
	Indicator   string `json:"indicator,omitempty"`
	Description string `json:"description,omitempty"`
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
	IncidentID      int    `json:"incident_id,omitempty"`
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
