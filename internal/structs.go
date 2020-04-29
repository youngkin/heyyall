package internal

import (
	"time"

	"github.com/youngkin/heyyall/api"
)

// Response contains information describing the results
// of a request to a specific endpoint
type Response struct {
	HTTPStatus      int
	Endpoint        api.Endpoint
	RequestDuration time.Duration
}

// Request contains the information needed to execute a request
// to an endpoint and return the response.
type Request struct {
	EP        api.Endpoint
	ResponseC chan Response
}

// EndpointSummary is used to report an overview of the results of
// a load test run for a given endpoint.
type EndpointSummary struct {
	// URL is the endpoint URL
	URL string
	// Method is the HTTP Method (e.g., GET, PUT, POST, DELETE)
	Method string
	// EPRunStats are the runtime stats pertinent to a specific endpoint
	EPRunStats Stats
}

// Stats is used to report detailed statistics from a load
// test run
type Stats struct {
	// TotalRqsts is the overall number of requests made during the run
	TotalRqsts int
	// TotalDuration is the overall run duration
	TotalDuration time.Duration
	// RqstRatePerSec is the overall request rate per second
	// rounded to the nearest integer
	RqstRatePerSec int
	// MaxRqstRatePerSec is the maximum request rate per second
	// over 1/10th of the run duration or number of requests
	MaxRqstRatePerSec int
	// MinRqstRatePerSec is the maximum request rate per second
	// over 1/10th of the run duration or number of requests
	MinRqstRatePerSec int
	// AvgRqstDuration is the average duration of a request in microseconds
	AvgRqstDuration int
	// MaxRqstDuration is the maximum request rate per second
	// over 1/10th of the run duration or number of requests
	MaxRqstDuration int
	// MinRqstDuration is the maximum request rate per second
	// over 1/10th of the run duration or number of requests
	MinRqstDuration int
	// ResponseDistribution is distribution of response times. There will be
	// 11 bucket; 10 microseconds or less, between 10us and 100us,
	// 100us and 1ms, 1ms to 10ms, 10ms to 100ms, 100ms to 1s, 1s to 1.1s,
	// 1.1s to 1.5s, 1.5s to 1.8s, 1.8s to 2.5s, 2.5s and above
	ResponseDistribution map[float32]int
	// HTTPStatusDistribution is the distribution of HTTP response statuses
	HTTPStatusDistribution map[string]int
}

// RunSummary is used to report an overview of the results of a
// load test run
type RunSummary struct {
	// OverallRunStats is a summary of runtime statistics
	// at an overall level
	OverallRunStats Stats
	// EndpointRunSummary is the per endpoint summary of results
	EndpointRunSummary []EndpointSummary
}
