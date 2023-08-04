package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	splunk "github.com/AmadeusITGroup/keptn-splunk-sli-provider/pkg/splunksdk/client"
	splunktest "github.com/AmadeusITGroup/keptn-splunk-sli-provider/pkg/splunksdk/pkg/utils"
	"github.com/AmadeusITGroup/keptn-splunk-sli-provider/pkg/utils"

	cloudevents "github.com/cloudevents/sdk-go/v2" // make sure to use v2 cloudevents here
	"github.com/cloudevents/sdk-go/v2/event/datacodec"
	keptn "github.com/keptn/go-utils/pkg/lib/keptn"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
	"github.com/keptn/go-utils/pkg/lib/v0_2_0/fake"
)

// You can configure your tests by specifying the path to get-sli triggered event file in json,
// the path to your sli.yaml file
// and the default result you await from splunk
// Indicators given in get-sli.triggered.json should match indicators in the given sli file
const (
	getSliTriggeredEventFile              = "../test/events/get-sli.triggered.json"
	configureMonitoringTriggeredEventFile = "../test/events/monitoring.configure.json"
	sliFilePath                           = "../test/data/podtatohead.sli.yaml"
	alertNamesFilePath                    = "../test/data/unitTests/firedAlerts.json"
	defaultSplunkTestResult               = 1250
	stage                                 = "production"
	project                               = "fulltour2"
	service                               = "helloservice"
	state                                 = "OPEN"
	problemTitle                          = "number_of_logs"
)

func initializeResponses(fileName string) (string, error) {

	// load a json http response for requests to get fired alerts
	file, err := os.ReadFile(fileName)
	if err != nil {
		return "", fmt.Errorf("Can't load %s: %w", fileName, err)
	}
	getResponse := string(file)

	return getResponse, nil
}

/**
 * loads a cloud event from the passed test json file and initializes a keptn object with it
 */
func initializeTestObjects(eventFileName string, resourceServiceUrl string) (*keptnv2.Keptn, *cloudevents.Event, error) {
	// load sample event
	eventFile, err := os.ReadFile(eventFileName)
	if err != nil {
		return nil, nil, fmt.Errorf("Can't load %s: %w", eventFileName, err)
	}

	incomingEvent := &cloudevents.Event{}
	err = json.Unmarshal(eventFile, incomingEvent)
	if err != nil {
		return nil, nil, fmt.Errorf("Error parsing: %w", err)
	}

	// Add a Fake EventSender to KeptnOptions
	var keptnOptions = keptn.KeptnOpts{
		EventSender: &fake.EventSender{},
	}
	keptnOptions.ConfigurationServiceURL = resourceServiceUrl
	keptnOptions.UseLocalFileSystem = true

	ddKeptn, err := keptnv2.NewKeptn(incomingEvent, keptnOptions)

	return ddKeptn, incomingEvent, err
}

// Tests the HandleSpecificSli function
func TestHandleSpecificSli(t *testing.T) {
	indicatorName := "test"
	data := &keptnv2.GetSLITriggeredEventData{}
	sliConfig := make(map[string]string, 1)
	sliConfig[indicatorName] = "test"

	//Building a mock splunk server returning default responses when getting  get and post requests

	splunkServer := utils.BuildMockSplunkServer(defaultSplunkTestResult)
	defer splunkServer.Close()

	//Retrieving the mock splunk server credentials
	splunkCreds := &utils.SplunkCredentials{
		Host:  strings.Split(strings.Split(splunkServer.URL, ":")[1], "//")[1],
		Port:  strings.Split(splunkServer.URL, ":")[2],
		Token: "apiToken",
	}

	client := splunk.NewClientAuthenticatedByToken(
		&http.Client{
			Timeout: time.Duration(60) * time.Second,
		},
		splunkCreds.Host,
		splunkCreds.Port,
		splunkCreds.Token,
		true,
	)
	sliResult, errored := handleSpecificSLI(client, indicatorName, data, sliConfig)

	if errored != nil {
		t.Fatal(errored.Error())
	}
	t.Logf("SLI Result : %v", sliResult.Value)
	if sliResult.Value != float64(defaultSplunkTestResult) {
		t.Fatalf("Wrong value for the metric %s : expected %v, got %v", indicatorName, defaultSplunkTestResult, sliResult.Value)
	}
}

// Tests the handleGetSliTriggered function
// Tests the handleGetSliTriggered function
func TestHandleGetSliTriggered(t *testing.T) {

	//Building a mock resource service server
	resourceServiceServer, err := buildMockResourceServiceServer(sliFilePath, shipyardFilePath, sloFilePath, remediationFilePath)
	if err != nil {
		t.Fatalf("Error reading sli file : %v", err)
	}
	defer resourceServiceServer.Close()

	//Building a mock splunk server
	splunkServer := buildMockSplunkServer(t)
	defer splunkServer.Close()

	//setting splunk credentials
	env := utils.EnvConfig{}
	env.SplunkPort = strings.Split(splunkServer.URL, ":")[2]
	env.SplunkHost = strings.Split(strings.Split(splunkServer.URL, ":")[1], "//")[1]
	env.SplunkApiToken = "apiToken"

	//Initializing test objects
	ddKeptn, incomingEvent, err := initializeTestObjects(getSliTriggeredEventFile, resourceServiceServer.URL+"/api/resource-service")
	if err != nil {
		t.Fatal(err)
	}

	data := &keptnv2.GetSLITriggeredEventData{}
	err = incomingEvent.DataAs(data)

	if err != nil {
		t.Fatalf("Error while getting keptn event data : %v", err)
	}

	// create splunk credentials
	splunkCreds, err := utils.GetSplunkCredentials(env)

	if err != nil {
		t.Fatalf("Failed to get splunk credentials: %s", err)
		return
	}
	client := utils.ConnectToSplunk(*splunkCreds, true)
	err = HandleGetSliTriggeredEvent(ddKeptn, *incomingEvent, data, client)

	if err != nil {
		t.Fatalf("Error : %v", err)
	}

	gotEvents := len(ddKeptn.EventSender.(*fake.EventSender).SentEvents)

	// Verify that HandleGetSliTriggeredEvent has sent 2 cloudevents
	if gotEvents != 2 {
		t.Fatalf("Expected two events to be sent, but got %v", gotEvents)
	}

	// Verify that the first CE sent is a .started event
	if keptnv2.GetStartedEventType(keptnv2.GetSLITaskName) != ddKeptn.EventSender.(*fake.EventSender).SentEvents[0].Type() {
		t.Fatal("Expected a get-sli.started event type")
	}

	// Verify that the second CE sent is a .finished event
	if keptnv2.GetFinishedEventType(keptnv2.GetSLITaskName) != ddKeptn.EventSender.(*fake.EventSender).SentEvents[1].Type() {
		t.Fatal("Expected a get-sli.finished event type")
	}

	// Verify thet the .finished event contains the sli results
	finishedEvent := ddKeptn.EventSender.(*fake.EventSender).SentEvents[1]
	var respData keptnv2.GetSLIFinishedEventData
	err = datacodec.Decode(context.Background(), finishedEvent.DataMediaType(), finishedEvent.Data(), &respData)
	if err != nil {
		t.Fatalf("Unable to decode data from the event : %v", err)
	}
	// print respData
	switch indicValues := respData.GetSLI.IndicatorValues; indicValues {
	case nil:
		t.Fatal("No results added into the response event for the indicators.")
	default:
		//printing SLI results if no error has occurred
		for _, sliResult := range indicValues {
			switch sliValue := sliResult.Value; sliValue {
			case float64(defaultSplunkTestResult):
				t.Logf("SLI Results for indicator %s : %v", sliResult.Metric, sliResult.Value)
			default:
				t.Fatalf("Wrong value for the metric %s : %v", sliResult.Metric, sliResult.Value)
			}
		}
	}
}

// Builds a fake splunk server able to respond when we try to list fired alerts and instances of fired alerts
func buildMockSplunkServer(t *testing.T) *httptest.Server {

	//getting the default splunk responses for listing fired alerts and instances of a fired alert
	getAlertsNamesResponse, err := initializeResponses(alertNamesFilePath)
	if err != nil {
		t.Fatal("Error initialising default responses for the mock splunk server.")
	}

	jsonResponsePOST := `{
		"sid": "10"
	}`
	jsonResponseGET := `{
		"results":[{"theRequest":"` + fmt.Sprint(defaultSplunkTestResult) + `"}]
	}`

	splunkResponses := make([]map[string]interface{}, 2)
	splunkResponses[0] = map[string]interface{}{
		"getAlertsNames": getAlertsNamesResponse,
		http.MethodPost:  jsonResponsePOST,
		http.MethodGet:   jsonResponseGET,
	}
	splunkServer := splunktest.MultitpleMockRequest(splunkResponses, true)

	return splunkServer
}

// Build a mock resource service server returning a response with the content of the sli file
func buildMockResourceServiceServer(sliFilePath string, shipyardFilePath string, sloFilePath string, remediationFilePath string) (*httptest.Server, error) {

	var getResponses []string
	var postResponses []string
	var paths []string

	err := updateGetResponses(&getResponses, &paths, sliFilePath, sliFileUri)
	if err != nil {
		return nil, err
	}
	err = updateGetResponses(&getResponses, &paths, shipyardFilePath, shipyardUri)
	if err != nil {
		return nil, err
	}
	err = updateGetResponses(&getResponses, &paths, sloFilePath, sloUri)
	if err != nil {
		return nil, err
	}
	err = updateGetResponses(&getResponses, &paths, remediationFilePath, remediationUri)
	if err != nil {
		return nil, err
	}

	resourceServiceServer := utils.MultitpleMockRequest(getResponses, postResponses, paths, false)

	return resourceServiceServer, nil
}

func updateGetResponses(getResponses *[]string, paths *[]string, filePath string, fileUri string) error {

	if filePath != "" {
		fileContent, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}

		*getResponses = append(*getResponses, `{
			"resourceContent": "`+base64.StdEncoding.EncodeToString(fileContent)+`",
			"resourceURI":"`+fileUri+`",
			"metadata": {
			  "upstreamURL": "https://github.com/user/keptn.git",
			  "version": "1.0.0"
			}
		  }`)

		*paths = append(*paths, fileUri)
	}

	return nil
}
