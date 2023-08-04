package utils

type EnvConfig struct {
	// Port on which to listen for cloudevents
	Port int `envconfig:"RCV_PORT" default:"8080"`
	// Path to which cloudevents are sent
	Path string `envconfig:"RCV_PATH" default:"/"`
	// Whether we are running locally (e.g., for testing) or on production
	Env string `envconfig:"ENV" default:"local"`
	// URL of the Keptn configuration service (this is where we can fetch files from the config repo)

	ConfigurationServiceUrl string `envconfig:"CONFIGURATION_SERVICE" default:""`

	SplunkApiToken   string `envconfig:"SP_API_TOKEN" default:""`
	SplunkHost       string `envconfig:"SP_HOST" default:""`
	SplunkPort       string `envconfig:"SP_PORT" default:"8089"`
	SplunkUsername   string `envconfig:"SP_USERNAME" default:""`
	SplunkPassword   string `envconfig:"SP_PASSWORD" default:""`
	SplunkSessionKey string `envconfig:"SP_SESSION_KEY" default:""`

	AlertSuppressPeriod  string `envconfig:"ALERT_SUPPRESS_PERIOD" default:"3m"`
	CronSchedule         string `envconfig:"CRON_SCHEDULE" default:"3m"`
	DispatchEarliestTime string `envconfig:"DISPATCH_EARLIEST_TIME" default:"*/1 * * * *"`
	DispatchLatestTime   string `envconfig:"DISPATCH_LATEST_TIME" default:"now"`
	Actions              string `envconfig:"ACTIONS" default:""`
	WebhookUrl           string `envconfig:"WEBHOOK_URL" default:""`
}
