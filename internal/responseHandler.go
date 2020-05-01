package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/rs/zerolog/log"
)

// EndpointSummary is used to report an overview of the results of
// a load test run for a given endpoint.
type EndpointSummary struct {
	totalDuration time.Duration
	// URL is the endpoint URL
	URL string
	// Method is the HTTP Method (e.g., GET, PUT, POST, DELETE)
	Method string
	// HTTPStatusDist is a map of HTTP Status (e.g., 200, 201, 404, etc)
	// to the number of occurrences (value) for a given status (key)
	HTTPStatusDist map[int]int
	// EPRunStats are the runtime stats pertinent to a specific endpoint
	EPRunStats *Stats
}

// Stats is used to report detailed statistics from a load
// test run
type Stats struct {
	// TotalRqsts is the overall number of requests made during the run
	TotalRqsts int64
	// TotalDuration is the overall run duration in seconds
	TotalDuration string
	// MaxRqstDuration is the longest request duration in microseconds
	maxRqstDuration time.Duration
	MaxRqstDuration string
	// MinRqstDuration is the smallest request duration in microseconds
	minRqstDuration time.Duration
	MinRqstDuration string
	// AvgRqstDuration is the average duration of a request in microseconds
	AvgRqstDuration string
}

// RunSummary is used to report an overview of the results of a
// load test run
type RunSummary struct {
	// RqstRatePerSec is the overall request rate per second
	// rounded to the nearest integer
	RqstRatePerSec float64
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
	// OverallRunStats is a summary of runtime statistics
	// at an overall level
	OverallRunStats Stats
	// EndpointRunSummary is the per endpoint summary of results keyed by URL
	EndpointRunSummary map[string]*EndpointSummary
}

// ResponseHandler is responsible for accepting, summarizing, and reporting
// on the overall load test results.
type ResponseHandler struct {
	Ctx       context.Context
	ResponseC chan Response
}

// Start begins the process of accepting responses. It expects to be run as a goroutine
func (rh ResponseHandler) Start() {
	log.Debug().Msg("ResponseHandler starting")
	epRunSummary := make(map[string]*EndpointSummary)
	stats := Stats{maxRqstDuration: -1, minRqstDuration: time.Duration(math.MaxInt64)}

	var totalDurationSummary time.Duration

	for {
		select {
		case <-rh.Ctx.Done():
			stats.TotalDuration = totalDurationSummary.String()
			stats.MaxRqstDuration = stats.maxRqstDuration.String()
			stats.MinRqstDuration = stats.minRqstDuration.String()
			avgRqstDuration := time.Duration(0)
			if stats.TotalRqsts > 0 {
				avgRqstDuration = totalDurationSummary / time.Duration(stats.TotalRqsts)
			}
			stats.AvgRqstDuration = avgRqstDuration.String()

			for _, epSumm := range epRunSummary {
				epSumm.EPRunStats.MaxRqstDuration = epSumm.EPRunStats.maxRqstDuration.String()
				epSumm.EPRunStats.MinRqstDuration = epSumm.EPRunStats.minRqstDuration.String()
				epSumm.EPRunStats.AvgRqstDuration = "0s"
				if epSumm.EPRunStats.TotalRqsts > 0 {
					epSumm.EPRunStats.AvgRqstDuration = (epSumm.totalDuration / time.Duration(epSumm.EPRunStats.TotalRqsts)).String()
				}
				epSumm.EPRunStats.TotalDuration = epSumm.totalDuration.String()
				log.Debug().Msgf("EndpointSummary: %+v", epSumm)
				log.Debug().Msgf("\tEPunStatus: %+v\n", *epSumm.EPRunStats)
			}

			runSummary := RunSummary{
				// Very short run times can result in a 'RqstRatePerSec' being zero due to rounding errors
				RqstRatePerSec:     float64(stats.TotalRqsts/int64(totalDurationSummary)) * float64(time.Second),
				OverallRunStats:    stats,
				EndpointRunSummary: epRunSummary,
			}
			// fmt.Printf("\n\nDEBUG:\tFINAL RunSummary: %+v\n", runSummary)
			// fmt.Printf("\tEndpointRunSummary: %+v\n", epRunSummary)
			// for _, eps := range epRunSummary {
			// 	fmt.Printf("\t\tEP Summary: %+v\n", eps)
			// }

			rsjson, err := json.Marshal(runSummary)
			if err != nil {
				fmt.Printf("error marshaling RunSummary into string: %+v. Error: %s\n", runSummary, err)
				return
			}

			// fmt.Printf("Run Summary:\n\n")
			fmt.Printf("%s\n", rsjson)
			return
		case resp := <-rh.ResponseC:
			stats.TotalRqsts++
			totalDurationSummary = totalDurationSummary + resp.RequestDuration
			if resp.RequestDuration > stats.maxRqstDuration {
				stats.maxRqstDuration = resp.RequestDuration
			}
			if resp.RequestDuration < stats.minRqstDuration {
				stats.minRqstDuration = resp.RequestDuration
			}

			var epSumm *EndpointSummary
			var ok bool
			epSumm, ok = epRunSummary[resp.Endpoint.URL]
			if !ok {
				epSumm = &EndpointSummary{
					URL:            resp.Endpoint.URL,
					Method:         resp.Endpoint.Method,
					HTTPStatusDist: make(map[int]int),
					EPRunStats: &Stats{
						maxRqstDuration: -1,
						minRqstDuration: time.Duration(math.MaxInt64),
					},
				}
				epRunSummary[resp.Endpoint.URL] = epSumm
			}

			epSumm.EPRunStats.TotalRqsts++
			epSumm.totalDuration = epSumm.totalDuration + resp.RequestDuration
			// fmt.Printf("\n\n\t\t!!!!!!!!! epSumm.totalDuration: %d !!!!!!!!!!!!\n\n", epSumm.totalDuration)

			if resp.RequestDuration > epSumm.EPRunStats.maxRqstDuration {
				epSumm.EPRunStats.maxRqstDuration = resp.RequestDuration
			}
			if resp.RequestDuration < epSumm.EPRunStats.minRqstDuration {
				epSumm.EPRunStats.minRqstDuration = resp.RequestDuration
			}

			_, ok = epSumm.HTTPStatusDist[resp.HTTPStatus]
			if !ok {
				epSumm.HTTPStatusDist[resp.HTTPStatus] = 0
			}
			epSumm.HTTPStatusDist[resp.HTTPStatus]++

			// fmt.Printf("DEBUG:\tEndpointSummary: %+v\n", epSumm)
			// fmt.Printf("\tEPRunStatus: %+v\n", *epSumm.EPRunStats)
		}
	}
}
