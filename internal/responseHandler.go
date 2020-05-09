// Copyright (c) 2020 Richard Youngkin. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package internal

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/rs/zerolog/log"
)

// RqstStats contains a set of common runtime stats reported at both the
// Summary and Endpoint level
type RqstStats struct {
	// TotalRqsts is the overall number of requests made during the run
	TotalRqsts int64
	// TotalRequestDuration is the sum of all request run durations
	TotalRequestDuration time.Duration
	// TotalRequestDurationStr is the string version of TotalRequestDuration
	TotalRequestDurationStr string
	// MaxRqstDuration is the longest request duration
	MaxRqstDuration time.Duration
	// MaxRqstDurationStr is a string representation of MaxRqstDuration
	MaxRqstDurationStr string
	// MinRqstDuration is the smallest request duration for an endpoint
	MinRqstDuration time.Duration
	// MinRqstDurationStr is a string representation of MinRqstDuration
	MinRqstDurationStr string
	// AvgRqstDuration is the average duration of a request for an endpoint
	AvgRqstDuration time.Duration
	// AvgRqstDurationStr is the average duration of a request in microseconds
	AvgRqstDurationStr string
}

// EndpointSummary is used to report an overview of the results of
// a load test run for a given endpoint.
type EndpointSummary struct {
	// URL is the endpoint URL
	URL string
	// HTTPMethodStatusDist summarizes, by HTTP method, the number of times a
	// given status was returned (e.g., 200, 201, 404, etc). More specifically,
	// it is a map keyed by HTTP method containing a map keyed by HTTP status
	// referencing the number of times that status was returned.
	HTTPMethodStatusDist map[string]map[int]int
	// RqstStats is a summary of runtime statistics
	RqstStats RqstStats
}

// RunSummary is used to report an overview of the results of a
// load test run
type RunSummary struct {
	// RqstRatePerSec is the overall request rate per second
	// rounded to the nearest integer
	RqstRatePerSec float64
	// RunDuration is the wall clock duration of the test
	RunDuration time.Duration
	// RunDurationStr is the string representation of RunDuration
	RunDurationStr string
	// ResponseDistribution is distribution of response times. There will be
	// 11 bucket; 10 microseconds or less, between 10us and 100us,
	// 100us and 1ms, 1ms to 10ms, 10ms to 100ms, 100ms to 1s, 1s to 1.1s,
	// 1.1s to 1.5s, 1.5s to 1.8s, 1.8s to 2.5s, 2.5s and above
	//ResponseDistribution map[float32]int
	// HTTPStatusDistribution is the distribution of HTTP response statuses
	//HTTPStatusDistribution map[string]int
	// MaxRqstRatePerSec is the maximum request rate per second
	// over 1/10th of the run duration or number of requests
	//MaxRqstRatePerSec int
	// MinRqstRatePerSec is the maximum request rate per second
	// over 1/10th of the run duration or number of requests
	//MinRqstRatePerSec int
	// RqstStats is a summary of runtime statistics
	RqstStats RqstStats
	// EndpointOverviewSummary describes how often each endpoint was called.
	// It is a map keyed by URL of a map keyed by HTTP verb with a value of
	// number of requests. So it's a summary of how often each HTTP verb
	// was called on each endpoint.
	EndpointOverviewSummary map[string]map[string]int
	// EndpointRunSummary is the per endpoint summary of results keyed by URL
	EndpointRunSummary map[string]*EndpointSummary
}

// ResponseHandler is responsible for accepting, summarizing, and reporting
// on the overall load test results.
type ResponseHandler struct {
	ResponseC chan Response
	DoneC     chan struct{}
}

// Start begins the process of accepting responses. It expects to be run as a goroutine.
func (rh ResponseHandler) Start() {
	log.Debug().Msg("ResponseHandler starting")
	epRunSummary := make(map[string]*EndpointSummary)
	runSummary := RunSummary{RqstStats: RqstStats{MaxRqstDuration: time.Duration(-1), MinRqstDuration: time.Duration(math.MaxInt64)}}
	runSummary.EndpointOverviewSummary = make(map[string]map[string]int)

	start := time.Now()
	var totalRunTime time.Duration

	for {
		select {
		case resp, ok := <-rh.ResponseC:
			if !ok {
				log.Debug().Msg("ResponseHandler: Summarizing results and exiting")
				err := rh.finalizeResponseStats(start, &totalRunTime, &runSummary, epRunSummary)
				if err != nil {
					log.Error().Err(err)
					close(rh.DoneC)
					return
				}
				rsjson, err := json.Marshal(runSummary)
				if err != nil {
					log.Error().Err(err).Msgf("error marshaling RunSummary into string: %+v.\n", runSummary)
					close(rh.DoneC)
					return
				}

				fmt.Printf("%s\n", string(rsjson))
				close(rh.DoneC)
				return
			}

			accumulateResponseStats(resp, &totalRunTime, &runSummary, epRunSummary)

		}
	}
}

func (rh ResponseHandler) finalizeResponseStats(start time.Time, totalRunTime *time.Duration,
	runSummary *RunSummary, epRunSummary map[string]*EndpointSummary) error {

	runSummary.RunDuration = time.Since(start)
	runSummary.RunDurationStr = runSummary.RunDuration.String()
	runSummary.RqstStats.TotalRequestDurationStr = totalRunTime.String()
	runSummary.RqstStats.MaxRqstDurationStr = runSummary.RqstStats.MaxRqstDuration.String()
	runSummary.RqstStats.MinRqstDurationStr = runSummary.RqstStats.MinRqstDuration.String()
	runSummary.RqstStats.AvgRqstDuration = time.Duration(0)
	if runSummary.RqstStats.TotalRqsts > 0 {
		runSummary.RqstStats.AvgRqstDuration = *totalRunTime / time.Duration(runSummary.RqstStats.TotalRqsts)
	}
	runSummary.RqstStats.AvgRqstDurationStr = runSummary.RqstStats.AvgRqstDuration.String()

	runSummary.RqstRatePerSec = (float64(runSummary.RqstStats.TotalRqsts) / float64(runSummary.RunDuration)) * float64(time.Second)
	runSummary.EndpointRunSummary = epRunSummary

	for _, epSumm := range epRunSummary {
		epSumm.RqstStats.MaxRqstDurationStr = epSumm.RqstStats.MaxRqstDuration.String()
		epSumm.RqstStats.MinRqstDurationStr = epSumm.RqstStats.MinRqstDuration.String()
		epSumm.RqstStats.AvgRqstDurationStr = "0s"
		if epSumm.RqstStats.TotalRqsts > 0 {
			epSumm.RqstStats.AvgRqstDuration = (epSumm.RqstStats.TotalRequestDuration / time.Duration(epSumm.RqstStats.TotalRqsts))
			epSumm.RqstStats.AvgRqstDurationStr = epSumm.RqstStats.AvgRqstDuration.String()
		}
		epSumm.RqstStats.TotalRequestDurationStr = epSumm.RqstStats.TotalRequestDuration.String()
		log.Debug().Msgf("EndpointSummary: %+v", epSumm)
	}

	return nil
}

func accumulateResponseStats(resp Response, totalRunTime *time.Duration, runSummary *RunSummary, epRunSummary map[string]*EndpointSummary) {
	runSummary.RqstStats.TotalRqsts++
	runSummary.RqstStats.TotalRequestDuration += resp.RequestDuration
	*totalRunTime = *totalRunTime + resp.RequestDuration
	if resp.RequestDuration > runSummary.RqstStats.MaxRqstDuration {
		runSummary.RqstStats.MaxRqstDuration = resp.RequestDuration
	}
	if resp.RequestDuration < runSummary.RqstStats.MinRqstDuration {
		runSummary.RqstStats.MinRqstDuration = resp.RequestDuration
	}

	var epStatusCount map[string]int
	epStatusCount, found := runSummary.EndpointOverviewSummary[resp.Endpoint.URL]
	if !found {
		runSummary.EndpointOverviewSummary[resp.Endpoint.URL] = make(map[string]int)
		epStatusCount = runSummary.EndpointOverviewSummary[resp.Endpoint.URL]
	}
	epStatusCount[resp.Endpoint.Method]++

	var epSumm *EndpointSummary
	epSumm, found = epRunSummary[resp.Endpoint.URL]
	if !found {
		epSumm = &EndpointSummary{
			URL:                  resp.Endpoint.URL,
			HTTPMethodStatusDist: make(map[string]map[int]int),
			RqstStats: RqstStats{
				MaxRqstDuration: -1,
				MinRqstDuration: time.Duration(math.MaxInt64),
			},
		}
		epRunSummary[resp.Endpoint.URL] = epSumm
	}

	epSumm.RqstStats.TotalRqsts++
	epSumm.RqstStats.TotalRequestDuration = epSumm.RqstStats.TotalRequestDuration + resp.RequestDuration

	if resp.RequestDuration > epSumm.RqstStats.MaxRqstDuration {
		epSumm.RqstStats.MaxRqstDuration = resp.RequestDuration
	}
	if resp.RequestDuration < epSumm.RqstStats.MinRqstDuration {
		epSumm.RqstStats.MinRqstDuration = resp.RequestDuration
	}

	_, ok := epSumm.HTTPMethodStatusDist[resp.Endpoint.Method]
	if !ok {
		epSumm.HTTPMethodStatusDist[resp.Endpoint.Method] = make(map[int]int)
		epSumm.HTTPMethodStatusDist[resp.Endpoint.Method][resp.HTTPStatus] = 0 // This is correct. It'll be incremented below
	}
	epSumm.HTTPMethodStatusDist[resp.Endpoint.Method][resp.HTTPStatus]++

}
