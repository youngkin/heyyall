// Copyright (c) 2020 Richard Youngkin. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package internal

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/youngkin/heyyall/api"
)

var (
	update           = flag.Bool("update", false, "update .golden files")
	goldenFileDir    = "testdata"
	goldenFileSuffix = ".golden"
)

// TestResponseHapopyPath is interesting given that it captures stdout
// for a later comparison for testing. However, it's not practical due
// to timing differences from run-to-run. As a result run results are
// not comparable between runs.
func TestResponseHandlerHappyPath(t *testing.T) {
	rescueStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("received error capturing stdout: %s", err)
	}
	os.Stdout = w
	testName := "HappyPath"

	// 	maxConcurrentRqsts := 1
	// 	httpStatus := 200
	// 	ep := api.Endpoint{URL: "http://somewhere.com/xyz", Method: http.MethodGet}
	// 	responseC := make(chan Response, maxConcurrentRqsts)
	// 	doneC := make(chan struct{})

	// 	start := time.Now()

	// 	responseHandler := ResponseHandler{
	// 		ResponseC: responseC,
	// 		DoneC:     doneC,
	// 	}
	// 	go responseHandler.Start()

	// 	time.Sleep(time.Millisecond * 100) // delay so RequestDuration isn't 0
	// 	t.Logf("duration in millis since start %d", time.Since(start)/time.Millisecond)
	// 	for i := 0; i < 110; i++ {
	// 		responseC <- Response{
	// 			HTTPStatus:      httpStatus,
	// 			Endpoint:        api.Endpoint{URL: ep.URL, Method: ep.Method},
	// 			RequestDuration: time.Millisecond * 100,
	// 		}
	// 	}

	// 	close(responseC)
	// 	<-doneC

	w.Close()
	actual, err := ioutil.ReadAll(r)
	if err != nil {
		t.Errorf("error reading actual results: %s", err)
	}

	if *update {
		updateGoldenFile(t, testName, string(actual))
	}

	expected := readGoldenFile(t, testName)
	if string(expected) != string(actual) {
		t.Errorf("expected %s, got %s", string(expected), string(actual))
	}

	os.Stdout = rescueStdout
}

func TestResponseStats(t *testing.T) {
	testName := "TestResponseStats"
	start := time.Now()
	time.Sleep(time.Millisecond * 1) // Test needs a runtime of at least 1ms
	url1 := "http://someurl/1"
	url2 := "http://someurl/2"
	url3 := "http://someurl/3"
	runSummary := RunSummary{
		RqstStats: RqstStats{
			MinRqstDuration: math.MaxInt64,
			MaxRqstDuration: 0,
		},
		EndpointOverviewSummary: make(map[string]map[string]int),
	}
	epRunSummary := make(map[string]*EndpointSummary)

	// URL1
	resp := Response{
		HTTPStatus:      http.StatusNotFound,
		Endpoint:        api.Endpoint{URL: url1, Method: http.MethodGet},
		RequestDuration: time.Millisecond * 100,
	}
	totalRunTime := time.Second * 0
	accumulateResponseStats(resp, &totalRunTime, &runSummary, epRunSummary)

	resp = Response{
		HTTPStatus:      http.StatusAccepted,
		Endpoint:        api.Endpoint{URL: url1, Method: http.MethodPut},
		RequestDuration: time.Millisecond * 1000,
	}
	accumulateResponseStats(resp, &totalRunTime, &runSummary, epRunSummary)
	resp = Response{
		HTTPStatus:      http.StatusAccepted,
		Endpoint:        api.Endpoint{URL: url1, Method: http.MethodPut},
		RequestDuration: time.Millisecond * 500,
	}
	accumulateResponseStats(resp, &totalRunTime, &runSummary, epRunSummary)

	// URL2
	resp = Response{
		HTTPStatus:      http.StatusCreated,
		Endpoint:        api.Endpoint{URL: url2, Method: http.MethodPost},
		RequestDuration: time.Millisecond * 250,
	}
	accumulateResponseStats(resp, &totalRunTime, &runSummary, epRunSummary)

	// URL3 - POST
	resp = Response{
		HTTPStatus:      http.StatusCreated,
		Endpoint:        api.Endpoint{URL: url3, Method: http.MethodPost},
		RequestDuration: time.Millisecond * 250,
	}
	accumulateResponseStats(resp, &totalRunTime, &runSummary, epRunSummary)
	// GET
	resp = Response{
		HTTPStatus:      http.StatusOK,
		Endpoint:        api.Endpoint{URL: url3, Method: http.MethodGet},
		RequestDuration: time.Millisecond * 250,
	}
	accumulateResponseStats(resp, &totalRunTime, &runSummary, epRunSummary)
	resp = Response{
		HTTPStatus:      http.StatusOK,
		Endpoint:        api.Endpoint{URL: url3, Method: http.MethodGet},
		RequestDuration: time.Millisecond * 750,
	}
	accumulateResponseStats(resp, &totalRunTime, &runSummary, epRunSummary)
	// PUT
	resp = Response{
		HTTPStatus:      http.StatusOK,
		Endpoint:        api.Endpoint{URL: url3, Method: http.MethodPut},
		RequestDuration: time.Millisecond * 250,
	}
	accumulateResponseStats(resp, &totalRunTime, &runSummary, epRunSummary)
	resp = Response{
		HTTPStatus:      http.StatusOK,
		Endpoint:        api.Endpoint{URL: url3, Method: http.MethodPut},
		RequestDuration: time.Millisecond * 750,
	}
	accumulateResponseStats(resp, &totalRunTime, &runSummary, epRunSummary)
	resp = Response{
		HTTPStatus:      http.StatusOK,
		Endpoint:        api.Endpoint{URL: url3, Method: http.MethodPut},
		RequestDuration: time.Millisecond * 1250,
	}
	accumulateResponseStats(resp, &totalRunTime, &runSummary, epRunSummary)
	resp = Response{
		HTTPStatus:      http.StatusOK,
		Endpoint:        api.Endpoint{URL: url3, Method: http.MethodPut},
		RequestDuration: time.Millisecond * 1750,
	}
	accumulateResponseStats(resp, &totalRunTime, &runSummary, epRunSummary)
	// DELETE
	resp = Response{
		HTTPStatus:      http.StatusOK,
		Endpoint:        api.Endpoint{URL: url3, Method: http.MethodDelete},
		RequestDuration: time.Millisecond * 900,
	}
	accumulateResponseStats(resp, &totalRunTime, &runSummary, epRunSummary)

	// FINALIZE
	rh := ResponseHandler{}
	err := rh.finalizeResponseStats(start, &totalRunTime, &runSummary, epRunSummary)
	if err != nil {
		t.Errorf("unexpected error finalizing response stats: %s", err)
	}
	actual, err := json.Marshal(runSummary)
	if err != nil {
		t.Errorf("error marshaling RunSummary into string: %+v. Error: %s\n", runSummary, err)
	}

	// fmt.Printf("\n%s\n\n", string(actual))

	if *update {
		updateGoldenFile(t, testName, string(actual))
	}

	expected := readGoldenFile(t, testName)
	expectedJSON := RunSummary{}
	err = json.Unmarshal(expected, &expectedJSON)
	if err != nil {
		t.Errorf("error unmarshaling GoldenFile %s into RunSummary, Error: %s\n", expected, err)
	}

	if len(expectedJSON.EndpointRunSummary) != len(runSummary.EndpointRunSummary) {
		t.Errorf("expected %d endpoints, got %d", len(expectedJSON.EndpointRunSummary), len(runSummary.EndpointRunSummary))
	}
	if len(expectedJSON.EndpointOverviewSummary) != len(runSummary.EndpointOverviewSummary) {
		t.Errorf("expected %d endpoints, got %d", len(expectedJSON.EndpointOverviewSummary), len(runSummary.EndpointOverviewSummary))
	}

	if expectedJSON.RqstStats != runSummary.RqstStats {
		t.Errorf("expected %+v, got %+v", expectedJSON.RqstStats, runSummary.RqstStats)
	}
	if expectedJSON.EndpointOverviewSummary[url1][http.MethodPut] != runSummary.EndpointOverviewSummary[url1][http.MethodPut] {
		t.Errorf("expected %d PUTs for %s, got %d", expectedJSON.EndpointOverviewSummary[url1][http.MethodPut], url1,
			runSummary.EndpointOverviewSummary[url1][http.MethodPut])
	}
	if expectedJSON.EndpointOverviewSummary[url1][http.MethodGet] != runSummary.EndpointOverviewSummary[url1][http.MethodGet] {
		t.Errorf("expected %d GETs for %s, got %d", expectedJSON.EndpointOverviewSummary[url1][http.MethodGet], url1,
			runSummary.EndpointOverviewSummary[url1][http.MethodPut])
	}

	if expectedJSON.EndpointOverviewSummary[url2][http.MethodPost] != runSummary.EndpointOverviewSummary[url2][http.MethodPost] {
		t.Errorf("expected %d GETs for %s, got %d", expectedJSON.EndpointOverviewSummary[url2][http.MethodPost], url2,
			runSummary.EndpointOverviewSummary[url2][http.MethodPost])
	}

	if expectedJSON.EndpointOverviewSummary[url3][http.MethodPost] != runSummary.EndpointOverviewSummary[url3][http.MethodPost] {
		t.Errorf("expected %d GETs for %s, got %d", expectedJSON.EndpointOverviewSummary[url3][http.MethodPost], url3,
			runSummary.EndpointOverviewSummary[url3][http.MethodPost])
	}
	if expectedJSON.EndpointOverviewSummary[url3][http.MethodPut] != runSummary.EndpointOverviewSummary[url3][http.MethodPut] {
		t.Errorf("expected %d GETs for %s, got %d", expectedJSON.EndpointOverviewSummary[url3][http.MethodPut], url3,
			runSummary.EndpointOverviewSummary[url3][http.MethodPut])
	}
	if expectedJSON.EndpointOverviewSummary[url3][http.MethodGet] != runSummary.EndpointOverviewSummary[url3][http.MethodGet] {
		t.Errorf("expected %d GETs for %s, got %d", expectedJSON.EndpointOverviewSummary[url3][http.MethodGet], url3,
			runSummary.EndpointOverviewSummary[url3][http.MethodGet])
	}
	if expectedJSON.EndpointOverviewSummary[url3][http.MethodDelete] != runSummary.EndpointOverviewSummary[url3][http.MethodDelete] {
		t.Errorf("expected %d GETs for %s, got %d", expectedJSON.EndpointOverviewSummary[url3][http.MethodDelete], url3,
			runSummary.EndpointOverviewSummary[url3][http.MethodDelete])
	}
}

func updateGoldenFile(t *testing.T, testName string, contents string) {
	gf := filepath.Join(goldenFileDir, testName+goldenFileSuffix)
	t.Log("update golden file")
	if err := ioutil.WriteFile(gf, []byte(contents), 0644); err != nil {
		t.Fatalf("failed to update golden file: %s", err)
	}
}

func readGoldenFile(t *testing.T, testName string) []byte {
	gf := filepath.Join(goldenFileDir, testName+goldenFileSuffix)
	gfc, err := ioutil.ReadFile(gf)
	if err != nil {
		t.Fatalf("failed reading golden file: %s", err)
	}
	return gfc
}
