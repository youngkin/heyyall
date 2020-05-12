// Copyright (c) 2020 Richard Youngkin. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package internal

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/youngkin/heyyall/api"
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
	// NormalizedMaxRqstDuration is the longest request duration rejecting outlier
	// durations more than 'x' times the MinRqstDuration
	NormalizedMaxRqstDuration time.Duration
	// NormalizedMaxRqstDurationStr is a string representation of NormalizedMaxRqstDuration
	NormalizedMaxRqstDurationStr string
	// MinRqstDuration is the smallest request duration for an endpoint
	MinRqstDuration time.Duration
	// MinRqstDurationStr is a string representation of MinRqstDuration
	MinRqstDurationStr string
	// AvgRqstDuration is the average duration of a request for an endpoint
	AvgRqstDuration time.Duration
	// AvgRqstDurationStr is the average duration of a request in microseconds
	AvgRqstDurationStr string
}

// EndpointDetail is used to report an overview of the results of
// a load test run for a given endpoint.
type EndpointDetail struct {
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

// RunResults is used to report an overview of the results of a
// load test run
type RunResults struct {
	// RunSummary is a roll-up of the detailed run results
	RunSummary RunSummary
	// EndpointSummary describes how often each endpoint was called.
	// It is a map keyed by URL of a map keyed by HTTP verb with a value of
	// number of requests. So it's a summary of how often each HTTP verb
	// was called on each endpoint.
	EndpointSummary map[string]map[string]int
	// EndpointDetails is the per endpoint summary of results keyed by URL
	EndpointDetails map[string]*EndpointDetail `json:",omitempty"`
}

// RunSummary is a roll-up of the detailed run results
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
}

// ResponseHandler is responsible for accepting, summarizing, and reporting
// on the overall load test results.
type ResponseHandler struct {
	ReportDetail  ReportDetail
	ResponseC     chan Response
	DoneC         chan struct{}
	NumRqsts      int
	NormFactor    int
	timingResults []time.Duration
	histogram     map[float64]int
}

// Start begins the process of accepting responses. It expects to be run as a goroutine.
func (rh *ResponseHandler) Start() {
	log.Debug().Msg("ResponseHandler starting")

	rh.timingResults = make([]time.Duration, 0, int(math.Min(float64(rh.NumRqsts), float64(api.MaxRqsts))))
	epRunSummary := make(map[string]*EndpointDetail)
	runSummary := RunSummary{RqstStats: RqstStats{MaxRqstDuration: time.Duration(-1), MinRqstDuration: time.Duration(math.MaxInt64)}}
	runResults := RunResults{RunSummary: runSummary}
	runResults.EndpointSummary = make(map[string]map[string]int)

	start := time.Now()
	var totalRunTime time.Duration

	for {
		select {
		case resp, ok := <-rh.ResponseC:
			if !ok {
				defer close(rh.DoneC)
				log.Debug().Msg("ResponseHandler: Summarizing results and exiting")
				err := rh.finalizeResponseStats(start, &totalRunTime, &runResults, epRunSummary)
				if err != nil {
					log.Error().Err(err)
					return
				}

				min, max := rh.generateHistogram(&runResults)

				rsjson, err := json.MarshalIndent(runResults, "    ", "  ")
				if err != nil {
					log.Error().Err(err).Msgf("error marshaling RunSummary into string: %+v.\n", runResults)
					return
				}

				fmt.Printf("\nResponse Time Histogram (seconds):\n")
				fmt.Println(rh.generateHistogramString(min, max))

				fmt.Printf("\n\nRun Results:\n")
				fmt.Printf("%s\n", string(rsjson[2:len(rsjson)-1]))
				return
			}

			rh.accumulateResponseStats(resp, &totalRunTime, &runResults, epRunSummary)

		}
	}
}

func (rh *ResponseHandler) finalizeResponseStats(start time.Time, totalRunTime *time.Duration,
	runResults *RunResults, epRunSummary map[string]*EndpointDetail) error {

	runResults.RunSummary.RunDuration = time.Since(start)
	runResults.RunSummary.RunDurationStr = runResults.RunSummary.RunDuration.String()
	runResults.RunSummary.RqstStats.TotalRequestDurationStr = totalRunTime.String()
	runResults.RunSummary.RqstStats.MaxRqstDurationStr = runResults.RunSummary.RqstStats.MaxRqstDuration.String()
	runResults.RunSummary.RqstStats.MinRqstDurationStr = runResults.RunSummary.RqstStats.MinRqstDuration.String()
	runResults.RunSummary.RqstStats.AvgRqstDuration = time.Duration(0)
	if runResults.RunSummary.RqstStats.TotalRqsts > 0 {
		runResults.RunSummary.RqstStats.AvgRqstDuration = *totalRunTime / time.Duration(runResults.RunSummary.RqstStats.TotalRqsts)
	}
	runResults.RunSummary.RqstStats.AvgRqstDurationStr = runResults.RunSummary.RqstStats.AvgRqstDuration.String()

	runResults.RunSummary.RqstRatePerSec = (float64(runResults.RunSummary.RqstStats.TotalRqsts) / float64(runResults.RunSummary.RunDuration)) * float64(time.Second)

	if rh.ReportDetail == Long {
		runResults.EndpointDetails = epRunSummary

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
	}

	return nil
}

func (rh *ResponseHandler) accumulateResponseStats(resp Response, totalRunTime *time.Duration, runResults *RunResults, epRunSummary map[string]*EndpointDetail) {
	rh.timingResults = append(rh.timingResults, resp.RequestDuration)
	runResults.RunSummary.RqstStats.TotalRqsts++
	runResults.RunSummary.RqstStats.TotalRequestDuration += resp.RequestDuration
	*totalRunTime = *totalRunTime + resp.RequestDuration

	if resp.RequestDuration > runResults.RunSummary.RqstStats.MaxRqstDuration {
		runResults.RunSummary.RqstStats.MaxRqstDuration = resp.RequestDuration
	}
	if resp.RequestDuration < runResults.RunSummary.RqstStats.MinRqstDuration {
		runResults.RunSummary.RqstStats.MinRqstDuration = resp.RequestDuration
	}

	var epStatusCount map[string]int
	epStatusCount, found := runResults.EndpointSummary[resp.Endpoint.URL]
	if !found {
		runResults.EndpointSummary[resp.Endpoint.URL] = make(map[string]int)
		epStatusCount = runResults.EndpointSummary[resp.Endpoint.URL]
	}
	epStatusCount[resp.Endpoint.Method]++

	var epSumm *EndpointDetail
	epSumm, found = epRunSummary[resp.Endpoint.URL]
	if !found {
		epSumm = &EndpointDetail{
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

// generateHistogram populates the histogram map, a map keyed by a float64 that's
// taken from the result set, referencing the number of observations in the 'range'
// of that number. It returns the min and max values for the histogram, i.e., the
// min and max number of observations in the histogram.
func (rh *ResponseHandler) generateHistogram(runResults *RunResults) (int, int) {
	numBins := calcNumBinsSturgesMethod(len(rh.timingResults))
	// numBins := calcNumBinsRiceMethod(len(rh.timingResults))
	runResults.RunSummary.RqstStats.NormalizedMaxRqstDuration = time.Duration(rh.NormFactor) * runResults.RunSummary.RqstStats.MinRqstDuration
	runResults.RunSummary.RqstStats.NormalizedMaxRqstDurationStr = runResults.RunSummary.RqstStats.NormalizedMaxRqstDuration.String()
	timeRange := runResults.RunSummary.RqstStats.MaxRqstDuration - runResults.RunSummary.RqstStats.MinRqstDuration
	if rh.NormFactor > 1 {
		timeRange = runResults.RunSummary.RqstStats.NormalizedMaxRqstDuration - runResults.RunSummary.RqstStats.MinRqstDuration
	}
	binWidth := float64(timeRange) / float64(numBins)
	if binWidth == 0 {
		binWidth = float64(runResults.RunSummary.RqstStats.MinRqstDuration + 1)
	}

	rh.histogram = make(map[float64]int)
	bins := make([]float64, 0, numBins)

	for i := 1; i <= numBins; i++ {
		rh.histogram[float64(i)*binWidth] = 0
		bins = append(bins, float64(i)*binWidth)
	}

	maxBinVal, minBinVal := 0, math.MaxInt32
	for _, obser := range rh.timingResults {
		// TODO: Might be able to get this to O(n*Log(n))) if did a binary search on bins as it's sorted
		for _, bin := range bins {
			if float64(obser) < bin {
				rh.histogram[bin]++
				if rh.histogram[bin] > maxBinVal {
					maxBinVal = rh.histogram[bin]
				}
				if rh.histogram[bin] < minBinVal {
					minBinVal = rh.histogram[bin]
				}
				break
			}
		}
	}

	if rh.NormFactor > 1 {
		// If the histogram is being normalized, pick up all the observations greater than largest bin's key
		// into a single bin. This will at least show how many observations occurred between 'largestBinKey'
		// and the MaxRqstDuration.
		largestBinKey := binWidth * float64(numBins)
		var tailBin int
		for _, obser := range rh.timingResults {
			if float64(obser) >= largestBinKey {
				tailBin++
			}
		}
		rh.histogram[float64(runResults.RunSummary.RqstStats.MaxRqstDuration)] = tailBin
		maxBinVal = int(math.Max(float64(tailBin), float64(maxBinVal)))
	}

	return minBinVal, maxBinVal
}

func (rh *ResponseHandler) printHistogram(min, max int) {
	barUnit := ">"
	// barUnit := "■"

	keys := make([]float64, 0, len(rh.histogram))
	for k := range rh.histogram {
		keys = append(keys, k)
	}
	sort.Float64s(keys)

	fmt.Printf("\tLatency\t\tNumber of Observations\n")
	fmt.Printf("\t-------\t\t----------------------\n")
	var sb strings.Builder
	for _, key := range keys {
		cnt := rh.histogram[key]
		barLen := ((cnt * 100) + (max / 2)) / max
		for i := 0; i < barLen; i++ {
			sb.WriteString(barUnit)
		}
		fmt.Printf("\t[%6.3f]\t%7v\t%s\n", key/float64(time.Second), cnt, sb.String())
		sb.Reset()
	}
}

func (rh *ResponseHandler) generateHistogramString(min, max int) string {
	barUnit := ">"
	// barUnit := "■"

	keys := make([]float64, 0, len(rh.histogram))
	for k := range rh.histogram {
		keys = append(keys, k)
	}
	sort.Float64s(keys)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\tLatency\t\tNumber of Observations\n"))
	sb.WriteString(fmt.Sprintf("\t-------\t\t----------------------\n"))
	for _, key := range keys {
		var sbBar strings.Builder
		cnt := rh.histogram[key]
		barLen := ((cnt * 100) + (max / 2)) / max
		for i := 0; i < barLen; i++ {
			sbBar.WriteString(barUnit)
		}
		sb.WriteString(fmt.Sprintf("\t[%6.3f]\t%7v\t%s\n", key/float64(time.Second), cnt, sbBar.String()))
		sbBar.Reset()
	}
	return sb.String()
}

func calcNumBinsSturgesMethod(numObservations int) int {
	return int(math.Ceil(math.Log2(float64(numObservations) + 1)))
}

// Results in a lot more bins at higher numbers than Sturges. For example, at 1,000,000
// observations Sturges generates 21 buckets to Rice's 200
func calcNumBinsRiceMethod(numObservations int) int {
	return int(math.Ceil(2 * math.Cbrt(float64(numObservations))))
}
