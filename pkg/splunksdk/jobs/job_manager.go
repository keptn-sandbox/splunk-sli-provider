package jobs

import (
	"net/http"
	"net/url"

	splunk "github.com/AmadeusITGroup/keptn-splunk-sli-provider/pkg/splunksdk/client"
	utils "github.com/AmadeusITGroup/keptn-splunk-sli-provider/pkg/splunksdk/pkg/utils"
)

func PostJob(client *splunk.SplunkClient, spRequest *SearchRequest) (*http.Response, error) {

	return HttpJobRequest(client, http.MethodPost, spRequest)
}

func GetJob(client *splunk.SplunkClient) (*http.Response, error) {

	return HttpJobRequest(client, http.MethodGet, nil)
}

func HttpJobRequest(client *splunk.SplunkClient, method string, spRequest *SearchRequest) (*http.Response, error) {

	if spRequest == nil {
		spRequest = &SearchRequest{}
	}

	spRequest.Params.OutputMode = "json"
	spRequest.Params.ExecMode = "blocking"

	// parameters of the request
	params := url.Values{}
	params.Add("output_mode", spRequest.Params.OutputMode)
	params.Add("exec_mode", spRequest.Params.ExecMode)

	if method == http.MethodPost {
		params.Add("search", utils.ValidateSearchQuery(spRequest.Params.SearchQuery))
		if spRequest.Params.EarliestTime != "" {
			params.Add("earliest_time", spRequest.Params.EarliestTime)
		}
		if spRequest.Params.LatestTime != "" {
			params.Add("latest_time", spRequest.Params.LatestTime)
		}
	}

	return splunk.MakeHttpRequest(client, method, spRequest.Headers, params)
}
