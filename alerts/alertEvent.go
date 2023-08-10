package alerts

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	splunkalerts "github.com/keptn-sandbox/splunk-sli-provider/pkg/splunksdk/alerts"
	splunk "github.com/keptn-sandbox/splunk-sli-provider/pkg/splunksdk/client"
	"github.com/keptn-sandbox/splunk-sli-provider/pkg/utils"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	api "github.com/keptn/go-utils/pkg/api/utils"
	keptncommons "github.com/keptn/go-utils/pkg/lib"
	"github.com/keptn/go-utils/pkg/lib/keptn"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
	"github.com/keptn/go-utils/pkg/lib/v0_2_0/fake"
)

const (
	remediationTaskName = "remediation"
	pollingFrequency    = 20      //indicates the frequency at which triggered alerts are checked in seconds
	keptnSuffix         = "keptn" //Added at the end of each splunk alert created using configure monitoring
	serviceName         = "splunk-sli-provider"
)

type SplunkAlertEvent struct {
	Sid         string      `json:"sid"`
	SearchName  string      `json:"search_name"`
	App         string      `json:"app"`
	Owner       string      `json:"owner"`
	ResultsLink string      `json:"results_link"`
	Result      alertResult `json:"result"`
}

// alert coming from splunk
type alertResult struct {
	Avg           string `json:"avg"`
	Count         string `json:"count"`
	DistinctCount string `json:"distinct_count"`
	Estdc         string `json:"estdc"`
	EstdcError    string `json:"estdc_error"`
	Exactperc     string `json:"exactperc"`
	Max           string `json:"max"`
	Mean          string `json:"mean"`
	Median        string `json:"median"`
	Min           string `json:"min"`
	Mode          string `json:"mode"`
	Perc          string `json:"perc"`
	Range         string `json:"range"`
	Stdev         string `json:"stdev"`
	Stdevp        string `json:"stdevp"`
	Sum           string `json:"sum"`
	Sumsq         string `json:"sumsq"`
	Upperperc     string `json:"upperperc"`
	Var           string `json:"var"`
	Varp          string `json:"varp"`
}

// type labels struct {
// 	AlertName  string `json:"alertname,omitempty"`
// 	Namespace  string `json:"namespace,omitempty"`
// 	PodName    string `json:"pod_name,omitempty"`
// 	Severity   string `json:"severity,omitempty"`
// 	Service    string `json:"service,omitempty" yaml:"service"`
// 	Stage      string `json:"stage,omitempty" yaml:"stage"`
// 	Project    string `json:"project,omitempty" yaml:"project"`
// 	Deployment string `json:"deployment,omitempty" yaml:"deployment"`
// }

type RemediationTriggeredEventData struct {
	keptnv2.EventData
	// Problem contains details about the problem
	Problem keptncommons.ProblemEventData `json:"problem"`
	// Deployment contains the current deployment, that is inferred from the alert details
	Deployment keptnv2.DeploymentFinishedData `json:"deployment"`
}

// ProcessAndForwardAlertEvent reads the payload from the request and sends a valid Cloud event to the keptn event broker
func ProcessAndForwardAlertEvent(triggeredInstance splunkalerts.EntryItem, logger *keptn.Logger, client *splunk.SplunkClient, ddKeptn *keptnv2.Keptn, keptnOptions keptn.KeptnOpts, envConfig utils.EnvConfig) error {

	logger.Info("New alert found in Splunk Alerting system : " + triggeredInstance.Name)

	const deploymentType = "primary"
	alertDetails := strings.Split(triggeredInstance.Content.SavedSearchName, ",")
	shkeptncontext := ""

	problemData := keptncommons.ProblemEventData{
		State:          "OPEN",
		ProblemID:      "",
		ProblemTitle:   alertDetails[3], //name of sli
		ProblemDetails: json.RawMessage(`{}`),
		ProblemURL:     net.JoinHostPort(client.Host, client.Port) + triggeredInstance.Links.Job + "/results",
		ImpactedEntity: fmt.Sprintf("%s-%s", alertDetails[2], deploymentType),
		Project:        alertDetails[0],
		Stage:          alertDetails[1],
		Service:        alertDetails[2],
		Labels: map[string]string{
			"deployment": deploymentType,
		},
	}

	newEventData := RemediationTriggeredEventData{
		EventData: keptnv2.EventData{
			Project: alertDetails[0],
			Stage:   alertDetails[1],
			Service: alertDetails[2],
			Labels: map[string]string{
				"Problem URL": net.JoinHostPort(client.Host, client.Port) + triggeredInstance.Links.Job + "/results",
			},
		},
		Problem: problemData,
		Deployment: keptnv2.DeploymentFinishedData{
			DeploymentNames: []string{
				deploymentType,
			},
		},
	}

	switch triggeredInstance.Content.Sid {
	case "":
		logger.Debug("NO SHKEPTNCONTEXT SET")
	default:
		shkeptncontext = createOrApplyKeptnContext(triggeredInstance.Content.Sid + time.Now().Format(time.UnixDate))
		logger.Debug("shkeptncontext=" + shkeptncontext)
	}

	logger.Debug("Sending event to eventbroker")
	err := createAndSendCE(newEventData, shkeptncontext, ddKeptn, keptnOptions, envConfig)

	return err

}

// createAndSendCE create a new problem.triggered event and send it to Keptn
func createAndSendCE(problemData RemediationTriggeredEventData, shkeptncontext string, ddKeptn *keptnv2.Keptn, keptnOptions keptn.KeptnOpts, envConfig utils.EnvConfig) error {
	source, _ := url.Parse("splunk")

	eventType := keptnv2.GetTriggeredEventType(problemData.Stage + "." + remediationTaskName)

	event := cloudevents.NewEvent()
	event.SetID(uuid.New().String())
	event.SetTime(time.Now())
	event.SetType(eventType)
	event.SetSource(source.String())
	event.SetDataContentType(cloudevents.ApplicationJSON)
	event.SetExtension("shkeptncontext", shkeptncontext)
	err := event.SetData(cloudevents.ApplicationJSON, problemData)
	if err != nil {
		return fmt.Errorf("unable to set cloud event data: %w", err)
	}

	if ddKeptn == nil {
		ddKeptn, err = keptnv2.NewKeptn(&event, keptnOptions)

		//Setting authentication header when accessing to keptn locally in order to be able to access to the resource-service
		if envConfig.Env == "local" {
			authToken := os.Getenv("KEPTN_API_TOKEN")
			authHeader := "x-token"
			ddKeptn.ResourceHandler = api.NewAuthenticatedResourceHandler(ddKeptn.ResourceHandler.BaseURL, authToken, authHeader, ddKeptn.ResourceHandler.HTTPClient, ddKeptn.ResourceHandler.Scheme)
		}

		if err != nil {
			return fmt.Errorf("Could not create Keptn Handler: %w", err)
		}
	}

	err = ddKeptn.SendCloudEvent(event)
	if err != nil {
		return err
	}

	return nil
}

// createOrApplyKeptnContext re-uses the existing Keptn Context or creates a new one based on the splunk alert id and the current time
func createOrApplyKeptnContext(contextID string) string {
	uuid.SetRand(nil)
	keptnContext := uuid.New().String()
	if contextID != "" {
		_, err := uuid.Parse(contextID)
		switch err {
		case nil:
			keptnContext = contextID
		default:
			switch {
			case len(contextID) < 16:
				// use provided contxtId as a seed
				paddedContext := fmt.Sprintf("%-16v", contextID)
				uuid.SetRand(strings.NewReader(paddedContext))
			default:
				// convert hash of contextID
				h := sha256.New()
				h.Write([]byte(contextID))
				bs := h.Sum(nil)

				uuid.SetRand(strings.NewReader(string(bs)))
			}

			keptnContext = uuid.New().String()
			uuid.SetRand(nil)
		}
	}
	return keptnContext
}

func isTestKeptn(i interface{}) bool {
	switch i.(type) {
	case *fake.EventSender:
		return true
	default:
		return false
	}
}

// FiringAlertsPoll will handle all requests for '/health' and '/ready'
func FiringAlertsPoll(client *splunk.SplunkClient, ddKeptn *keptnv2.Keptn, keptnOptions keptn.KeptnOpts, envConfig utils.EnvConfig) {

	shkeptncontext := uuid.New().String()
	logger := keptn.NewLogger(shkeptncontext, "", serviceName)

	for {

		//listing fired alerts
		logger.Info("Searching for triggered alerts ...")
		triggeredAlerts, err := splunkalerts.GetTriggeredAlerts(client)
		if err != nil {
			logger.Errorf("Error calling GetTriggeredAlerts() while searchcing for new alerts: %v : %v", triggeredAlerts, err)
		}

		for _, triggeredAlert := range triggeredAlerts.Entry {

			if strings.HasSuffix(triggeredAlert.Name, keptnSuffix) {

				triggeredInstances, err := splunkalerts.GetInstancesOfTriggeredAlert(client, triggeredAlert.Links.List)
				if err != nil {
					logger.Errorf("Error calling GetInstancesOfTriggeredAlert(): %v : %v", triggeredInstances, err)
				}

				for _, triggeredInstance := range triggeredInstances.Entry {
					if triggeredInstance.Content.TriggerTime <= int(time.Now().Unix()) && triggeredInstance.Content.TriggerTime > int(time.Now().Unix())-pollingFrequency-2 {
						err = ProcessAndForwardAlertEvent(triggeredInstance, logger, client, ddKeptn, keptnOptions, envConfig)
						switch err {
						case nil:
							logger.Debug("Event successfully dispatched to eventbroker")

						default:
							logger.Errorf("Could not Process and Forward cloud event: %v", err)
						}
					}
				}

			}

		}
		// Condition only verified in case of a test
		if ddKeptn != nil && isTestKeptn(ddKeptn.EventSender) {
			return
		}
		time.Sleep(pollingFrequency * time.Second)
	}
}
