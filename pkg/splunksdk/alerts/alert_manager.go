package alerts

import (
	"net/http"
	"net/url"

	splunk "github.com/keptn-sandbox/splunk-sli-provider/pkg/splunksdk/client"
)

func PostAlert(client *splunk.SplunkClient, spAlert *AlertRequest) (*http.Response, error) {

	return HttpAlertRequest(client, http.MethodPost, spAlert)
}

func GetAlerts(client *splunk.SplunkClient) (*http.Response, error) {

	return HttpAlertRequest(client, http.MethodGet, nil)
}

func DeleteAlert(client *splunk.SplunkClient, spAlert *AlertRequest) (*http.Response, error) {

	return HttpAlertRequest(client, "DELETE", spAlert)
}

func HttpAlertRequest(client *splunk.SplunkClient, method string, spAlert *AlertRequest) (*http.Response, error) {

	if spAlert == nil {
		spAlert = &AlertRequest{}
	}

	spAlert.Params.OutputMode = "json"

	// parameters of the request
	params := url.Values{}
	params.Add("output_mode", spAlert.Params.OutputMode)

	if method == http.MethodPost {

		if spAlert.Params.Name != "" {
			params.Add("name", spAlert.Params.Name)
		}
		if spAlert.Params.Actions != "" {
			params.Add("actions", spAlert.Params.Actions)
		}
		if spAlert.Params.WebhookUrl != "" {
			params.Add("action.webhook.param.url", spAlert.Params.WebhookUrl)
		}
		if spAlert.Params.SearchQuery != "" {
			params.Add("search", spAlert.Params.SearchQuery)
		}
		if spAlert.Params.CronSchedule != "" {
			params.Add("cron_schedule", spAlert.Params.CronSchedule)
		}
		if spAlert.Params.AlertCondition != "" {
			params.Add("alert_condition", spAlert.Params.AlertCondition)
		}
		if spAlert.Params.AlertSuppress != "" {
			params.Add("alert.suppress", spAlert.Params.AlertSuppress)
		}
		if spAlert.Params.AlertSuppressPeriod != "" {
			params.Add("alert.suppress.period", spAlert.Params.AlertSuppressPeriod)
		}

		params.Add("is_scheduled", "1")

		if spAlert.Params.EarliestTime != "" {
			params.Add("dispatch.earliest_time", spAlert.Params.EarliestTime)
		}
		if spAlert.Params.LatestTime != "" {
			params.Add("dispatch.latest_time", spAlert.Params.LatestTime)
		}

		params.Add("alert_type", "custom")

		if spAlert.Params.Description != "" {
			params.Add("description", spAlert.Params.Description)
		}

		params.Add("alert.track", "1")

	}
	if spAlert.Headers == nil {
		spAlert.Headers = map[string]string{"Content-Type": "application/x-www-form-urlencoded"}
	}
	return splunk.MakeHttpRequest(client, method, spAlert.Headers, params)
}
