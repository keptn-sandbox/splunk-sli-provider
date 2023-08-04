package utils

import (
	"strings"
)

// check if the search string contains the earliest or latest time and return the time and the query
func getQueryTime(kind string, searchQuery string, defaultTime string) (string, string) {
	if !strings.Contains(searchQuery, kind) {
		return defaultTime, searchQuery
	}
	startIndex := strings.Index(searchQuery, kind)
	q1 := strings.Fields(searchQuery[startIndex:])

	timeValue := ""
	if !strings.HasPrefix(q1[0][len(kind)+1:], "\"") {
		timeValue = q1[0][len(kind)+1:]
		searchQuery = strings.ReplaceAll(searchQuery, q1[0], "")

		return strings.TrimSuffix(timeValue, "\""), searchQuery
	}
	for i, v := range q1 {
		switch i {
		case 0:
			timeValue += v[len(kind)+2:]
		default:
			timeValue += " " + v
		}
		if strings.HasSuffix(v, "\"") {
			break
		}
	}
	searchQuery = strings.ReplaceAll(searchQuery, timeValue, "")
	searchQuery = strings.ReplaceAll(searchQuery, kind+"=\"", "")

	return strings.TrimSuffix(timeValue, "\""), searchQuery
}

// get the earliest, latest time from the splunk search and also update the search query
func RetrieveQueryTimeRange(earliestTime string, latestTime string, searchQuery string) (string, string, string) {

	earliestTime, searchQuery = getQueryTime("earliest", searchQuery, earliestTime)
	latestTime, searchQuery = getQueryTime("latest", searchQuery, latestTime)

	return earliestTime, latestTime, searchQuery
}
