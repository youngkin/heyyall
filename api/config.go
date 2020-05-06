// Copyright (c) 2020 Richard Youngkin. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package api provides the public datastructures that can be used to
// create a runtime configuration file.
package api

// Endpoint contains the information needed to to send a request,
// in the desired proportion to total requests, to a given
// HTTP endpoint (e.g., someplace.com).
type Endpoint struct {
	// URL is the endpoint address
	URL string
	// Method is the HTTP Method
	Method string
	// RqstBody is the request data to be sent to the endpoint
	RqstBody string
	// RqstPercent is the relative weight of how often a request
	// to this endpoint will be made. It's a percent of all requests
	// to be made. As such the RqstPercent of all Endpoints in the
	// config must add to 100.
	RqstPercent int
	// NumRequests is the total number of requests to make. See
	// LoadTestConfig.RunDuration for the behavior when both
	// RunDuration and NumRequests are specified.
	NumRequests int
	// Accept header
	// content type header
}

// LoadTestConfig contains all the information needed to configure
// and execute a load test run
type LoadTestConfig struct {
	// RqstRate is the desired overall requests per second
	RqstRate int
	// MaxConcurrentRqsts is the overall number of simulataneously
	// running requests
	MaxConcurrentRqsts int
	// RunDuration is how long the test will run. It can be expressed
	// in seconds or minutes as xs or xm where x is an integer (e.g.,
	// 10s for 10 seconds, 5m for 5 minutes). Only one of NumRequests or
	// RunDuration can be specified. The tool will exit with an appropriate
	// error message if this isn't true.
	RunDuration string
	// NumRequests is the total number of requests to make. Specifying
	// both RunDuration and NumRequests is an error. See RunDuration
	// above for a bit more info.
	NumRequests int
	// OutputType specifies if the output will be written in JSON or
	// CSV format. Acceptable values are "JSON" and "CSV". If not
	// specified output will be in JSON format.
	OutputType string
	// Endpoints is the set of endpoints (Endpoint) to make requests to
	Endpoints []Endpoint
}
