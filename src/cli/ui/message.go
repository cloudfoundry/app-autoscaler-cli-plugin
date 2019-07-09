package ui

const (
	OK     = "OK"
	FAILED = "FAILED"

	NOCFAPIEndpoint    = "No Cloud Foundry api endpoint set. Use 'cf api' to set Cloud Foundry endpoint first."
	NoEndpoint         = "No AutoScaler api endpoint set. Use 'cf autoscaling-api' to set an endpoint."
	APIEndpoint        = "Autoscaler api endpoint: %s"
	SetAPIEndpoint     = "Setting AutoScaler api endpoint to %s..."
	UnsetAPIEndpoint   = "Unsetting AutoScaler api endpoint."
	InvalidAPIEndpoint = "Invalid AutoScaler API endpoint : %s"
	InvalidSSLCerts    = "Invalid SSL Cert for %s \nTIP: Use --skip-ssl-validation to continue with an insecure API endpoint."
	InconsistentDomain = "Failed to set AutoScaler domain to %s since it is inconsitent with the domain of CF API %s."

	Unauthorized  = "Unauthorized. Failed to access AutoScaler API endpoint %s."
	LoginRequired = "You must be logged in %s first."

	FailToLoadPolicyFile = "Failed to read policy file %s."
	PolicyNotFound       = "No policy defined for app %s."
	InvalidPolicy        = "Invalid policy definition: %v."

	ShowPolicyHint   = "Retrieving policy for app %s..."
	AttachPolicyHint = "Attaching policy for app %s..."
	DetachPolicyHint = "Detaching policy for app %s..."

	ShowAggregatedMetricsHint = "Retrieving aggregated %s metrics for app %s..."
	ShowHistoryHint           = "Retrieving scaling event history for app %s..."

	SavePolicyHint           = "Saving policy for app %s to %s... "
	SaveCredentialHint       = "Saving custom metric credential for app %s to %s... "
	SaveAggregatedMetricHint = "Saving aggregated metrics for app %s to %s... "
	SaveHistoryHint          = "Saving scaling event history for app %s to %s... "

	UnrecognizedTimeFormat = "Unrecognized date time format: %s. \nSupported formats are yyyy-MM-ddTHH:mm:ss+/-hhmm, yyyy-MM-ddTHH:mm:ssZ with an input later than 1970-01-01T00:00:00Z."
	UnrecognizedMetricName = "Unrecognized metric name: %s. \nSupported value: memoryused, memoryutil, responsetime, throughput, cpu."
	InvalidTimeRange       = "Invalid time range. The start time %s is greater than the end time %s."

	AggregatedMetricsNotFound = "No aggregated %s metrics were found for app %s."
	HistoryNotFound           = "No event history were found for app %s."

	MoreRecordsWarning    = "TIP: More records available. Please re-run the command with --start or --end option to fetch more."
	DeprecatedDescWarning = "TIP: The default order is set to descending now. Please remove the DEPRECATED flag '--desc'."

	ShowCredentialHint   = "Retrieving custom metrics credential for app %s..."
	CreateCredentialHint = "Creating custom metrics credential for app %s..."
	DeleteCredentialHint = "Deleting custom metrics credential for app %s..."
)
