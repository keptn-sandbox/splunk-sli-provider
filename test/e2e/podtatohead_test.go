package e2e

import (
	"encoding/base64"
	"os"
	"testing"
	"time"

	"github.com/keptn/go-utils/pkg/api/models"
	"github.com/stretchr/testify/require"
)

const (
	podtatoDeployV1Event            = "../events/podtatohead.deploy-v0.1.1.triggered.json"
	podtatoDeployV2Event            = "../events/podtatohead.deploy-v0.1.2.triggered.json"
	podtatoConfigureMonitoringEvent = "../events/podtatohead.configure-monitoring.json"
	podtatoShipyardFile             = "../data/podtatohead.shipyard.yaml"
	podtatoJobConfigFile            = "../data/podtatohead.job-config.yaml"
	podtatoServiceChart             = "../data/podtatoservice.tgz"
	podtatoSliFile                  = "../data/podtatohead.sli.yaml"
	podtatoSloFile                  = "../data/podtatohead.slo.yaml"
	splunkHecScript                 = "../data/splunk-hec.py"
	splunkLogFile                   = "../data/splunk-log.txt"
)

func TestPodtatoheadEvaluation(t *testing.T) {
	if !isE2ETestingAllowed() {
		t.Skip("Skipping TestHelloWorldDeployment, not allowed by environment")
	}

	// Setup the E2E test environment
	testEnv, err := newTestEnvironment(
		podtatoDeployV1Event,
		podtatoShipyardFile,
		podtatoJobConfigFile,
	)
	require.NoError(t, err)

	err = testEnv.API.DeleteProject(testEnv.EventData.Project)
	require.NoError(t, err)

	additionalResources := []struct {
		FilePath     string
		ResourceName string
	}{
		{FilePath: podtatoServiceChart, ResourceName: "charts/podtatoservice.tgz"},
		{FilePath: splunkHecScript, ResourceName: "scripts/splunk-hec.py"},
		{FilePath: splunkLogFile, ResourceName: "scripts/splunk-log.txt"},
		{FilePath: podtatoSliFile, ResourceName: "splunk/sli.yaml"},
		{FilePath: podtatoSloFile, ResourceName: "slo.yaml"},
	}
	shipyardFileBase64 := base64.StdEncoding.EncodeToString(testEnv.shipyard)

	_, errP := testEnv.API.APIHandler.CreateProject(models.CreateProject{
		Name:     &testEnv.EventData.Project,
		Shipyard: &shipyardFileBase64,
	})

	require.NoError(t, errP.ToError())
	token, errToken := GetGiteaToken()
	require.NoError(t, errToken)

	os.Setenv("GITEA_TOKEN", token)
	err = testEnv.SetupTestEnvironment()
	require.NoError(t, err)

	// Make sure project is delete after the tests are completed
	defer func() {
		if err := testEnv.Cleanup(); err != nil {
			require.NoError(t, err)
		}
	}()

	// Upload additional resources to the keptn project
	for _, resource := range additionalResources {
		content, err := os.ReadFile(resource.FilePath)
		require.NoError(t, err, "Unable to read file %s", resource.FilePath)

		err = testEnv.API.AddServiceResource(testEnv.EventData.Project, testEnv.EventData.Stage,
			testEnv.EventData.Service, resource.ResourceName, string(content))

		require.NoErrorf(t, err, "unable to create file %s", resource.ResourceName)
	}

	// Test if the configuration of splunk was without errors
	t.Run("Configure splunk", func(t *testing.T) {
		// Configure monitoring
		t.Log("Configure splunk")
		configureMonitoring, err := readKeptnContextExtendedCE(podtatoConfigureMonitoringEvent)
		require.NoError(t, err)

		configureMonitoringContext, err := testEnv.API.SendEvent(configureMonitoring)
		require.NoError(t, err)

		// wait until splunk is configured correctly ...
		requireWaitForEvent(t,
			testEnv.API,
			5*time.Second,
			1*time.Second,
			configureMonitoringContext,
			"sh.keptn.event.configure-monitoring.finished",
			func(event *models.KeptnContextExtendedCE) bool {
				responseEventData, err := parseKeptnEventData(event)
				require.NoError(t, err)
				return responseEventData.Result == "pass" && responseEventData.Status == "succeeded"
			},
			"splunk-sli-provider",
		)
	})

	// Test deployment of podtatohead v0.1.1 where all SLI values must be according to SLO
	t.Run("Deploy podtatohead v0.1.1", func(t *testing.T) {
		t.Log("Deploy podtatohead v0.1.1")
		// Send the event to keptn to deploy, test and evaluate the service
		keptnContext, err := testEnv.API.SendEvent(testEnv.Event)

		require.NoError(t, err)

		// Checking a .started event is received from the evaluation process
		requireWaitForEvent(t,
			testEnv.API,
			15*time.Minute,
			1*time.Second,
			keptnContext,
			"sh.keptn.event.get-sli.started",
			func(_ *models.KeptnContextExtendedCE) bool {
				return true
			},
			"splunk-sli-provider",
		)

		requireWaitForEvent(t,
			testEnv.API,
			15*time.Minute,
			1*time.Second,
			keptnContext,
			"sh.keptn.event.get-sli.finished",
			func(event *models.KeptnContextExtendedCE) bool {
				responseEventData, err := parseKeptnEventData(event)
				require.NoError(t, err)
				return responseEventData.Result == "pass" && responseEventData.Status == "succeeded"
			},
			"splunk-sli-provider",
		)

		requireWaitForEvent(t,
			testEnv.API,
			15*time.Minute,
			1*time.Second,
			keptnContext,
			"sh.keptn.event.evaluation.finished",
			func(event *models.KeptnContextExtendedCE) bool {
				responseEventData, err := parseKeptnEventData(event)
				require.NoError(t, err)

				return responseEventData.Result == "pass" && responseEventData.Status == "succeeded"
			},
			"lighthouse-service",
		)
	})
	// Note: Remediation skipped in this test because it is configured to trigger after 10m
	// TODO: Maybe make a REST call to the alertmanager and ask which alerts are pending?
}
