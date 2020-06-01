// Copyright (c) 2020 Richard Youngkin. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package api

import "time"

// RqstStats contains a set of common runtime stats reported at both the
// Summary and Endpoint level
type RqstStats struct {
	// TimingResultsNanos contains the duration of each request.
	TimingResultsNanos []time.Duration
	// TotalRqsts is the overall number of requests made during the run
	TotalRqsts int64
	// TotalRequestDurationNanos is the sum of all request run durations
	TotalRequestDurationNanos time.Duration
	// MaxRqstDurationNanos is the longest request duration
	MaxRqstDurationNanos time.Duration
	// NormalizedMaxRqstDurationNanos is the longest request duration rejecting outlier
	// durations more than 'x' times the MinRqstDuration
	NormalizedMaxRqstDurationNanos time.Duration
	// MinRqstDurationNanos is the smallest request duration for an endpoint
	MinRqstDurationNanos time.Duration
	// AvgRqstDurationNanos is the average duration of a request for an endpoint
	AvgRqstDurationNanos time.Duration
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
	// RunDurationNanos is the wall clock duration of the test
	RunDurationNanos time.Duration

	// MaxRqstRatePerSec is the maximum request rate per second
	// over 1/10th of the run duration or number of requests
	//MaxRqstRatePerSec int
	// MinRqstRatePerSec is the maximum request rate per second
	// over 1/10th of the run duration or number of requests
	//MinRqstRatePerSec int

	// RqstStats is a summary of runtime statistics
	RqstStats RqstStats
	// DNSLookupNanos records how long it took to resolve the hostname to an IP Address
	DNSLookupNanos []time.Duration
	// TCPConnSetupNanos records how long it took to setup the TCP connection
	TCPConnSetupNanos []time.Duration
	// RqstRoundTripNanos records duration from the time the TCP connection was setup
	// until the response was received
	RqstRoundTripNanos []time.Duration
	// TLSHandshakeNanos records the time it took to complete the TLS negotiation with
	// the server. It's only meaningful for HTTPS connections
	TLSHandshakeNanos []time.Duration
}
