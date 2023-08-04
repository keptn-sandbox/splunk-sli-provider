package utils

import (
	"net"
	"strings"

	splunk "github.com/AmadeusITGroup/keptn-splunk-sli-provider/pkg/splunksdk/client"
)

func ValidateSearchQuery(searchQuery string) string {
	// the search must start with the "search" keyword
	const query_prefix = "search "
	if !strings.HasPrefix(searchQuery, query_prefix) {
		return query_prefix + searchQuery
	}
	return searchQuery
}

func ValidateAlertQuery(alertQuery string) string {
	// the search must start with the "search" keyword
	const query_prefix = "search "
	if strings.HasPrefix(alertQuery, query_prefix) {
		return strings.Replace(alertQuery, "search ", "", 1)
	}
	return alertQuery
}

func CreateEndpoint(client *splunk.SplunkClient, service string) {
	host := client.Host
	port := client.Port

	switch {
	case strings.HasPrefix(host, "https://"):
		host = strings.Replace(host, "https://", "", 1)
	case strings.HasPrefix(host, "http://"):
		host = strings.Replace(host, "http://", "", 1)
	}

	client.Endpoint = "https://" + net.JoinHostPort(host, port) + "/" + service
	client.Endpoint = strings.ReplaceAll(client.Endpoint, " ", "")
}
