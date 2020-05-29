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
	// TotalRequestDurationStr is the string version of TotalRequestDuration
	TotalRequestDurationStr string
	// MaxRqstDurationStr is a string representation of MaxRqstDuration
	MaxRqstDurationStr string
	// NormalizedMaxRqstDurationStr is a string representation of NormalizedMaxRqstDuration
	NormalizedMaxRqstDurationStr string
	// MinRqstDurationStr is a string representation of MinRqstDuration
	MinRqstDurationStr string
	// AvgRqstDurationStr is the average duration of a request in microseconds
	AvgRqstDurationStr string
	// TotalRequestDuration is the sum of all request run durations
	TotalRequestDuration time.Duration
	// MaxRqstDuration is the longest request duration
	MaxRqstDuration time.Duration
	// NormalizedMaxRqstDuration is the longest request duration rejecting outlier
	// durations more than 'x' times the MinRqstDuration
	NormalizedMaxRqstDuration time.Duration
	// MinRqstDuration is the smallest request duration for an endpoint
	MinRqstDuration time.Duration
	// AvgRqstDuration is the average duration of a request for an endpoint
	AvgRqstDuration time.Duration
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
	// HTTPMethodRqstStats provides summary request statistics by HTTP Method. It is
	// map of RqstStats keyed by HTTP method.
	HTTPMethodRqstStats map[string]*RqstStats
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
	// DNSLookup records how long it took to resolve the hostname to an IP Address
	DNSLookup []time.Duration
	// TCPConnSetup records how long it took to setup the TCP connection
	TCPConnSetup []time.Duration
	// RqstRoundTrip records duration from the time the TCP connection was setup
	// until the response was received
	RqstRoundTrip []time.Duration
	// TLSHandshake records the time it took to complete the TLS negotiation with
	// the server. It's only meaningful for HTTPS connections
	TLSHandshake []time.Duration
}

// ResponseHandler is responsible for accepting, summarizing, and reporting
// on the overall load test results.
type ResponseHandler struct {
	ReportDetail ReportDetail
	ResponseC    chan Response
	DoneC        chan struct{}
	NumRqsts     int
	NormFactor   int
	// TimingResults contains the duration of each request.
	TimingResults []time.Duration
	// histogram contains a count of observations that are <= to the value of the key.
	// The key is a number that represents response duration.
	histogram map[float64]int
}

// Start begins the process of accepting responses. It expects to be run as a goroutine.
func (rh *ResponseHandler) Start() {
	log.Debug().Msg("ResponseHandler starting")

	rh.TimingResults = make([]time.Duration, 0, int(math.Min(float64(rh.NumRqsts), float64(api.MaxRqsts))))
	epRunSummary := make(map[string]*EndpointDetail)
	runSummary := RunSummary{RqstStats: RqstStats{MaxRqstDuration: time.Duration(-1), MinRqstDuration: time.Duration(math.MaxInt64)}}
	runResults := RunResults{RunSummary: runSummary}
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
					runResults.RunSummary.DNSLookup = append(runResults.RunSummary.DNSLookup, r.DNSLookupDuration)
					runResults.RunSummary.TCPConnSetup = append(runResults.RunSummary.TCPConnSetup, r.TCPConnDuration)
					runResults.RunSummary.RqstRoundTrip = append(runResults.RunSummary.RqstRoundTrip, r.RoundTripDuration)
					runResults.RunSummary.TLSHandshake = append(runResults.RunSummary.TLSHandshake, r.TLSHandshakeDuration)
				}

				err := rh.finalizeResponseStats(start, &totalRunTime, &runResults, epRunSummary)
				if err != nil {
					log.Error().Err(err)
					return
				}

				fmt.Println("")
				printRunSummary(runResults.RunSummary)

				fmt.Println("")
				printRqstSummary(rh)

				min, max := rh.generateHistogram(&runResults)
				fmt.Printf("\nRequest Duration Histogram (seconds):\n")
				fmt.Println(rh.generateHistogramString(min, max))

				fmt.Println("")
				printNetworkSummary(runResults.RunSummary)
				fmt.Println("")

				// if max != 0 {
				// 	fmt.Printf("\nResponse Time Histogram (seconds):\n")
				// 	fmt.Println(rh.generateHistogramString(min, max))
				// } else {
				// 	fmt.Println("\nUnable to generate Response Time Histogram.")
				// 	log.Error().Msg("'max' histogram bin value was 0, no histogram can be created")
				// }

				// fmt.Println("\n\nHTTP Trace Results (microsecs):")
				// fmt.Printf("\tDNS Lookup:        Min: %d Median: %d P90: %d P99:%d\n",
				// 	calcPMin(dnsResults), calcPMedian(dnsResults), calcP90(dnsResults), calcP99(dnsResults))
				// fmt.Printf("\tTCP Conn:          Min: %d Median: %d P90: %d P99:%d\n",
				// 	calcPMin(tcpResults), calcPMedian(tcpResults), calcP90(tcpResults), calcP99(tcpResults))
				// fmt.Printf("\tRoundTrip:         Min: %d Median: %d P90: %d P99:%d\n",
				// 	calcPMin(roundTripResults), calcPMedian(roundTripResults), calcP90(roundTripResults), calcP99(roundTripResults))
				// fmt.Printf("\tTLS Handshake:    Min: %d Median: %d P90: %d P99:%d\n",
				// 	calcPMin(TLSHandshakeResults), calcPMedian(TLSHandshakeResults), calcP90(TLSHandshakeResults), calcP99(TLSHandshakeResults))

				if rh.ReportDetail == Long {
					rsjson, err := json.MarshalIndent(runResults, "    ", "  ")
					if err != nil {
						log.Error().Err(err).Msgf("error marshaling RunSummary into string: %+v.\n", runResults)
						return
					}

					fmt.Printf("\n\nRun Results:\n")
					fmt.Printf("%s\n", string(rsjson[2:len(rsjson)-1]))
				}

				return
			}

			responses = append(responses, resp)
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

		for _, epDetail := range epRunSummary {
			for _, methodRqstStats := range epDetail.HTTPMethodRqstStats {
				methodRqstStats.MaxRqstDurationStr = methodRqstStats.MaxRqstDuration.String()
				methodRqstStats.MinRqstDurationStr = methodRqstStats.MinRqstDuration.String()
				methodRqstStats.AvgRqstDurationStr = "0s"
				if methodRqstStats.TotalRqsts > 0 {
					methodRqstStats.AvgRqstDuration = (methodRqstStats.TotalRequestDuration / time.Duration(methodRqstStats.TotalRqsts))
					methodRqstStats.AvgRqstDurationStr = methodRqstStats.AvgRqstDuration.String()
				}
				methodRqstStats.TotalRequestDurationStr = methodRqstStats.TotalRequestDuration.String()
				log.Debug().Msgf("EndpointSummary: %+v", epDetail)
			}
		}
	}

	return nil
}

func (rh *ResponseHandler) accumulateResponseStats(resp Response, totalRunTime *time.Duration, runResults *RunResults, epRunSummary map[string]*EndpointDetail) {
	rh.TimingResults = append(rh.TimingResults, resp.RequestDuration)
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
	epStatusCount, ok := runResults.EndpointSummary[resp.Endpoint.URL]
	if !ok {
		runResults.EndpointSummary[resp.Endpoint.URL] = make(map[string]int)
		epStatusCount = runResults.EndpointSummary[resp.Endpoint.URL]
	}
	epStatusCount[resp.Endpoint.Method]++

	var epDetail *EndpointDetail
	epDetail, ok = epRunSummary[resp.Endpoint.URL]
	if !ok {
		epDetail = &EndpointDetail{
			URL:                  resp.Endpoint.URL,
			HTTPMethodStatusDist: make(map[string]map[int]int),
			HTTPMethodRqstStats:  make(map[string]*RqstStats),
		}
		epRunSummary[resp.Endpoint.URL] = epDetail
	}

	methodRqstStats, ok := epDetail.HTTPMethodRqstStats[resp.Endpoint.Method]
	if !ok {
		epDetail.HTTPMethodRqstStats[resp.Endpoint.Method] = &RqstStats{
			MaxRqstDuration: -1,
			MinRqstDuration: time.Duration(math.MaxInt64),
		}
		methodRqstStats = epDetail.HTTPMethodRqstStats[resp.Endpoint.Method]
	}

	methodRqstStats.TotalRqsts++
	methodRqstStats.TotalRequestDuration = methodRqstStats.TotalRequestDuration + resp.RequestDuration

	if resp.RequestDuration > methodRqstStats.MaxRqstDuration {
		methodRqstStats.MaxRqstDuration = resp.RequestDuration
	}
	if resp.RequestDuration < methodRqstStats.MinRqstDuration {
		methodRqstStats.MinRqstDuration = resp.RequestDuration
	}

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
func (rh *ResponseHandler) generateHistogram(runResults *RunResults) (minBinCount, maxBinCount int) {
	numBins := calcNumBinsSturgesMethod(len(rh.TimingResults))
	// numBins := calcNumBinsRiceMethod(len(rh.timingResults))
	runResults.RunSummary.RqstStats.NormalizedMaxRqstDuration = time.Duration(rh.NormFactor) * runResults.RunSummary.RqstStats.MinRqstDuration
	runResults.RunSummary.RqstStats.NormalizedMaxRqstDurationStr = runResults.RunSummary.RqstStats.NormalizedMaxRqstDuration.String()

	binWidth := float64(runResults.RunSummary.RqstStats.MaxRqstDuration) / float64(numBins)
	if rh.NormFactor > 1 {
		maxNormDur := time.Duration(math.Min(float64(runResults.RunSummary.RqstStats.MaxRqstDuration),
			float64(runResults.RunSummary.RqstStats.NormalizedMaxRqstDuration)))
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
	for _, observation := range rh.TimingResults {
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

	if rh.NormFactor > 1 && runResults.RunSummary.RqstStats.NormalizedMaxRqstDuration < runResults.RunSummary.RqstStats.MaxRqstDuration {
		// If the histogram is being normalized, pick up all the observations greater than largest bin's key
		// into a single bin. This will show how many observations occurred between 'largestBinKey' and the
		// MaxRqstDuration.
		largestBinKey := binWidth * float64(numBins)
		var tailBinCount int
		for _, observation := range rh.TimingResults {
			if float64(observation) > largestBinKey {
				tailBinCount++
			}
		}
		rh.histogram[float64(runResults.RunSummary.RqstStats.MaxRqstDuration)] = tailBinCount
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
	sb.WriteString(fmt.Sprintf("\tLatency   Number of Observations\n"))
	sb.WriteString(fmt.Sprintf("\t--------  ----------------------\n"))
	for _, key := range keys {
		var sbBar strings.Builder
		cnt := rh.histogram[key]
		barLen := ((cnt * 100) + (max / 2)) / max
		for i := 0; i < barLen; i++ {
			sbBar.WriteString(barUnit)
		}
		sb.WriteString(fmt.Sprintf("\t[%6.3f]   %7v\t%s\n", key/float64(time.Second), cnt, sbBar.String()))
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
