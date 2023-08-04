package utils

import (
	"net/http"
	"net/http/httptest"
	"strings"
)

func MultitpleMockRequest(getResponses []string, postResponses []string, paths []string, sslVerificationActivated bool) *httptest.Server {
	var server *httptest.Server
	handlerFunction := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		writeResponses(getResponses, postResponses, w, r, paths)
	})
	switch sslVerificationActivated {
	case true:
		server = httptest.NewTLSServer(handlerFunction)
	default:
		server = httptest.NewServer(handlerFunction)
	}
	return server
}

func writeResponses(getResponses []string, postResponses []string, w http.ResponseWriter, r *http.Request, paths []string) {

	switch method := r.Method; method {
	case http.MethodGet:
		for i, response := range getResponses {
			if response != "" && strings.HasSuffix(r.URL.Path, paths[i]) {
				_, _ = (w).Write([]byte(response))
			}
		}
	case http.MethodPost:
		for i, response := range postResponses {
			if response != "" && strings.HasSuffix(r.URL.Path, paths[i]) {
				_, _ = (w).Write([]byte(response))
			}
		}
	}

}
