// Copyright (c) 2020 Richard Youngkin. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package internal

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
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

// TestResponseHappyPath is interesting given that it captures stdout
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
	runResults := RunResults{
		RunSummary: RunSummary{
			RqstStats: RqstStats{
				MinRqstDuration: math.MaxInt64,
				MaxRqstDuration: 0,
			},
		},
		EndpointSummary: make(map[string]map[string]int),
	}
	epRunSummary := make(map[string]*EndpointDetail)

	rh := ResponseHandler{ReportDetail: Long}

	// URL1
	resp := Response{
		HTTPStatus:      http.StatusNotFound,
		Endpoint:        api.Endpoint{URL: url1, Method: http.MethodGet},
		RequestDuration: time.Millisecond * 100,
	}
	totalRunTime := time.Second * 0
	rh.accumulateResponseStats(resp, &totalRunTime, &runResults, epRunSummary)

	resp = Response{
		HTTPStatus:      http.StatusAccepted,
		Endpoint:        api.Endpoint{URL: url1, Method: http.MethodPut},
		RequestDuration: time.Millisecond * 1000,
	}
	rh.accumulateResponseStats(resp, &totalRunTime, &runResults, epRunSummary)
	resp = Response{
		HTTPStatus:      http.StatusAccepted,
		Endpoint:        api.Endpoint{URL: url1, Method: http.MethodPut},
		RequestDuration: time.Millisecond * 500,
	}
	rh.accumulateResponseStats(resp, &totalRunTime, &runResults, epRunSummary)

	// URL2
	resp = Response{
		HTTPStatus:      http.StatusCreated,
		Endpoint:        api.Endpoint{URL: url2, Method: http.MethodPost},
		RequestDuration: time.Millisecond * 250,
	}
	rh.accumulateResponseStats(resp, &totalRunTime, &runResults, epRunSummary)

	// URL3 - POST
	resp = Response{
		HTTPStatus:      http.StatusCreated,
		Endpoint:        api.Endpoint{URL: url3, Method: http.MethodPost},
		RequestDuration: time.Millisecond * 250,
	}
	rh.accumulateResponseStats(resp, &totalRunTime, &runResults, epRunSummary)
	// GET
	resp = Response{
		HTTPStatus:      http.StatusOK,
		Endpoint:        api.Endpoint{URL: url3, Method: http.MethodGet},
		RequestDuration: time.Millisecond * 250,
	}
	rh.accumulateResponseStats(resp, &totalRunTime, &runResults, epRunSummary)
	resp = Response{
		HTTPStatus:      http.StatusOK,
		Endpoint:        api.Endpoint{URL: url3, Method: http.MethodGet},
		RequestDuration: time.Millisecond * 750,
	}
	rh.accumulateResponseStats(resp, &totalRunTime, &runResults, epRunSummary)
	// PUT
	resp = Response{
		HTTPStatus:      http.StatusOK,
		Endpoint:        api.Endpoint{URL: url3, Method: http.MethodPut},
		RequestDuration: time.Millisecond * 250,
	}
	rh.accumulateResponseStats(resp, &totalRunTime, &runResults, epRunSummary)
	resp = Response{
		HTTPStatus:      http.StatusOK,
		Endpoint:        api.Endpoint{URL: url3, Method: http.MethodPut},
		RequestDuration: time.Millisecond * 750,
	}
	rh.accumulateResponseStats(resp, &totalRunTime, &runResults, epRunSummary)
	resp = Response{
		HTTPStatus:      http.StatusOK,
		Endpoint:        api.Endpoint{URL: url3, Method: http.MethodPut},
		RequestDuration: time.Millisecond * 1250,
	}
	rh.accumulateResponseStats(resp, &totalRunTime, &runResults, epRunSummary)
	resp = Response{
		HTTPStatus:      http.StatusOK,
		Endpoint:        api.Endpoint{URL: url3, Method: http.MethodPut},
		RequestDuration: time.Millisecond * 1750,
	}
	rh.accumulateResponseStats(resp, &totalRunTime, &runResults, epRunSummary)
	// DELETE
	resp = Response{
		HTTPStatus:      http.StatusOK,
		Endpoint:        api.Endpoint{URL: url3, Method: http.MethodDelete},
		RequestDuration: time.Millisecond * 900,
	}
	rh.accumulateResponseStats(resp, &totalRunTime, &runResults, epRunSummary)

	// TODO: Add read from DONE channel to verify that ResponseHandler closes
	// out all resources as expected

	// FINALIZE
	err := rh.finalizeResponseStats(start, &totalRunTime, &runResults, epRunSummary)
	if err != nil {
		t.Errorf("unexpected error finalizing response stats: %s", err)
	}
	actual, err := json.Marshal(runResults)
	if err != nil {
		t.Errorf("error marshaling RunSummary into string: %+v. Error: %s\n", runResults, err)
	}

	fmt.Printf("\n%s\n\n", string(actual))

	if *update {
		updateGoldenFile(t, testName, string(actual))
	}

	expected := readGoldenFile(t, testName)
	expectedJSON := RunResults{}
	err = json.Unmarshal(expected, &expectedJSON)
	if err != nil {
		t.Errorf("error unmarshaling GoldenFile %s into RunSummary, Error: %s\n", expected, err)
	}

	if len(expectedJSON.EndpointDetails) != len(runResults.EndpointDetails) {
		t.Errorf("expected %d endpoints, got %d", len(expectedJSON.EndpointDetails), len(runResults.EndpointDetails))
	}
	if len(expectedJSON.EndpointSummary) != len(runResults.EndpointSummary) {
		t.Errorf("expected %d endpoints, got %d", len(expectedJSON.EndpointSummary), len(runResults.EndpointSummary))
	}

	if expectedJSON.RunSummary.RqstStats != runResults.RunSummary.RqstStats {
		t.Errorf("expected %+v, got %+v", expectedJSON.RunSummary.RqstStats, runResults.RunSummary.RqstStats)
	}

	if expectedJSON.EndpointSummary[url1][http.MethodPut] != runResults.EndpointSummary[url1][http.MethodPut] {
		t.Errorf("expected %d PUTs for %s, got %d", expectedJSON.EndpointSummary[url1][http.MethodPut], url1,
			runResults.EndpointSummary[url1][http.MethodPut])
	}
	if expectedJSON.EndpointSummary[url1][http.MethodGet] != runResults.EndpointSummary[url1][http.MethodGet] {
		t.Errorf("expected %d GETs for %s, got %d", expectedJSON.EndpointSummary[url1][http.MethodGet], url1,
			runResults.EndpointSummary[url1][http.MethodPut])
	}

	if expectedJSON.EndpointSummary[url2][http.MethodPost] != runResults.EndpointSummary[url2][http.MethodPost] {
		t.Errorf("expected %d GETs for %s, got %d", expectedJSON.EndpointSummary[url2][http.MethodPost], url2,
			runResults.EndpointSummary[url2][http.MethodPost])
	}

	if expectedJSON.EndpointSummary[url3][http.MethodPost] != runResults.EndpointSummary[url3][http.MethodPost] {
		t.Errorf("expected %d GETs for %s, got %d", expectedJSON.EndpointSummary[url3][http.MethodPost], url3,
			runResults.EndpointSummary[url3][http.MethodPost])
	}
	if expectedJSON.EndpointSummary[url3][http.MethodPut] != runResults.EndpointSummary[url3][http.MethodPut] {
		t.Errorf("expected %d GETs for %s, got %d", expectedJSON.EndpointSummary[url3][http.MethodPut], url3,
			runResults.EndpointSummary[url3][http.MethodPut])
	}
	if expectedJSON.EndpointSummary[url3][http.MethodGet] != runResults.EndpointSummary[url3][http.MethodGet] {
		t.Errorf("expected %d GETs for %s, got %d", expectedJSON.EndpointSummary[url3][http.MethodGet], url3,
			runResults.EndpointSummary[url3][http.MethodGet])
	}
	if expectedJSON.EndpointSummary[url3][http.MethodDelete] != runResults.EndpointSummary[url3][http.MethodDelete] {
		t.Errorf("expected %d GETs for %s, got %d", expectedJSON.EndpointSummary[url3][http.MethodDelete], url3,
			runResults.EndpointSummary[url3][http.MethodDelete])
	}

	if *expectedJSON.EndpointDetails[url3].HTTPMethodRqstStats[http.MethodGet] != *runResults.EndpointDetails[url3].HTTPMethodRqstStats[http.MethodGet] {
		t.Errorf("expected %+v for %s method %s, got %+v", expectedJSON.EndpointDetails[url3].HTTPMethodRqstStats[http.MethodGet], url3,
			http.MethodGet, runResults.EndpointDetails[url3].HTTPMethodRqstStats[http.MethodGet])
	}
	if *expectedJSON.EndpointDetails[url3].HTTPMethodRqstStats[http.MethodPut] != *runResults.EndpointDetails[url3].HTTPMethodRqstStats[http.MethodPut] {
		t.Errorf("expected %+v for %s method %s, got %+v", expectedJSON.EndpointDetails[url3].HTTPMethodRqstStats[http.MethodPut], url3,
			http.MethodPut, runResults.EndpointDetails[url3].HTTPMethodRqstStats[http.MethodPut])
	}
	if *expectedJSON.EndpointDetails[url3].HTTPMethodRqstStats[http.MethodPost] != *runResults.EndpointDetails[url3].HTTPMethodRqstStats[http.MethodPost] {
		t.Errorf("expected %+v for %s method %s, got %+v", expectedJSON.EndpointDetails[url3].HTTPMethodRqstStats[http.MethodPost], url3,
			http.MethodPost, runResults.EndpointDetails[url3].HTTPMethodRqstStats[http.MethodPost])
	}
	if *expectedJSON.EndpointDetails[url3].HTTPMethodRqstStats[http.MethodDelete] != *runResults.EndpointDetails[url3].HTTPMethodRqstStats[http.MethodDelete] {
		t.Errorf("expected %+v for %s method %s, got %+v", expectedJSON.EndpointDetails[url3].HTTPMethodRqstStats[http.MethodDelete], url3,
			http.MethodDelete, runResults.EndpointDetails[url3].HTTPMethodRqstStats[http.MethodDelete])
	}

}

// TestGenHistogramSturges validates histogram generation when the Sturges method
// for calculating the number of histogram bins is used. The other method is the
// Rice method.
func TestGenHistogramSturges(t *testing.T) {
	testCases := []struct {
		name              string
		expectedMaxBinVal int
		expectedMinBinVal int
		expectedHist      map[float64]int
		respHandler       *ResponseHandler
		runResults        RunResults
	}{
		{
			name:              "No observations, nf = 0",
			expectedMaxBinVal: 0,
			expectedMinBinVal: math.MaxInt32,
			expectedHist:      map[float64]int{},
			respHandler: &ResponseHandler{
				timingResults: []time.Duration{},
				NormFactor:    0,
			},
			runResults: RunResults{},
		},
		{
			name:              "Observations: 1; nf = 0",
			expectedMaxBinVal: 1,
			expectedMinBinVal: 1,
			expectedHist:      map[float64]int{1: 1},
			respHandler: &ResponseHandler{
				timingResults: []time.Duration{time.Nanosecond * 1},
				NormFactor:    0,
			},
			runResults: RunResults{
				RunSummary: RunSummary{
					RqstStats: RqstStats{
						MinRqstDuration: time.Nanosecond * 1,
						MaxRqstDuration: time.Nanosecond * 1,
					},
				},
			},
		},
		{
			name:              "Observations: 3, 4; nf = 0",
			expectedMaxBinVal: 2,
			expectedMinBinVal: 0,
			expectedHist:      map[float64]int{2: 0, 4: 2},
			respHandler: &ResponseHandler{
				timingResults: []time.Duration{time.Nanosecond * 3, time.Nanosecond * 4},
				NormFactor:    0,
			},
			runResults: RunResults{
				RunSummary: RunSummary{
					RqstStats: RqstStats{
						MinRqstDuration: time.Nanosecond * 3,
						MaxRqstDuration: time.Nanosecond * 4,
					},
				},
			},
		},
		{
			name:              "Observations: 2, 4; nf = 0",
			expectedMaxBinVal: 1,
			expectedMinBinVal: 1,
			expectedHist:      map[float64]int{2: 1, 4: 1},
			respHandler: &ResponseHandler{
				timingResults: []time.Duration{time.Nanosecond * 2, time.Nanosecond * 4},
				NormFactor:    0,
			},
			runResults: RunResults{
				RunSummary: RunSummary{
					RqstStats: RqstStats{
						MinRqstDuration: time.Nanosecond * 2,
						MaxRqstDuration: time.Nanosecond * 4,
					},
				},
			},
		},
		{
			name:              "Observations 1, 2, 3, 4; nf = 0",
			expectedMaxBinVal: 2,
			expectedMinBinVal: 1,
			expectedHist:      map[float64]int{1.3333333333333333: 1, 2.6666666666666665: 1, 4: 2},
			respHandler: &ResponseHandler{
				timingResults: []time.Duration{time.Nanosecond * 1, time.Nanosecond * 2, time.Nanosecond * 3, time.Nanosecond * 4},
				NormFactor:    0,
			},
			runResults: RunResults{
				RunSummary: RunSummary{
					RqstStats: RqstStats{
						MinRqstDuration: time.Nanosecond * 1,
						MaxRqstDuration: time.Nanosecond * 4,
					},
				},
			},
		},
		{
			name:              "Observations 1, 2, 3, 4; nf = 3",
			expectedMaxBinVal: 1,
			expectedMinBinVal: 1,
			expectedHist:      map[float64]int{1: 1, 2: 1, 3: 1, 4: 1},
			respHandler: &ResponseHandler{
				timingResults: []time.Duration{time.Nanosecond * 1, time.Nanosecond * 2, time.Nanosecond * 3, time.Nanosecond * 4},
				NormFactor:    3,
			},
			runResults: RunResults{
				RunSummary: RunSummary{
					RqstStats: RqstStats{
						MinRqstDuration: time.Nanosecond * 1,
						MaxRqstDuration: time.Nanosecond * 4,
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			minBinKeyVal, maxBinKeyVal := tc.respHandler.generateHistogram(&tc.runResults)
			if maxBinKeyVal != tc.expectedMaxBinVal {
				t.Errorf("expected MaxBinVal %d,\n got %d", tc.expectedMaxBinVal, maxBinKeyVal)
			}
			if maxBinKeyVal != tc.expectedMaxBinVal {
				t.Errorf("expected MinBinVal %d, got %d", tc.expectedMinBinVal, minBinKeyVal)
			}
			if !histEqual(tc.expectedHist, tc.respHandler.histogram) {
				t.Errorf("expected %+v, got %+v", tc.expectedHist, tc.respHandler.histogram)
			}
		})
	}
}

func histEqual(a, b map[float64]int) bool {
	if len(a) != len(b) {
		return false
	}
	akeys := make([]float64, len(a))
	i := 0
	for k := range a {
		akeys[i] = k
		i++
	}

	for _, k := range akeys {
		b, ok := b[k]
		if !ok {
			return false
		}
		if b != a[k] {
			return false
		}
	}

	return true
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

func generateNormalDistribution(mean float64, stdDev int) float64 {
	x := rand.NormFloat64()*float64(stdDev) + float64(mean)
	if x < 0 {
		x = x * -1
	}
	return x

	// var v1, v2, s float64

	// for {
	// 	u1 := rand.Float64()
	// 	u2 := rand.Float64()
	// 	v1 = 2*u1 - 1
	// 	v2 = 2*u2 - 1
	// 	s = v1*v1 + v2*v2
	// 	if s < 1 {
	// 		break
	// 	}
	// }

	// if s == 0 {
	// 	return float64(0)
	// }

	// x := mean + (float64(stdDev) * (v1 * math.Sqrt(-2*math.Log(s)/s)))
	// if x < 0 {
	// 	x = x * -1
	// }
	// return x
}
