package utils

import (
	"testing"

	splunkjob "github.com/AmadeusITGroup/keptn-splunk-sli-provider/pkg/splunksdk/jobs"
)

// Tests the retrieveSearchTimeRange function
func TestRetrieveSearchTimeRange(t *testing.T) {

	const earliestTimeInRequest = "-2m"
	const earliestTimeInParams = "-1m"
	const latestTimeInRequest = "+2m"
	const latestTimeInParams = "+1m"

	splunkRequestParams := &splunkjob.SearchParams{}

	//Verify if the function overwrites the time values in params and set theme to the values specified in the search query
	splunkRequestParams.SearchQuery = "source=/opt/splunk/var/log/secure.log sourcetype=osx_secure earliest=" + earliestTimeInRequest + " latest=" + latestTimeInRequest + " |stats count"
	checkRetrieveSearchTimeRange(t, splunkRequestParams, earliestTimeInParams, latestTimeInParams, earliestTimeInRequest, latestTimeInRequest)

	//Verify if the function overwrites only the latest time value in params
	splunkRequestParams.SearchQuery = "source=/opt/splunk/var/log/secure.log sourcetype=osx_secure latest=" + latestTimeInRequest + " |stats count"
	checkRetrieveSearchTimeRange(t, splunkRequestParams, earliestTimeInParams, latestTimeInParams, earliestTimeInParams, latestTimeInRequest)

	//Verify if the function keeps the default values in params
	splunkRequestParams.SearchQuery = "source=/opt/splunk/var/log/secure.log sourcetype=osx_secure |stats count"
	checkRetrieveSearchTimeRange(t, splunkRequestParams, earliestTimeInParams, latestTimeInParams, earliestTimeInParams, latestTimeInParams)

	//Verify if the function overwrites only the earliest time value in params
	splunkRequestParams.SearchQuery = "source=/opt/splunk/var/log/secure.log sourcetype=osx_secure earliest=" + earliestTimeInRequest + " |stats count"
	checkRetrieveSearchTimeRange(t, splunkRequestParams, earliestTimeInParams, latestTimeInParams, earliestTimeInRequest, latestTimeInParams)

	//Verify if the function ignores the second earliest time given in the query
	splunkRequestParams.SearchQuery = "source=/opt/splunk/var/log/secure.log sourcetype=osx_secure earliest=" + earliestTimeInRequest + " earliest=" + earliestTimeInParams + " |stats count"
	checkRetrieveSearchTimeRange(t, splunkRequestParams, earliestTimeInParams, latestTimeInParams, earliestTimeInRequest, latestTimeInParams)

}

// checks if we have the expected parameters in the final request sent to splunk
func checkRetrieveSearchTimeRange(t *testing.T, splunkRequestParams *splunkjob.SearchParams, earliestTimeInParams string, latestTimeInParams string, expectedEarliestTime string, expectedLatestTime string) {

	// reinit the params
	splunkRequestParams.EarliestTime = earliestTimeInParams
	splunkRequestParams.LatestTime = latestTimeInParams

	splunkRequestParams.EarliestTime,
		splunkRequestParams.LatestTime,
		splunkRequestParams.SearchQuery = RetrieveQueryTimeRange(splunkRequestParams.EarliestTime, splunkRequestParams.LatestTime, splunkRequestParams.SearchQuery)

	switch {
	case splunkRequestParams.EarliestTime != expectedEarliestTime || splunkRequestParams.LatestTime != expectedLatestTime:
		t.Fatalf("EarliestTime value %s and LatestTime value %s in params are incorrect, should be %s and %s.",
			splunkRequestParams.EarliestTime, splunkRequestParams.LatestTime, expectedEarliestTime, expectedLatestTime)
	default:
		t.Log("Checked")
	}

}
