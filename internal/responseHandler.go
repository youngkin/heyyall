package internal

import (
	"context"
	"fmt"
	"time"
)

// ResponseHandler is responsible for accepting, summarizing, and reporting
// on the overall load test results.
type ResponseHandler struct {
	Ctx       context.Context
	ResponseC chan Response
}

// Start begins the process of accepting responses. It expects to be run as a goroutine
func (rh ResponseHandler) Start() {
	fmt.Printf("INFO:\tResponseHandler starting: %+v\n", rh)
	start := time.Now()

	// TODO IMPLEMENT
	numResponses := 0
	for {
		select {
		case <-rh.Ctx.Done():
			runDur := (time.Since(start)) / time.Second
			fmt.Printf("INFO:\tResponseHandler received close signal; handled %d responses in %d seconds\n", numResponses, runDur)
			return
		case resp := <-rh.ResponseC:
			fmt.Printf("INFO:\tReceived response: %+v\n", resp)
			numResponses++
		}
	}
}
