package utils

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	splunk "github.com/keptn-sandbox/splunk-sli-provider/pkg/splunksdk/client"
	splunktest "github.com/keptn-sandbox/splunk-sli-provider/pkg/splunksdk/pkg/utils"

	logger "github.com/sirupsen/logrus"
)

type SplunkCredentials struct {
	Host       string `json:"host" yaml:"spHost"`
	Port       string `json:"port" yaml:"spPort"`
	Username   string `json:"username" yaml:"spUsername"`
	Password   string `json:"password" yaml:"spPassword"`
	Token      string `json:"token" yaml:"spApiToken"`
	SessionKey string `json:"sessionKey" yaml:"spSessionKey"`
}

// getSplunkCredentials get the splunk host, port and api token from the environment variables set from secret
func GetSplunkCredentials(env EnvConfig) (*SplunkCredentials, error) {

	logger.Info("Trying to retrieve splunk credentials ...")
	splunkCreds := SplunkCredentials{}
	switch {
	case env.SplunkHost != "" && env.SplunkPort != "" && (env.SplunkApiToken != "" || (env.SplunkUsername != "" && env.SplunkPassword != "") || env.SplunkSessionKey != ""):
		splunkCreds.Host = strings.ReplaceAll(env.SplunkHost, " ", "")
		splunkCreds.Token = env.SplunkApiToken
		splunkCreds.Port = env.SplunkPort
		splunkCreds.Username = env.SplunkUsername
		splunkCreds.Password = env.SplunkPassword
		splunkCreds.SessionKey = env.SplunkSessionKey

		logger.Info("Successfully retrieved splunk credentials")

	default:
		if env.SplunkHost == "" {
			logger.Error("SP_HOST not set")
		}
		if env.SplunkPort == "" {
			logger.Error("SP_PORT not set")
		}
		if env.SplunkApiToken == "" {
			logger.Error("SP_API_TOKEN not set")
		}
		if env.SplunkUsername == "" || env.SplunkPassword == "" {
			logger.Error("SP_USERNAME and SP_PASSWORD not set")
		}
		if env.SplunkSessionKey == "" {
			logger.Error("SP_SESSION_KEY not set")
		}
		return nil, fmt.Errorf("invalid credentials found in SP_HOST, SP_PORT, SP_HOST, SP_API_TOKEN, SP_USERNAME, SP_PASSWORD and/or SP_SESSION_KEY")
	}

	return &splunkCreds, nil
}

// Creates an authenticated splunk client
func ConnectToSplunk(splunkCreds SplunkCredentials, skipSSL bool) *splunk.SplunkClient {

	logger.Info("Connecting to Splunk ...")
	var client *splunk.SplunkClient
	switch {
	case splunkCreds.Token != "":
		client = splunk.NewClientAuthenticatedByToken(
			&http.Client{
				Timeout: time.Duration(60) * time.Second,
			},
			splunkCreds.Host,
			splunkCreds.Port,
			splunkCreds.Token,
			skipSSL,
		)
	case splunkCreds.SessionKey != "":
		client = splunk.NewClientAuthenticatedBySessionKey(
			&http.Client{
				Timeout: time.Duration(60) * time.Second,
			},
			splunkCreds.Host,
			splunkCreds.Port,
			splunkCreds.SessionKey,
			skipSSL,
		)
	default:
		client = splunk.NewBasicAuthenticatedClient(
			&http.Client{
				Timeout: time.Duration(60) * time.Second,
			},
			splunkCreds.Host,
			splunkCreds.Port,
			splunkCreds.Username,
			splunkCreds.Password,
			skipSSL,
		)
	}

	return client
}

// Build a mock splunk server returning default responses when getting  get and post requests
func BuildMockSplunkServer(splunkResult float64) *httptest.Server {

	jsonResponsePOST := `{
		"sid": "10"
	}`
	jsonResponseGET := `{
		"results":[{"theRequest":"` + fmt.Sprintf("%f", splunkResult) + `"}]
	}`
	splunkResponses := make([]map[string]interface{}, 2)
	splunkResponses[0] = map[string]interface{}{
		http.MethodPost: jsonResponsePOST,
	}
	splunkResponses[1] = map[string]interface{}{
		http.MethodGet: jsonResponseGET,
	}
	splunkServer := splunktest.MultitpleMockRequest(splunkResponses, true)

	return splunkServer
}
