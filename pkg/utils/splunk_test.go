package utils

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
)

// Tests the getSplunkCredentials function
func TestGetSplunkCredentials(t *testing.T) {
	err := godotenv.Load(".env.local")
	env := EnvConfig{}
	env.SplunkApiToken = os.Getenv("SPLUNK_API_TOKEN")
	env.SplunkHost = os.Getenv("SPLUNK_HOST")
	env.SplunkPort = os.Getenv("SPLUNK_PORT")
	if err != nil {
		env.SplunkApiToken = "splunkApiTOken"
		env.SplunkHost = "splunkHost"
		env.SplunkPort = "splunkPort"
	}

	sp, err := GetSplunkCredentials(env)

	switch err {
	case nil:
		t.Logf("Splunk credentials : %v", sp)
		if sp.Host == "" || sp.Port == "" || sp.Token == "" {
			t.Fatal("If Host, Port or token are empty. An error should be returned")
		}
	default:
		t.Logf("Received expected error : %v", err)
	}
}
