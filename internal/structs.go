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
	EP api.Endpoint
}
