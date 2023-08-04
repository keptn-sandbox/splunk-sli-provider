package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

func main() {

	body := []byte(`{
		"data": {
			"get-sli": {
			  "customFilters": [],
			  "end": "2021-01-15T15:09:45.000Z",
			  "indicators": [
				"number_of_errors"
			  ],
			  "sliProvider": "splunk",
			  "start": "2021-01-15T15:04:45.000Z"
			},
			"labels": null,
			"message": "",
			"project": "fulltour",
			"result": "",
			"service": "helloservice",
			"stage": "qa",
			"status": ""
		},
		"id": "7dbc47b8-a1db-4cb7-9f64-dd55d5563f67",
		"shkeptncontext": "78e941cd-a946-4058-94df-8e577bbe4ds8de",
		"shkeptnspecversion": "0.2.4",
		"source": "lighthouse-service",
		"specversion": "1.0",
		"time": "2023-06-01T09:31:37.106185069Z",
		"gitcommitid": "dbfa126d494e980a66a81180d24227ffff82ee04",
		"type": "sh.keptn.event.get-sli.triggered"}`,
	)
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, "http://localhost:8080/", bytes.NewBuffer(body))
	if err != nil {
		fmt.Printf("Error : %s\n", err)
		return
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Cache-Control", "no-cache")
	req.Header.Add("Content-Type", "application/cloudevents+json")
	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	defer res.Body.Close()
	bo, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Response : %s", bo)
}
