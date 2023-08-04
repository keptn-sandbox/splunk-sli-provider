package jobs

import (
	"net/http"
	"testing"
	"time"

	splunk "github.com/AmadeusITGroup/keptn-splunk-sli-provider/pkg/splunksdk/client"
	"github.com/AmadeusITGroup/keptn-splunk-sli-provider/pkg/splunksdk/pkg/utils"
	splunkTest "github.com/AmadeusITGroup/keptn-splunk-sli-provider/pkg/splunksdk/pkg/utils"

	"github.com/joho/godotenv"
)

func TestGetMetric(t *testing.T) {

	_ = godotenv.Load(".env")

	jsonResponsePOST := `{
		"sid": "1689673231.191"
	}`

	jsonResponseGET := `{
		"results":[{"count":"2566"}]
	}`

	responses := make([]map[string]interface{}, 2)
	responses[0] = map[string]interface{}{
		http.MethodPost: jsonResponsePOST,
	}
	responses[1] = map[string]interface{}{
		http.MethodGet: jsonResponseGET,
	}

	server := splunkTest.MultitpleMockRequest(responses, true)

	client := splunk.NewClientAuthenticatedByToken(
		&http.Client{
			Timeout: time.Duration(60) * time.Second,
		},
		splunkTest.GetTestHostname(server),
		splunkTest.GetTestPort(server),
		splunkTest.GetTestToken(),
		true,
	)

	defer server.Close()

	spReq := SearchRequest{
		Params: SearchParams{
			SearchQuery: "source=\"http:podtato-error\" (index=\"keptn-splunk-dev\") \"[error]\" | stats count",
		},
	}

	metric, err := GetMetricFromNewJob(client, &spReq)

	if err != nil {
		t.Fatalf("Got an error : %s", err)
	}

	expectedMetric := 2566
	if metric != float64(expectedMetric) {
		t.Fatalf("Expected %v but got %v.", expectedMetric, metric)
	}
}

func TestCreateJob(t *testing.T) {

	_ = godotenv.Load(".env")

	jsonResponsePOST := `{
		"sid": "1689673231.191"
	}`
	server := splunkTest.MockRequest(jsonResponsePOST, true)
	defer server.Close()

	spReq := SearchRequest{
		Params: SearchParams{
			SearchQuery: "source=\"http:podtato-error\" (index=\"keptn-splunk-dev\") \"[error]\" | stats count",
		},
	}
	client := splunk.NewClientAuthenticatedByToken(
		&http.Client{
			Timeout: time.Duration(60) * time.Second,
		},
		splunkTest.GetTestHostname(server),
		splunkTest.GetTestPort(server),
		splunkTest.GetTestToken(),
		true,
	)

	utils.CreateEndpoint(client, splunkTest.JobsPathv2)

	sid, err := CreateJob(client, &spReq, splunkTest.JobsPathv2)

	if err != nil {
		t.Fatalf("Got an error : %s", err)
	}

	expectedSID := "1689673231.191"
	if sid != expectedSID {
		t.Fatalf("Expected %v but got %v.", expectedSID, sid)
	}
}

func TestRetrieveJobResult(t *testing.T) {

	_ = godotenv.Load(".env")

	jsonResponseGET := `{
		"results":[{"count":"2566"}]
	}`
	server := splunkTest.MockRequest(jsonResponseGET, true)
	defer server.Close()

	client := splunk.NewClientAuthenticatedByToken(
		&http.Client{
			Timeout: time.Duration(60) * time.Second,
		},
		splunkTest.GetTestHostname(server),
		splunkTest.GetTestPort(server),
		splunkTest.GetTestToken(),
		true,
	)
	utils.CreateEndpoint(client, splunkTest.JobsPathv2)
	results, err := RetrieveJobResult(client, "1689673231.191")

	if err != nil {
		t.Fatalf("Got an error : %s", err)
	}

	expectedRes := make([]map[string]string, 1)
	expectedRes[0] = map[string]string{
		"count": "2566",
	}

	if results[0]["count"] != expectedRes[0]["count"] {
		t.Fatalf("Expected %v but got %v.", expectedRes, results)
	}
}
