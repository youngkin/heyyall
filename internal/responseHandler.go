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

// Response contains information describing the results
// of a request to a specific endpoint
type Response struct {
	HTTPStatus           int
	Endpoint             api.Endpoint
	RequestDuration      time.Duration
	DNSLookupDuration    time.Duration
	TCPConnDuration      time.Duration
	RoundTripDuration    time.Duration
	TLSHandshakeDuration time.Duration
}

// ResponseHandler is responsible for accepting, summarizing, and reporting
// on the overall load test results.
type ResponseHandler struct {
	OutputType OutputType
	ResponseC  chan Response
	ProgressC  chan interface{}
	DoneC      chan interface{}
	NumRqsts   int
	NormFactor int
	// histogram contains a count of observations that are <= to the value of the key.
	// The key is a number that represents response duration.
	histogram map[float64]int
}

// Start begins the process of accepting responses. It expects to be run as a goroutine.
func (rh *ResponseHandler) Start() {
	log.Debug().Msg("ResponseHandler starting")

	epRunSummary := make(map[string]*api.EndpointDetail)
	runSummary := api.RunSummary{RqstStats: api.RqstStats{MaxRqstDurationNanos: time.Duration(-1), MinRqstDurationNanos: time.Duration(math.MaxInt64)}}
	runResults := api.RunResults{RunSummary: runSummary}
	runResults.EndpointSummary = make(map[string]map[string]int)

	start := time.Now()
	var totalRunTime time.Duration
	responses := make([]Response, 0, 10)

	for {
		select {
		case resp, ok := <-rh.ResponseC:
			if !ok {
				defer close(rh.DoneC)
				log.Debug().Msg("ResponseHandler: Summarizing results and exiting")

				for _, r := range responses {
					rh.accumulateResponseStats(r, &totalRunTime, &runResults, epRunSummary)
					runResults.RunSummary.DNSLookupNanos = append(runResults.RunSummary.DNSLookupNanos, r.DNSLookupDuration)
					runResults.RunSummary.TCPConnSetupNanos = append(runResults.RunSummary.TCPConnSetupNanos, r.TCPConnDuration)
					runResults.RunSummary.RqstRoundTripNanos = append(runResults.RunSummary.RqstRoundTripNanos, r.RoundTripDuration)
					runResults.RunSummary.TLSHandshakeNanos = append(runResults.RunSummary.TLSHandshakeNanos, r.TLSHandshakeDuration)
				}

				err := rh.finalizeResponseStats(start, &totalRunTime, &runResults, epRunSummary)
				if err != nil {
					log.Error().Err(err)
					return
				}

				if rh.OutputType == Text {
					fmt.Println("")
					printRunSummary(runResults.RunSummary)

					fmt.Println("")
					printRqstLatency(runResults.RunSummary.RqstStats)

					min, max := rh.generateHistogram(&runResults)
					fmt.Printf("\nRequest Latency Histogram (secs):\n")
					fmt.Println(rh.generateHistogramString(min, max))

					fmt.Println("")
					printEndpointDetails(runResults.EndpointDetails)

					fmt.Println("")
					printNetworkDetails(runResults.RunSummary)

					return
				}

				rsjson, err := json.MarshalIndent(runResults, "    ", "  ")
				if err != nil {
					log.Error().Err(err).Msgf("error marshaling RunSummary into string: %+v.\n", runResults)
					return
				}
				fmt.Printf("%s\n", string(rsjson[2:len(rsjson)-1]))

				return
			}

			responses = append(responses, resp)
			// If rh.NumRqsts > 0 then the load test is being limited by total number of requests sent, not time.
			// In this case each received request represents progress that must be recorded.
			if rh.NumRqsts > 0 {
				rh.ProgressC <- struct{}{}
			}
		}
	}
}

func (rh *ResponseHandler) finalizeResponseStats(start time.Time, totalRunTime *time.Duration,
	runResults *api.RunResults, epRunSummary map[string]*api.EndpointDetail) error {

	runResults.RunSummary.RunDurationNanos = time.Since(start)
	runResults.RunSummary.RqstStats.AvgRqstDurationNanos = time.Duration(0)
	if runResults.RunSummary.RqstStats.TotalRqsts > 0 {
		runResults.RunSummary.RqstStats.AvgRqstDurationNanos = *totalRunTime / time.Duration(runResults.RunSummary.RqstStats.TotalRqsts)
	}

	runResults.RunSummary.RqstRatePerSec = (float64(runResults.RunSummary.RqstStats.TotalRqsts) / float64(runResults.RunSummary.RunDurationNanos)) * float64(time.Second)

	runResults.EndpointDetails = epRunSummary

	for _, epDetail := range epRunSummary {
		for _, methodRqstStats := range epDetail.HTTPMethodRqstStats {
			if methodRqstStats.TotalRqsts > 0 {
				methodRqstStats.AvgRqstDurationNanos = (methodRqstStats.TotalRequestDurationNanos / time.Duration(methodRqstStats.TotalRqsts))
			}
			log.Debug().Msgf("EndpointSummary: %+v", epDetail)
		}
	}

	return nil
}

func (rh *ResponseHandler) accumulateResponseStats(resp Response, totalRunTime *time.Duration,
	runResults *api.RunResults, epRunSummary map[string]*api.EndpointDetail) {

	runResults.RunSummary.RqstStats.TimingResultsNanos = append(runResults.RunSummary.RqstStats.TimingResultsNanos, resp.RequestDuration)
	runResults.RunSummary.RqstStats.TotalRqsts++
	runResults.RunSummary.RqstStats.TotalRequestDurationNanos += resp.RequestDuration
	*totalRunTime = *totalRunTime + resp.RequestDuration

	if resp.RequestDuration > runResults.RunSummary.RqstStats.MaxRqstDurationNanos {
		runResults.RunSummary.RqstStats.MaxRqstDurationNanos = resp.RequestDuration
	}
	if resp.RequestDuration < runResults.RunSummary.RqstStats.MinRqstDurationNanos {
		runResults.RunSummary.RqstStats.MinRqstDurationNanos = resp.RequestDuration
	}

	var epStatusCount map[string]int
	epStatusCount, ok := runResults.EndpointSummary[resp.Endpoint.URL]
	if !ok {
		runResults.EndpointSummary[resp.Endpoint.URL] = make(map[string]int)
		epStatusCount = runResults.EndpointSummary[resp.Endpoint.URL]
	}
	epStatusCount[resp.Endpoint.Method]++

	var epDetail *api.EndpointDetail
	epDetail, ok = epRunSummary[resp.Endpoint.URL]
	if !ok {
		epDetail = &api.EndpointDetail{
			URL:                  resp.Endpoint.URL,
			HTTPMethodStatusDist: make(map[string]map[int]int),
			HTTPMethodRqstStats:  make(map[string]*api.RqstStats),
		}
		epRunSummary[resp.Endpoint.URL] = epDetail
	}

	methodRqstStats, ok := epDetail.HTTPMethodRqstStats[resp.Endpoint.Method]
	if !ok {
		epDetail.HTTPMethodRqstStats[resp.Endpoint.Method] = &api.RqstStats{
			MaxRqstDurationNanos: -1,
			MinRqstDurationNanos: time.Duration(math.MaxInt64),
		}
		methodRqstStats = epDetail.HTTPMethodRqstStats[resp.Endpoint.Method]
	}

	methodRqstStats.TotalRqsts++
	methodRqstStats.TotalRequestDurationNanos = methodRqstStats.TotalRequestDurationNanos + resp.RequestDuration

	if resp.RequestDuration > methodRqstStats.MaxRqstDurationNanos {
		methodRqstStats.MaxRqstDurationNanos = resp.RequestDuration
	}
	if resp.RequestDuration < methodRqstStats.MinRqstDurationNanos {
		methodRqstStats.MinRqstDurationNanos = resp.RequestDuration
	}
	methodRqstStats.TimingResultsNanos = append(methodRqstStats.TimingResultsNanos, resp.RequestDuration)

	_, ok = epDetail.HTTPMethodStatusDist[resp.Endpoint.Method]
	if !ok {
		epDetail.HTTPMethodStatusDist[resp.Endpoint.Method] = make(map[int]int)
		epDetail.HTTPMethodStatusDist[resp.Endpoint.Method][resp.HTTPStatus] = 0 // This is correct. It'll be incremented below
	}
	epDetail.HTTPMethodStatusDist[resp.Endpoint.Method][resp.HTTPStatus]++

}

// generateHistogram populates the histogram map, a map keyed by a float64 that's
// taken from the result set, referencing the number of observations in the 'range'
// of that number. It returns the min and max values for the histogram, i.e., the
// min and max number of observations in the histogram.
func (rh *ResponseHandler) generateHistogram(runResults *api.RunResults) (minBinCount, maxBinCount int) {
	numBins := calcNumBinsSturgesMethod(len(runResults.RunSummary.RqstStats.TimingResultsNanos))
	// numBins := calcNumBinsRiceMethod(len(runResults.RunSummary.RqstStats.TimingResultsNanos))
	runResults.RunSummary.RqstStats.NormalizedMaxRqstDurationNanos = time.Duration(rh.NormFactor) * runResults.RunSummary.RqstStats.MinRqstDurationNanos

	binWidth := float64(runResults.RunSummary.RqstStats.MaxRqstDurationNanos) / float64(numBins)
	if rh.NormFactor > 1 {
		maxNormDur := time.Duration(math.Min(float64(runResults.RunSummary.RqstStats.MaxRqstDurationNanos),
			float64(runResults.RunSummary.RqstStats.NormalizedMaxRqstDurationNanos)))
		binWidth = float64(maxNormDur) / float64(numBins)
	}
	rh.histogram = make(map[float64]int, numBins)
	binValues := make([]float64, 0, numBins)

	for i := 1; i <= numBins; i++ {
		rh.histogram[float64(i)*binWidth] = 0
		binValues = append(binValues, float64(i)*binWidth)
	}

	maxBinCount, minBinCount = 0, math.MaxInt32

	// NOTE: this algorithm depends on 'binValues' being sorted in ascending order. This ensures
	// that the observation gets assigned to the correct bin, i.e., the lowest bin value that is
	// >= to the observation. 'binValues' is a slice whose values are appended in ascending order,
	// so it is already sorted.
	for _, observation := range runResults.RunSummary.RqstStats.TimingResultsNanos {
		// TODO: Might be able to get this to O(n*Log(n))) if did a binary search on binKeys as it's sorted
		for _, binVal := range binValues {
			if float64(observation) <= binVal {
				rh.histogram[binVal]++
				if rh.histogram[binVal] > maxBinCount {
					maxBinCount = rh.histogram[binVal]
				}
				break
			}
		}
	}

	for _, v := range rh.histogram {
		if v < minBinCount {
			minBinCount = v
		}
	}

	if rh.NormFactor > 1 && runResults.RunSummary.RqstStats.NormalizedMaxRqstDurationNanos < runResults.RunSummary.RqstStats.MaxRqstDurationNanos {
		// If the histogram is being normalized, pick up all the observations greater than largest bin's key
		// into a single bin. This will show how many observations occurred between 'largestBinKey' and the
		// MaxRqstDuration.
		largestBinKey := binWidth * float64(numBins)
		var tailBinCount int
		for _, observation := range runResults.RunSummary.RqstStats.TimingResultsNanos {
			if float64(observation) > largestBinKey {
				tailBinCount++
			}
		}
		rh.histogram[float64(runResults.RunSummary.RqstStats.MaxRqstDurationNanos)] = tailBinCount
		maxBinCount = int(math.Max(float64(tailBinCount), float64(maxBinCount)))
		minBinCount = int(math.Min(float64(tailBinCount), float64(minBinCount)))
	}

	return minBinCount, maxBinCount
}

func (rh *ResponseHandler) generateHistogramString(min, max int) string {
	// barUnit := ">"
	barUnit := "❱"
	// barUnit := "■"
	// barUnit := "➤"
	// barUnit := "⭆"
	// barUnit := '➯'

	keys := make([]float64, 0, len(rh.histogram))
	for k := range rh.histogram {
		keys = append(keys, k)
	}
	sort.Float64s(keys)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\tLatency   Observations\n"))
	// sb.WriteString(fmt.Sprintf("\t--------  ----------------------\n"))
	for _, key := range keys {
		var sbBar strings.Builder
		cnt := rh.histogram[key]
		barLen := ((cnt * 100) + (max / 2)) / max
		for i := 0; i < barLen; i++ {
			sbBar.WriteString(barUnit)
		}
		sb.WriteString(fmt.Sprintf("\t[%4.4f] %7v\t%s\n", key/float64(time.Second), cnt, sbBar.String()))
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
