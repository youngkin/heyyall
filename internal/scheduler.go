package internal

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/youngkin/heyyall/api"
)

// Scheduler is the component that will schedule requests to endpoints. It
// expects to be run as a goroutine.
type Scheduler struct {
	// Context is used to cancel the goroutine
	Ctx context.Context
	// SchedC is used to send requests to the Scheduler
	SchedC chan Request
	// Rate is the desired overall requests per second
	Rate int
	// MaxConcurrentRqsts is the overall number of simulataneously
	// running requests
	MaxConcurrentRqsts int
	client             http.Client
	// concurrencySem is a semaphore (implemented via channels) to control max
	// concurrent requests

	concurrencySem chan struct{}
}

// Start begins a loop that schedules each request for forwarding
// to the endpoint in the request
func (s Scheduler) Start() {
	fmt.Printf("INFO:\tScheduler starting: %+v\n", s)
	s.concurrencySem = make(chan struct{}, s.MaxConcurrentRqsts)
	s.client = http.Client{Timeout: time.Second * 5}
	for {
		select {
		case <-s.Ctx.Done():
			fmt.Println("INFO:\tScheduler received close signal, exiting")
			return
		case rqst := <-s.SchedC:
			// Rqst ability to send off a request
			s.concurrencySem <- struct{}{}
			go s.processRqst(rqst)
		}
	}
}

func (s Scheduler) processRqst(rqst Request) {
	fmt.Printf("INFO:\tReceived request: %+v. STOP PRINTING THIS MESSAGE!!!!\n", rqst)
	req, err := http.NewRequest(rqst.EP.Method, rqst.EP.URL, bytes.NewBuffer([]byte(rqst.EP.RqstBody)))
	if err != nil {
		fmt.Printf("ERROR:\tScheduler unable to create http request, error: %+v\n", err)
		return
	}

	start := time.Now()
	resp, err := s.client.Do(req)
	if err != nil {
		fmt.Printf("ERROR:\tScheduler: error sending request: %+v\n", err)
		return
	}
	rqst.ResponseC <- Response{
		HTTPStatus:      resp.StatusCode,
		Endpoint:        api.Endpoint{URL: rqst.EP.URL},
		RequestDuration: time.Since(start),
	}
	// signal a request has completed processing
	<-s.concurrencySem
}
