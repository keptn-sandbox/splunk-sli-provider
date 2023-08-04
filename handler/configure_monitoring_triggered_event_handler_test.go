package handler

import (
	"strings"
	"testing"

	"github.com/AmadeusITGroup/keptn-splunk-sli-provider/pkg/splunksdk/alerts"
	splunk "github.com/AmadeusITGroup/keptn-splunk-sli-provider/pkg/splunksdk/client"
	"github.com/AmadeusITGroup/keptn-splunk-sli-provider/pkg/utils"

	keptnv1 "github.com/keptn/go-utils/pkg/lib"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
	"github.com/keptn/go-utils/pkg/lib/v0_2_0/fake"
)

const (
	sloFilePath         = "../test/data/podtatohead.slo.yaml"
	shipyardFilePath    = "../test/data/unitTests/shipyard.yaml"
	remediationFilePath = "../test/data/unitTests/remediation.yaml"
	shipyardUri         = "shipyard.yaml"
	sloUri              = "slo.yaml"
	remediationUri      = "remediation.yaml"
	sli                 = "number_of_errors"
	criteria            = ">=100"
)

func TestHandleConfigureMonitoringTriggeredEvent(t *testing.T) {

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
	ddKeptn, incomingEvent, err := initializeTestObjects(configureMonitoringTriggeredEventFile, resourceServiceServer.URL+"/api/resource-service")
	if err != nil {
		t.Fatal(err)
	}

	if incomingEvent.Type() == keptnv1.ConfigureMonitoringEventType {
		incomingEvent.SetType(keptnv2.GetTriggeredEventType(keptnv2.ConfigureMonitoringTaskName))
	}

	data := &keptnv2.ConfigureMonitoringTriggeredEventData{}
	err = incomingEvent.DataAs(data)

	if err != nil {
		t.Fatal("Error getting keptn event data")
	}

	var alertCreated bool

	createAlert = func(client *splunk.SplunkClient, spAlert *alerts.AlertRequest) error {

		if spAlert.Params.Name == data.Project+","+stage+","+data.Service+","+sli+","+criteria+","+KeptnSuffix &&
			spAlert.Params.SearchQuery == `source="http:podtato-error" (index="keptn-splunk-dev") "[error]" | stats count` &&
			spAlert.Params.AlertCondition == "search count "+criteria {
			alertCreated = true
		}

		return nil
	}
	// create splunk credentials
	splunkCreds, err := utils.GetSplunkCredentials(env)

	if err != nil {
		t.Fatalf("Failed to get splunk credentials: %s", err)
		return
	}
	client := utils.ConnectToSplunk(*splunkCreds, true)
	data.ConfigureMonitoring.Type = "splunk"
	err = HandleConfigureMonitoringTriggeredEvent(ddKeptn, *incomingEvent, data, env, client, false)

	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	gotEvents := len(ddKeptn.EventSender.(*fake.EventSender).SentEvents)

	// Verify that HandleGetSliTriggeredEvent has sent 2 cloudevents
	if gotEvents != 2 {
		t.Fatalf("Expected two events to be sent, but got %v", gotEvents)
	}

	// Verify that the first CE sent is a .started event
	if keptnv2.GetStartedEventType(keptnv2.ConfigureMonitoringTaskName) != ddKeptn.EventSender.(*fake.EventSender).SentEvents[0].Type() {
		t.Fatal("Expected a configure-monitoring.started event type")
	}

	// Verify that the second CE sent is a .finished event
	if keptnv2.GetFinishedEventType(keptnv2.ConfigureMonitoringTaskName) != ddKeptn.EventSender.(*fake.EventSender).SentEvents[1].Type() {
		t.Fatal("Expected a configure-monitoring.finished event type")
	}

	// Verify if createAlert has been called (We have one stage, one objective and one criteria so only one alert should be created)
	if alertCreated == false {
		t.Fatal("No alert has been created")
	}
}
