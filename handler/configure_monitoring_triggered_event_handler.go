package handler

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/AmadeusITGroup/keptn-splunk-sli-provider/alerts"
	splunkalerts "github.com/AmadeusITGroup/keptn-splunk-sli-provider/pkg/splunksdk/alerts"
	splunk "github.com/AmadeusITGroup/keptn-splunk-sli-provider/pkg/splunksdk/client"
	"github.com/AmadeusITGroup/keptn-splunk-sli-provider/pkg/utils"

	cloudevents "github.com/cloudevents/sdk-go/v2" // make sure to use v2 cloudevents here
	api "github.com/keptn/go-utils/pkg/api/utils"
	keptnevents "github.com/keptn/go-utils/pkg/lib"
	"github.com/keptn/go-utils/pkg/lib/keptn"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
	logger "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var createAlert = splunkalerts.CreateAlert

// Handles configure monitoring event
func HandleConfigureMonitoringTriggeredEvent(ddKeptn *keptnv2.Keptn, incomingEvent cloudevents.Event, data *keptnv2.ConfigureMonitoringTriggeredEventData, envConfig utils.EnvConfig, client *splunk.SplunkClient, pollingSystemHasBeenStarted bool) error {

	if isNotForSplunk(data.ConfigureMonitoring.Type) {
		logger.Infof("Event is not for splunk but for %s", data.ConfigureMonitoring.Type)
		return fmt.Errorf("event is not for splunk but for %s", data.ConfigureMonitoring.Type)
	}

	if isProjectOrServiceNotSet(data) {
		logger.Infof("A project and a service have to be defined")
		return fmt.Errorf("a project and a service have to be defined")
	}

	var shkeptncontext string
	//Configuring the logger
	_ = incomingEvent.Context.ExtensionAs("shkeptncontext", &shkeptncontext)
	utils.ConfigureLogger(incomingEvent.Context.GetID(), shkeptncontext, "LOG_LEVEL")

	//Sending the configure monitoring started event
	logger.Infof("Handling configure-monitoring.triggered Event: %s", incomingEvent.Context.GetID())
	_, err := ddKeptn.SendTaskStartedEvent(data, serviceName)
	if err != nil {
		logger.Errorf("err when sending task started the event: %v", err)
		return err
	}

	//Creating the alerts
	setPollingSystem, err := CreateSplunkAlertsForEachStage(client, ddKeptn, *data, envConfig)
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	switch {
	case !pollingSystemHasBeenStarted && setPollingSystem:
		go func() {
			// Starts polling for triggered alerts if configure monitoring is successful
			alerts.FiringAlertsPoll(client, ddKeptn, keptn.KeptnOpts{}, envConfig)
		}()
	case pollingSystemHasBeenStarted:
		logger.Info("Polling system has already been started")
	default:
		logger.Info("No alerts configured, no need to start the polling system")
	}

	//Making the configure monitoring finished event
	configureMonitoringFinishedEventData := &keptnv2.ConfigureMonitoringFinishedEventData{
		EventData: keptnv2.EventData{
			Status:  keptnv2.StatusSucceeded,
			Result:  keptnv2.ResultPass,
			Project: data.Project,
			Stage:   "",
			Service: data.Service,
			Message: "Finished configuring monitoring",
		},
	}

	logger.Infof("Configure Monitoring finished event: %v", *configureMonitoringFinishedEventData)

	// Sending the Configure Monitoring finished event
	_, err = ddKeptn.SendTaskFinishedEvent(configureMonitoringFinishedEventData, serviceName)
	if err != nil {
		err := fmt.Errorf("failed to send task finished CloudEvent (%w), aborting... ", err)
		logger.Error(err)
		return err
	}

	return nil
}

// Creates alerts for each stage defined in the shipyard file after removing potential ancient alerts of the service
func CreateSplunkAlertsForEachStage(client *splunk.SplunkClient, k *keptnv2.Keptn, eventData keptnv2.ConfigureMonitoringTriggeredEventData, envConfig utils.EnvConfig) (bool, error) {

	logger.Infof("Removing previous alerts set for the service %v in project %v", eventData.Service, eventData.Project)

	//listing all alerts
	alertsList, err := splunkalerts.ListAlertsNames(client)
	if err != nil {
		logger.Errorf("Error calling ListAlertsNames(): %v : %v", alertsList, err)
		return false, fmt.Errorf("error calling ListAlertsNames(): %v : %w", alertsList, err)
	}

	//removing all preexisting alerts concerning the project and the service
	for _, alert := range alertsList.Item {
		if strings.HasSuffix(alert.Name, KeptnSuffix) && strings.Contains(alert.Name, eventData.Project) && strings.Contains(alert.Name, eventData.Service) {
			logger.Infof("Removing alert %v", alert.Name)
			err := splunkalerts.RemoveAlert(client, alert.Name)
			if err != nil {
				logger.Errorf("Error calling RemoveAlert(): %v : %v", alertsList, err)
				return false, fmt.Errorf("error calling RemoveAlert(): %v : %w", alertsList, err)
			}
		}
	}

	// if no alerts are configured, no need to start the polling system
	var setPollingSystem bool
	//Getting the shipyard configuration
	scope := api.NewResourceScope()
	scope.Project(eventData.Project)
	scope.Resource("shipyard.yaml")

	shipyard, err := k.GetShipyard()
	if err != nil {
		return false, err
	}

	//Creating the alerts for each stage of the shipyard file
	for _, stage := range shipyard.Spec.Stages {
		logger.Infof("Creating alerts for stage : %v", stage)
		setPollingSystemTmp, err := CreateSplunkAlerts(client, k, eventData, stage, envConfig)
		if err != nil {
			return false, fmt.Errorf("error configuring splunk alerts: %w", err)
		}
		if setPollingSystemTmp {
			setPollingSystem = true
		}
	}

	return setPollingSystem, nil
}

// Creates the splunk alerts of a particular stage if slo.yaml and remediation.yaml files are defined
func CreateSplunkAlerts(client *splunk.SplunkClient, k *keptnv2.Keptn, eventData keptnv2.ConfigureMonitoringTriggeredEventData, stage keptnv2.Stage, envConfig utils.EnvConfig) (bool, error) {

	//Trying to retrieve SLO file
	slos, err := retrieveSLOs(k.ResourceHandler, eventData, stage.Name)
	if err != nil || slos == nil {
		logger.Info("No SLO file found for stage " + stage.Name + " error : " + err.Error() + ". No alerting rules created for this stage")
		return false, nil
	}

	const remediationFileDefaultName = "remediation.yaml"

	//Trying to retrieve remediation file
	resourceScope := api.NewResourceScope()
	resourceScope.Project(eventData.Project)
	resourceScope.Service(eventData.Service)
	resourceScope.Stage(stage.Name)
	resourceScope.Resource(remediationFileDefaultName)

	_, err = k.ResourceHandler.GetResource(*resourceScope)

	if errors.Is(err, api.ResourceNotFoundError) {
		logger.Infof("No remediation defined for project %s stage %s, skipping setup of splunk alerts",
			eventData.Project, stage.Name)
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("error retrieving remediation definition %s for project %s and stage %s: %w",
			remediationFileDefaultName, eventData.Project, stage.Name, err)
	}

	// get SLI searches
	projectCustomQueries, err := getCustomQueries(k, eventData.Project, stage.Name, eventData.Service)
	if err != nil {
		log.Println("Failed to get custom queries for project " + eventData.Project)
		log.Println(err.Error())
		return false, err
	}

	logger.Info("Going over SLO.objectives")

	//For each objective
	if len(slos.Objectives) == 0 {
		logger.Info("No objectives defined in the SLO file for stage " + stage.Name + ". No alerting rules created for this stage")
		return false, nil
	}
	for _, objective := range slos.Objectives {
		logger.Info("SLO: " + objective.DisplayName + ", " + objective.SLI)

		//getting the splunk search query for the objective
		query := projectCustomQueries[objective.SLI]

		if err != nil || query == "" {
			logger.Error("No query defined for SLI " + objective.SLI + " in project " + eventData.Project)
			continue
		}
		logger.Info("query= " + query)

		//getting the name of the result field of the splunk sli search
		resultField, err := getResultFieldName(query)
		if err != nil {
			log.Println("Failed to get the result field name in order to create the alert condition for " + eventData.Project)
			log.Println(err.Error())
			return false, err
		}

		//For each criteria of each pass criteria group of an objective (corresponding to an sli)
		if objective.Pass != nil {
			for _, criteriaGroup := range objective.Pass {
				for _, criteria := range criteriaGroup.Criteria {

					//building the splunk alert condition
					//TO SUPPORT RELATIVE CRITERIA I'LL HAVE TO MODIFY THAT PART
					if strings.Contains(criteria, "+") || strings.Contains(criteria, "-") || strings.Contains(
						criteria, "%",
					) || (!strings.Contains(criteria, "<") && !strings.Contains(criteria, ">")) {
						continue
					}

					switch {
					case strings.Contains(criteria, "<="):
						criteria = strings.Replace(criteria, "<=", ">", -1)
					case strings.Contains(criteria, "<"):
						criteria = strings.Replace(criteria, "<", ">=", -1)
					case strings.Contains(criteria, ">="):
						criteria = strings.Replace(criteria, ">=", "<", -1)
					case strings.Contains(criteria, ">"):
						criteria = strings.Replace(criteria, ">", "<=", -1)
					case strings.Contains(criteria, "="):
						criteria = strings.Replace(criteria, "=", "!=", -1)
					default:
						criteria = strings.Replace(criteria, "!=", "=", -1)
					}

					//Sanitize criteria : remove whitespaces
					criteria = strings.Replace(criteria, " ", "", -1)

					//Setting some alert parameters
					alertCondition := buildAlertCondition(resultField, criteria)
					alertName := buildAlertName(eventData, stage.Name, objective.SLI, criteria)
					cronSchedule := "*/1 * * * *"
					alertSuppress := "1"

					//Creates the alert datastructure
					params := splunkalerts.AlertParams{
						Name:                alertName,
						CronSchedule:        cronSchedule,
						SearchQuery:         query,
						EarliestTime:        envConfig.DispatchEarliestTime,
						LatestTime:          envConfig.DispatchLatestTime,
						AlertCondition:      alertCondition,
						AlertSuppress:       alertSuppress,
						AlertSuppressPeriod: envConfig.AlertSuppressPeriod,
						Actions:             envConfig.Actions,
						WebhookUrl:          envConfig.WebhookUrl,
					}
					params.EarliestTime, params.LatestTime, params.SearchQuery = utils.RetrieveQueryTimeRange(params.EarliestTime, params.LatestTime, params.SearchQuery)

					spAlert := splunkalerts.AlertRequest{
						Params:  params,
						Headers: map[string]string{},
					}

					//Creates the alert in splunk
					err = createAlert(client, &spAlert)
					if err != nil {
						logger.Errorf("Error calling CreateAlert(): %v : %v", spAlert.Params.SearchQuery, err)
						return false, fmt.Errorf("error calling CreateAlert(): %v : %w", spAlert.Params.SearchQuery, err)
					}

				}
			}
		}
	}
	return true, nil
}

// Retrieves the SLOs from the slo.yaml file
func retrieveSLOs(resourceHandler *api.ResourceHandler, eventData keptnv2.ConfigureMonitoringTriggeredEventData, stage string) (*keptnevents.ServiceLevelObjectives, error) {
	resourceScope := api.NewResourceScope()
	resourceScope.Project(eventData.Project)
	resourceScope.Service(eventData.Service)
	resourceScope.Stage(stage)
	resourceScope.Resource("slo.yaml")

	resource, err := resourceHandler.GetResource(*resourceScope)
	if err != nil || resource.ResourceContent == "" {
		return nil, fmt.Errorf("No SLO file available for service %s in stage %s", eventData.Service, stage)
	}
	var slos keptnevents.ServiceLevelObjectives

	err = yaml.Unmarshal([]byte(resource.ResourceContent), &slos)

	if err != nil {
		return nil, fmt.Errorf("invalid SLO file format")
	}

	return &slos, nil
}

// Returns the splunk searches defined in the sli.yaml file
func getCustomQueries(k *keptnv2.Keptn, project string, stage string, service string) (map[string]string, error) {
	log.Println("Checking for custom SLI queries")

	customQueries, err := k.GetSLIConfiguration(project, stage, service, sliFileUri)
	if err != nil {
		return nil, err
	}

	return customQueries, nil
}

// Returns the name of the splunk search result
func getResultFieldName(searchQuery string) (string, error) {
	//returns the first word after "stats" in the search
	if strings.Contains(searchQuery, "stats") {
		startIndex := strings.Index(searchQuery, "stats") + 4
		i := 0
		for {
			i = i + 1
			if (startIndex+i == len(searchQuery) || searchQuery[startIndex+i] == " "[0]) && searchQuery[startIndex+i-1] != "s"[0] {
				return searchQuery[startIndex+2 : startIndex+i], nil
			}
			if startIndex+i == len(searchQuery) {
				break
			}
		}
	}
	return "", fmt.Errorf("no aggregation function found in the search query")
}

// Appends "search", "result name" and criteria
// e.g. search count > 0
func buildAlertCondition(resultField string, criteria string) string {
	return "search " + resultField + " " + criteria
}

// Builds the name of the alert by appending names of project, stage, service, sli and criteria.
// Appends "keptn" as a suffix in order to identu=ify it as an alert for keptn
func buildAlertName(eventData keptnv2.ConfigureMonitoringTriggeredEventData, stage string, sli string, criteria string) string {
	return eventData.Project + "," + stage + "," + eventData.Service + "," + sli + "," + criteria + "," + KeptnSuffix
}

// check if the configure monitoring triggered event is not for splunk service
func isNotForSplunk(sliProvider string) bool {
	return sliProvider != "splunk"
}

// check if the project and/or the service have not been set in the configure monitoring triggered event
func isProjectOrServiceNotSet(data *keptnv2.ConfigureMonitoringTriggeredEventData) bool {
	return data.Project == "" || data.Service == ""
}
