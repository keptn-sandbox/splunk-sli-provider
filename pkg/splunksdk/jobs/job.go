package jobs

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	splunk "github.com/keptn-sandbox/splunk-sli-provider/pkg/splunksdk/client"
	utils "github.com/keptn-sandbox/splunk-sli-provider/pkg/splunksdk/pkg/utils"
)

const resutltUri = "results"
const jobsPathv2 = "services/search/v2/jobs/"

type SearchRequest struct {
	Headers map[string]string
	Params  SearchParams
}

type SearchParams struct {
	// splunk search in spl syntax
	SearchQuery string
	OutputMode  string `default:"json"`
	// splunk returns a job SID only if the job is complete
	ExecMode string `default:"blocking"`
	// earliest (inclusive) time bounds for the search
	EarliestTime string
	// latest (exclusive) time bounds for the search
	LatestTime string
}

// Return a metric from a new created job
func GetMetricFromNewJob(client *splunk.SplunkClient, spRequest *SearchRequest) (float64, error) {

	sid, err := CreateJob(client, spRequest, jobsPathv2)
	if err != nil {
		return -1, fmt.Errorf("error while creating the job : %w", err)
	}

	res, err := RetrieveJobResult(client, sid)

	if err != nil {
		return -1, fmt.Errorf("error while handling the results. Error message : %w", err)
	}
	// if the result is not a metric
	if len(res) != 1 {
		if len(res) == 0 {
			err = fmt.Errorf("no result found")
		}
		return -1, fmt.Errorf("result is not a metric. Error message : %w", err)
	}
	var metrics []string

	for _, v := range res[0] {
		metrics = append(metrics, v)
	}
	metric, err := strconv.ParseFloat(metrics[0], 64)
	if err != nil {
		return -1, fmt.Errorf("convert metric to float failed. Error message : %w", err)
	}

	return metric, nil
}

// this function create a new job and return its SID
func CreateJob(client *splunk.SplunkClient, spRequest *SearchRequest, service string) (string, error) {

	// create the endpoint for the request
	utils.CreateEndpoint(client, jobsPathv2)

	resp, err := PostJob(client, spRequest)

	if err != nil {
		return "", fmt.Errorf("error while making the post request : %w", err)
	}

	body, err := io.ReadAll(resp.Body)
	// handle error
	if !strings.HasPrefix(strconv.Itoa(resp.StatusCode), "2") {
		status, err := splunk.HandleHttpError(body)
		switch err {
		case nil:
			return "", fmt.Errorf("http error :  %s", status)
		default:
			return "", fmt.Errorf("http error :  %s", resp.Status)
		}
	}

	if err != nil {
		return "", fmt.Errorf("error while getting the body of the post request : %w", err)
	}

	// create the new endpoint for the post request
	var sid string
	sid, err = getSID(body)
	if err != nil {
		return "", fmt.Errorf("error : %w", err)
	}

	return sid, nil
}

// return the result of a job get by its SID
func RetrieveJobResult(client *splunk.SplunkClient, sid string) ([]map[string]string, error) {

	newEndpoint := client.Endpoint + sid
	// check if the endpoint is correctly formed
	if !strings.HasSuffix(newEndpoint, "/") {
		newEndpoint += "/"
	}

	// the endpoint where to find the corresponding job
	client.Endpoint = newEndpoint + resutltUri

	// make the get request
	getResp, err := GetJob(client)
	if err != nil {
		return nil, fmt.Errorf("error while making the get request : %w", err)
	}

	// get the body of the response
	getBody, err := io.ReadAll(getResp.Body)
	// handle error
	if !strings.HasPrefix(strconv.Itoa(getResp.StatusCode), "2") {
		status, err := splunk.HandleHttpError(getBody)
		switch err {
		case nil:
			return nil, fmt.Errorf("http error :  %s", status)
		default:
			return nil, fmt.Errorf("http error :  %s", getResp.Status)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("error while getting the body of the get request : %w", err)
	}

	// only get the result section of the response
	type Response struct {
		Results []map[string]string `json:"results"`
	}

	results := Response{}
	errUmarshall := json.Unmarshal([]byte(getBody), &results)

	if errUmarshall != nil {
		return nil, errUmarshall
	}
	return results.Results, nil
}

// Return the sid from the body of the given response
func getSID(resp []byte) (string, error) {
	respJson := string(resp)

	var sid map[string]string
	errUmarshall := json.Unmarshal([]byte(respJson), &sid)

	if errUmarshall != nil {
		return "", errUmarshall
	}

	if len(sid) <= 0 {
		return "", fmt.Errorf("no sid found")
	}
	return sid["sid"], nil
}
