package models

type ScalingType int

type ScalingStatus int

type ScalingPolicy struct {
	InstanceMin  int               `json:"instance_min_count"`
	InstanceMax  int               `json:"instance_max_count"`
	ScalingRules []*ScalingRule    `json:"scaling_rules,omitempty"`
	Schedules    *ScalingSchedules `json:"schedules,omitempty"`
}

type ScalingRule struct {
	MetricType            string `json:"metric_type"`
	StatWindowSeconds     int    `json:"stat_window_secs,omitempty"`
	BreachDurationSeconds int    `json:"breach_duration_secs,omitempty"`
	Threshold             int64  `json:"threshold"`
	Operator              string `json:"operator"`
	CoolDownSeconds       int    `json:"cool_down_secs,omitempty"`
	Adjustment            string `json:"adjustment"`
}

type ScalingSchedules struct {
	Timezone              string                  `json:"timezone"`
	RecurringSchedules    []*RecurringSchedule    `json:"recurring_schedule,omitempty"`
	SpecificDateSchedules []*SpecificDateSchedule `json:"specific_date,omitempty"`
}

type RecurringSchedule struct {
	StartTime             string `json:"start_time"`
	EndTime               string `json:"end_time"`
	DaysOfWeek            []int  `json:"days_of_week,omitempty"`
	DaysOfMonth           []int  `json:"days_of_month,omitempty"`
	StartDate             string `json:"start_date,omitempty"`
	EndDate               string `json:"end_date,omitempty"`
	ScheduledInstanceMin  int    `json:"instance_min_count"`
	ScheduledInstanceMax  int    `json:"instance_max_count"`
	ScheduledInstanceInit int    `json:"initial_min_instance_count"`
}

type SpecificDateSchedule struct {
	StartDateTime         string `json:"start_date_time"`
	EndDateTime           string `json:"end_date_time"`
	ScheduledInstanceMin  int    `json:"instance_min_count"`
	ScheduledInstanceMax  int    `json:"instance_max_count"`
	ScheduledInstanceInit int    `json:"initial_min_instance_count"`
}

type AppAggregatedMetric struct {
	AppId     string `json:"app_id"`
	Name      string `json:"name"`
	Unit      string `json:"unit"`
	Value     string `json:"value"`
	Timestamp int64  `json:"timestamp"`
}

type AppScalingHistory struct {
	AppId        string        `json:"app_id"`
	Timestamp    int64         `json:"timestamp"`
	ScalingType  ScalingType   `json:"scaling_type"`
	Status       ScalingStatus `json:"status"`
	OldInstances int           `json:"old_instances"`
	NewInstances int           `json:"new_instances"`
	Reason       string        `json:"reason"`
	Message      string        `json:"message"`
	Error        string        `json:"error"`
}

type AggregatedMetricsResults struct {
	TotalResults uint32                 `json:"total_results"`
	TotalPages   uint16                 `json:"total_pages"`
	Page         uint16                 `json:"page"`
	Metrics      []*AppAggregatedMetric `json:"resources"`
}

type HistoryResults struct {
	TotalResults uint32               `json:"total_results"`
	TotalPages   uint16               `json:"total_pages"`
	Page         uint16               `json:"page"`
	Histories    []*AppScalingHistory `json:"resources"`
}

type Credential struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type CredentialResponse struct {
	AppId string `json:"app_id"`
	*Credential
	Url string `json:"url"`
}
