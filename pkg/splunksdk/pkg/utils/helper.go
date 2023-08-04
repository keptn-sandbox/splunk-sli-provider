package utils

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
)

const JobsPathv2 = "services/search/v2/jobs/"
const SavedSearchesPath = "services/saved/searches/"
const GetAlertsNames = "getAlertsNames"
const GetTriggeredAlerts = "getTriggeredAlerts"
const CreateAlerts = "createAlerts"
const GetTriggeredInstances = "getTriggeredInstances"

// mock an http server
func MockRequest(response string, sslVerificationActivated bool) *httptest.Server {
	var server *httptest.Server
	handlerFunction := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		writeResponses(response, w, r)
	})
	switch sslVerificationActivated {
	case true:
		server = httptest.NewTLSServer(handlerFunction)

	default:
		server = httptest.NewServer(handlerFunction)
	}
	return server
}

func MultitpleMockRequest(responses []map[string]interface{}, sslVerificationActivated bool) *httptest.Server {
	var server *httptest.Server
	handlerFunction := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		writeResponses(responses, w, r)
	})
	switch sslVerificationActivated {
	case true:
		server = httptest.NewTLSServer(handlerFunction)
	default:
		server = httptest.NewServer(handlerFunction)
	}
	return server
}

func writeResponses(responses interface{}, w http.ResponseWriter, r *http.Request) {

	switch resps := responses.(type) {
	case []map[string]interface{}:
		for _, response := range resps {
			switch method := r.Method; method {
			case http.MethodGet:
				switch {
				case response[GetTriggeredAlerts] != nil && strings.HasSuffix(r.URL.Path, "services/alerts/fired_alerts/"):
					_, _ = fmt.Fprintln(w, response[GetTriggeredAlerts])
				case response[GetTriggeredInstances] != nil && strings.Contains(r.URL.Path, "search/alerts/fired_alerts/"):
					_, _ = fmt.Fprintln(w, response[GetTriggeredInstances])
				case response[GetAlertsNames] != nil && strings.Contains(r.URL.Path, "services/saved/searches/"):
					_, _ = fmt.Fprintln(w, response[GetAlertsNames])
				case response[method] != nil:
					_, _ = fmt.Fprintln(w, response[method])
				}
			case http.MethodPost:
				switch {
				case response[CreateAlerts] != nil && strings.Contains(r.URL.Path, "services/saved/searches/"):
					_, _ = fmt.Fprintln(w, response[CreateAlerts])
				case response[method] != nil:
					_, _ = fmt.Fprintln(w, response[method])
				}
			}
		}
	case string:
		_, _ = (w).Write([]byte(resps))
	}
}

func GetTestHostname(server *httptest.Server) string {
	if os.Getenv("SPLUNK_HOST") != "" {
		return os.Getenv("SPLUNK_HOST")
	}
	return strings.Split(strings.Split(server.URL, ":")[1], "//")[1]

}

func GetTestPort(server *httptest.Server) string {
	if os.Getenv("SPLUNK_PORT") != "" {
		return os.Getenv("SPLUNK_PORT")
	}
	return strings.Split(server.URL, ":")[2]
}

func GetTestToken() string {
	if os.Getenv("SPLUNK_TOKEN") != "" {
		return os.Getenv("SPLUNK_TOKEN")
	}
	return "default"
}
