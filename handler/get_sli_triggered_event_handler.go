package handler

import (
	"fmt"

	splunk "github.com/keptn-sandbox/splunk-sli-provider/pkg/splunksdk/client"
	splunkjobs "github.com/keptn-sandbox/splunk-sli-provider/pkg/splunksdk/jobs"
	"github.com/keptn-sandbox/splunk-sli-provider/pkg/utils"

	cloudevents "github.com/cloudevents/sdk-go/v2" // make sure to use v2 cloudevents here
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
	logger "github.com/sirupsen/logrus"
)

const sliFileUri = "splunk/sli.yaml"
const KeptnSuffix = "keptn"
const serviceName = "splunk-sli-provider"

// HandleGetSliTriggeredEvent handles get-sli.triggered events if SLIProvider == splunk
func HandleGetSliTriggeredEvent(ddKeptn *keptnv2.Keptn, incomingEvent cloudevents.Event, data *keptnv2.GetSLITriggeredEventData, client *splunk.SplunkClient) error {
	var shkeptncontext string
	_ = incomingEvent.Context.ExtensionAs("shkeptncontext", &shkeptncontext)
	utils.ConfigureLogger(incomingEvent.Context.GetID(), shkeptncontext, "LOG_LEVEL")

	logger.Infof("Handling get-sli.triggered Event: %s", incomingEvent.Context.GetID())

	// Step 1 - Do we need to do something?
	// Lets make sure we are only processing an event that really belongs to our SLI Provider
	if data.GetSLI.SLIProvider != "splunk" {
		logger.Infof("Not handling get-sli event as it is meant for %s", data.GetSLI.SLIProvider)
		return nil
	}

	// Step 2 - Send out a get-sli.started CloudEvent
	// The get-sli.started cloud-event is new since Keptn 0.8.0 and is required to be send when the task is started
	_, err := ddKeptn.SendTaskStartedEvent(data, serviceName)
	if err != nil {
		err := fmt.Errorf("failed to send task started CloudEvent (%w), aborting... ", err)
		logger.Error(err)
		return err
	}

	// Step 4 - prep-work
	// Get any additional input / configuration data
	// - Labels: get the incoming labels for potential config data and use it to pass more labels on result, e.g: links
	// - SLI.yaml: if your service uses SLI.yaml to store query definitions for SLIs get that file from Keptn
	labels := data.Labels
	if labels == nil {
		labels = make(map[string]string)
	}

	// Step 5 - get SLI Config File
	// Get SLI File from splunk subdirectory of the config repo - to add the file use:
	//   keptn add-resource --project=PROJECT --stage=STAGE --service=SERVICE --resource=my-sli-config.yaml  --resourceUri=splunk/sli.yaml
	sliConfig, err := ddKeptn.GetSLIConfiguration(data.Project, data.Stage, data.Service, sliFileUri)
	// FYI you do not need to "fail" if sli.yaml is missing, you can also assume smart defaults like we do
	// in keptn-contrib/dynatrace-service and keptn-sandbox/splunk-sli-provider
	logger.Infof("SLI Config: %s", sliConfig)
	if err != nil {
		// failed to fetch sli config file
		err := fmt.Errorf("failed to fetch SLI file %s from config repo: %w", sliFileUri, err)
		logger.Error(err)
		// send a get-sli.finished event with status=error and result=failed back to Keptn

		_, _ = ddKeptn.SendTaskFinishedEvent(&keptnv2.EventData{
			Status: keptnv2.StatusErrored,
			Result: keptnv2.ResultFailed,
			Labels: labels,
		}, serviceName)

		return err
	}
	// Step 6 - do your work - iterate through the list of requested indicators and return their values
	// Indicators: this is the list of indicators as requested in the SLO.yaml
	// SLIResult: this is the array that will receive the results
	indicators := data.GetSLI.Indicators
	sliResults := []*keptnv2.SLIResult{}

	logger.Info("indicators:", indicators)
	var sliResult *keptnv2.SLIResult

	for _, indicatorName := range indicators {
		sliResult, err = handleSpecificSLI(client, indicatorName, data, sliConfig)
		if err != nil {
			break
		}

		sliResults = append(sliResults, sliResult)
	}

	logger.Infof("SLI Results: %v", sliResults)
	// Step 7 - Build get-sli.finished event data
	getSliFinishedEventData := &keptnv2.GetSLIFinishedEventData{
		EventData: keptnv2.EventData{
			Status: keptnv2.StatusSucceeded,
			Result: keptnv2.ResultPass,
			Labels: labels,
		},
		GetSLI: keptnv2.GetSLIFinished{
			IndicatorValues: sliResults,
			Start:           data.GetSLI.Start,
			End:             data.GetSLI.End,
		},
	}

	if err != nil {
		getSliFinishedEventData.EventData.Status = keptnv2.StatusErrored
		getSliFinishedEventData.EventData.Result = keptnv2.ResultFailed
		getSliFinishedEventData.EventData.Message = fmt.Sprintf("error from the %s while getting slis : %v", serviceName, err)
	}

	logger.Infof("SLI finished event: %v", *getSliFinishedEventData)

	_, err = ddKeptn.SendTaskFinishedEvent(getSliFinishedEventData, serviceName)

	if err != nil {
		err := fmt.Errorf("failed to send task finished CloudEvent (%w), aborting... ", err)
		logger.Error(err)
		return err
	}

	return nil
}

// Executes the splunk search and return the metric value
func handleSpecificSLI(client *splunk.SplunkClient, indicatorName string, data *keptnv2.GetSLITriggeredEventData, sliConfig map[string]string) (*keptnv2.SLIResult, error) {

	query := sliConfig[indicatorName]
	params := splunkjobs.SearchParams{
		SearchQuery:  query,
		EarliestTime: data.GetSLI.Start,
		LatestTime:   data.GetSLI.End,
	}

	// take the time range from the sli file if it is set
	params.EarliestTime, params.LatestTime, params.SearchQuery = utils.RetrieveQueryTimeRange(params.EarliestTime, params.LatestTime, params.SearchQuery)
	logger.Infof("actual query sent to splunk: %v, from: %v, to: %v", params.SearchQuery, params.EarliestTime, params.LatestTime)

	if query == "" {
		return nil, fmt.Errorf("no query found for indicator %s", indicatorName)
	}

	spReq := splunkjobs.SearchRequest{
		Params:  params,
		Headers: map[string]string{},
	}

	// get the metric we want
	sliValue, err := splunkjobs.GetMetricFromNewJob(client, &spReq)
	if err != nil {
		return nil, fmt.Errorf("error getting value for the query: %v : %w", spReq.Params.SearchQuery, err)
	}

	logger.Infof("response from the metrics api: %v", sliValue)

	sliResult := &keptnv2.SLIResult{
		Metric:  indicatorName,
		Value:   sliValue,
		Success: true,
	}
	logger.WithFields(logger.Fields{"indicatorName": indicatorName}).Infof("SLI result from the metrics api: %v", sliResult)

	return sliResult, nil
}
