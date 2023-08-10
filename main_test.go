package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	splunk "github.com/keptn-sandbox/splunk-sli-provider/pkg/splunksdk/client"
	"github.com/keptn-sandbox/splunk-sli-provider/pkg/utils"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/kelseyhightower/envconfig"
	keptnv1 "github.com/keptn/go-utils/pkg/lib"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
	logger "github.com/sirupsen/logrus"
)

var testPortforMain = 38888

// Tests the parseKeptnCloudEventPayload function
func TestParseKeptnCloudEventPayload(t *testing.T) {
	incomingEvent, err := extractEvent("test/events/get-sli.triggered.json")
	if err != nil {
		t.Fatalf("Error getting keptn event : %v", err)
	}
	eventData := &keptnv2.GetSLITriggeredEventData{}
	err = parseKeptnCloudEventPayload(*incomingEvent, eventData)

	//fails if eventData has not been modified
	if err != nil || eventData.Project == "" {
		t.Fatalf("Failed to parse keptn cloud event payload")
	}
	t.Logf("%v", eventData)
}

// Tests the processKeptnCloudEvent function by checking if it processes get-sli.triggered and configure monitoring events
func TestProcessKeptnCloudEvent(t *testing.T) {

	t.Log("Initializing get sli triggered event")
	var calledSLI *bool = new(bool)
	var calledConfig *bool = new(bool)

	*calledSLI = false
	*calledConfig = false

	handleConfigureMonitoringTriggeredEvent = func(ddKeptn *keptnv2.Keptn, incomingEvent event.Event, data *keptnv2.ConfigureMonitoringTriggeredEventData, env utils.EnvConfig, client *splunk.SplunkClient, pollingSystemHasBeenStarted bool) error {
		*calledConfig = true
		return nil
	}
	handleGetSliTriggeredEvent = func(ddKeptn *keptnv2.Keptn, incomingEvent event.Event, data *keptnv2.GetSLITriggeredEventData, client *splunk.SplunkClient) error {
		*calledSLI = true
		return nil
	}

	//Test for a get-sli.triggered event
	checkProcessKeptnCloudEvent(t, "test/events/get-sli.triggered.json", calledSLI, calledConfig)

	//Test for a monitoring.configure event
	checkProcessKeptnCloudEvent(t, "test/events/monitoring.configure.json", calledSLI, calledConfig)

	//Test for a random event
	checkProcessKeptnCloudEvent(t, "test/events/release.triggered.json", calledSLI, calledConfig)
}

// Tests the _main function by ensuring that it listens to cloudevents and trigger the procKeptnCE function
func TestCloudEventListener(t *testing.T) {

	if err := envconfig.Process("", &env); err != nil {
		logger.Fatalf("Failed to process env var: %s", err)
	}
	env.Port = testPortforMain
	env.Env = "test"

	var handled bool
	processKeptnCloudEvent = func(ctx context.Context, event cloudevents.Event) error {
		handled = true
		return nil
	}

	args := []string{}
	go CloudEventListener(args)
	//sleep for 2 seconds to let the previous go routine the time to start listening for events
	time.Sleep(time.Duration(2) * time.Second)
	err := sendTestCloudEvent("test/events/get-sli.triggered.json")
	if err != nil {
		logger.Fatalf("Couldn't send cloud event : %v", err)
	}

	if handled == false {
		t.Fatal("The function didn't handle the event.")
	}
}

// Tests the main function by verifying if the exit code corresponds to the one returned by cloudEventListener function
// func TestMain(t *testing.T) {
// 	const expectedReturn = 15

// 	cloudEventListener = func(args []string) int {
// 		return expectedReturn
// 	}
// 	if os.Getenv("BE_MAIN") == "1" {
// 		main()
// 		return
// 	}
// 	cmd := exec.Command(os.Args[0], "-test.run=TestMain")
// 	cmd.Env = append(os.Environ(), "BE_MAIN=1")
// 	err := cmd.Run()

// 	if e, ok := err.(*exec.ExitError); ok && e.ExitCode() == expectedReturn {
// 		return
// 	}
// 	t.Fatalf("process ran with err %v, want exit status %v", expectedReturn, err)
// }

// reads the json event file and convert its content into an event
func extractEvent(eventFileName string) (*event.Event, error) {

	eventFile, err := os.ReadFile(eventFileName)
	if err != nil {
		return nil, err
	}

	incomingEvent := &cloudevents.Event{}
	err = json.Unmarshal(eventFile, incomingEvent)
	if err != nil {
		return nil, err
	}

	return incomingEvent, err
}

// Check if events are handled (not handled) when they should be (shouldn't be)
func checkProcessKeptnCloudEvent(t *testing.T, fileName string, calledSLI *bool, calledConfig *bool) {
	configureMonitoringTriggeredEventv1 := "sh.keptn.event.monitoring.configure.triggered"
	configureMonitoringTriggeredEvent := "sh.keptn.event.configure-monitoring.triggered"

	incomingEvent, err := extractEvent(fileName)
	if err != nil {
		t.Fatalf("Error getting keptn event : %v", err)
	}
	err = processKeptnCloudEvent(context.Background(), *incomingEvent)

	if err != nil {
		//verify if events that should be handled are not skipped
		if strings.HasPrefix(err.Error(), UnhandleKeptnCloudEvent) &&
			(incomingEvent.Type() == "sh.keptn.event.configure-monitoring" ||
				incomingEvent.Type() == keptnv2.GetTriggeredEventType(keptnv2.GetSLITaskName) ||
				incomingEvent.Type() == configureMonitoringTriggeredEventv1 ||
				incomingEvent.Type() == keptnv1.ConfigureMonitoringEventType) {
			t.Fatal("The function didn't handle an event that should have been handled.")
		}
		return
	}
	//verify if events that should be handled are handled correctly
	switch incomingEvent.Type() {
	case configureMonitoringTriggeredEvent, keptnv2.ConfigureMonitoringTaskName:
		if *calledConfig == false {
			t.Fatal("The configure monitoring event has not been handled.")

		}
	case keptnv2.GetTriggeredEventType(keptnv2.GetSLITaskName):
		if *calledSLI == false {
			t.Fatal("The get-sli triggered event has not been handled.")

		}
	case keptnv1.ConfigureMonitoringEventType:
		t.Fatal("keptnv1 configure monitoring event must be converted into keptnv2 configure monitoring event.")
	}
}

// Sends a cloud event
func sendTestCloudEvent(eventFileName string) error {
	body, err := os.ReadFile(eventFileName)
	if err != nil {
		return fmt.Errorf("Can't load %s: %w", eventFileName, err)
	}

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, "http://localhost:"+fmt.Sprint(testPortforMain), bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("Error : %w\n", err)
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Cache-Control", "no-cache")
	req.Header.Add("Content-Type", "application/cloudevents+json")

	res, err := client.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	_, err = io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	return nil
}
