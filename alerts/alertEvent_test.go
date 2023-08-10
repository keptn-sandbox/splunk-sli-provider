package alerts

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	splunktest "github.com/keptn-sandbox/splunk-sli-provider/pkg/splunksdk/pkg/utils"
	"github.com/keptn-sandbox/splunk-sli-provider/pkg/utils"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/event/datacodec"
	"github.com/google/uuid"
	"github.com/keptn/go-utils/pkg/lib/keptn"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
	"github.com/keptn/go-utils/pkg/lib/v0_2_0/fake"
)

const (
	alertNamesFilePath         = "../test/data/unitTests/firedAlerts.json"
	firedAlertInstanceFilePath = "../test/data/unitTests/firedAlertInstances.json"
	stage                      = "production"
	project                    = "fulltour2"
	service                    = "helloservice"
	state                      = "OPEN"
	problemTitle               = "number_of_logs"
)

/**
 * loads a cloud event from the passed test json file and initializes a keptn object with it
 */
func initializeObjects() (*keptnv2.Keptn, error) {
	// load sample event

	source, _ := url.Parse("splunk")

	eventType := keptnv2.GetTriggeredEventType(stage + "." + remediationTaskName)

	event := cloudevents.NewEvent()
	event.SetID(uuid.New().String())
	event.SetTime(time.Now())
	event.SetType(eventType)
	event.SetSource(source.String())
	event.SetDataContentType(cloudevents.ApplicationJSON)
	shkeptncontext := createOrApplyKeptnContext("Sid" + time.Now().Format(time.UnixDate))
	event.SetExtension("shkeptncontext", shkeptncontext)

	var keptnOptions = keptn.KeptnOpts{
		EventSender: &fake.EventSender{},
	}

	ddKeptn, err := keptnv2.NewKeptn(&event, keptnOptions)

	return ddKeptn, err
}

func TestFiringAlertsPoll(t *testing.T) {

	//Building a mock splunk server
	splunkServer := buildMockAlertSplunkServer(t)
	defer splunkServer.Close()

	//setting splunk credentials
	env := utils.EnvConfig{}

	env.SplunkPort = strings.Split(splunkServer.URL, ":")[2]
	env.SplunkHost = strings.Split(strings.Split(splunkServer.URL, ":")[1], "//")[1]
	env.SplunkApiToken = "apiToken"

	ddKeptn, err := initializeObjects()
	if err != nil {
		t.Fatal(err)
	}
	splunkCreds, err := utils.GetSplunkCredentials(env)
	if err != nil {
		t.Fatalf("failed to get Splunk Credentials: %v", err)
	}
	client := utils.ConnectToSplunk(*splunkCreds, true)

	ddKeptn.UseLocalFileSystem = false
	FiringAlertsPoll(client, ddKeptn, keptn.KeptnOpts{}, env)

	gotEvents := len(ddKeptn.EventSender.(*fake.EventSender).SentEvents)

	// Verify that HandleGetSliTriggeredEvent has sent a cloudevent
	if gotEvents != 1 {
		t.Fatalf("Expected one event to be sent, but got %v", gotEvents)
	}

	// Verify that the CE sent is a <stage>.remediation.triggered event
	if keptnv2.GetTriggeredEventType(stage+"."+remediationTaskName) != ddKeptn.EventSender.(*fake.EventSender).SentEvents[0].Type() {
		t.Fatal("Expected a " + stage + "." + remediationTaskName + " event type but got " + ddKeptn.EventSender.(*fake.EventSender).SentEvents[0].Type())
	}

	var respData RemediationTriggeredEventData
	err = datacodec.Decode(context.Background(), ddKeptn.EventSender.(*fake.EventSender).SentEvents[0].DataMediaType(), ddKeptn.EventSender.(*fake.EventSender).SentEvents[0].Data(), &respData)

	if err != nil {
		t.Fatal("Error decoding the data of the remediation.triggered event sent")
	}
	// Verify that the correct data is included in the remediation.triggered event sent
	if respData.Project != project || respData.Service != service || respData.Stage != stage || respData.Problem.State != state || respData.Problem.ProblemTitle != problemTitle {
		t.Fatal("The data (project, stage, service, problem state or problem title) sent for the remediation.triggered event is incorrect")
	}

}

/**
 * loads from files the default responses we want the fake splunk server to send
 */
func initializeResponses(fileName string) (string, error) {

	// load a json http response for requests to get fired alerts
	file, err := os.ReadFile(fileName)
	if err != nil {
		return "", fmt.Errorf("Can't load %s: %w", fileName, err)
	}
	getResponse := string(file)

	return getResponse, nil
}

// Builds a fake splunk server able to respond when we try to list fired alerts and instances of fired alerts
func buildMockAlertSplunkServer(t *testing.T) *httptest.Server {

	//getting the default splunk responses for listing fired alerts and instances of a fired alert
	getFiredAlertsResponse, err := initializeResponses(alertNamesFilePath)
	if err != nil {
		t.Fatal("Error initializing default responses for the mock splunk server.")
	}
	getFiredAlertInstancesResponse, err := initializeResponses(firedAlertInstanceFilePath)
	if err != nil {
		t.Fatal("Error initializing default responses for the mock splunk server.")
	}

	//Customizing the fired alert and the instances of the fired alert for the responses we want to send when the fake splunk server receive a request
	getFiredAlertInstancesResponse = strings.Replace(getFiredAlertInstancesResponse, "1689080402", fmt.Sprint(time.Now().Unix()), -1)
	getFiredAlertsResponse = strings.Replace(getFiredAlertsResponse, "production", stage, -1)
	getFiredAlertInstancesResponse = strings.Replace(getFiredAlertInstancesResponse, "production", stage, -1)
	getFiredAlertsResponse = strings.Replace(getFiredAlertsResponse, "fulltour2", project, -1)
	getFiredAlertInstancesResponse = strings.Replace(getFiredAlertInstancesResponse, "fulltour2", project, -1)
	getFiredAlertsResponse = strings.Replace(getFiredAlertsResponse, "helloservice", service, -1)
	getFiredAlertInstancesResponse = strings.Replace(getFiredAlertInstancesResponse, "helloservice", service, -1)
	getFiredAlertsResponse = strings.Replace(getFiredAlertsResponse, "number_of_logs", problemTitle, -1)
	getFiredAlertInstancesResponse = strings.Replace(getFiredAlertInstancesResponse, "number_of_logs", problemTitle, -1)

	jsonResponsePOST := `{
		"sid": "10"
	}`
	jsonResponseGET := `{
		"results":[{"theRequest":"` + fmt.Sprint(200) + `"}]
	}`

	splunkResponses := make([]map[string]interface{}, 2)
	splunkResponses[0] = map[string]interface{}{
		"getTriggeredAlerts":    getFiredAlertsResponse,
		"getTriggeredInstances": getFiredAlertInstancesResponse,
		http.MethodPost:         jsonResponsePOST,
		http.MethodGet:          jsonResponseGET,
	}
	splunkServer := splunktest.MultitpleMockRequest(splunkResponses, true)

	return splunkServer
}
