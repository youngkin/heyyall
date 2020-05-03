package internal

import (
	"bytes"
	"context"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/youngkin/heyyall/api"
)

// Scheduler is the component that will schedule requests to endpoints. It
// expects to be run as a goroutine.
type Scheduler struct {
	// Context is used to cancel the goroutine
	Ctx context.Context
	// SchedC is used to send requests to the Scheduler
	SchedC chan Request
	// ResponseC is used to send the results of a request to the response handler
	ResponseC chan Response
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
	log.Debug().Msg("Scheduler starting")
	s.concurrencySem = make(chan struct{}, s.MaxConcurrentRqsts)
	t := &http.Transport{
		MaxIdleConnsPerHost: s.MaxConcurrentRqsts,
		DisableCompression:  false,
		DisableKeepAlives:   false,
	}
	s.client = http.Client{Transport: t, Timeout: time.Second * 15}
	for {
		select {
		case <-s.Ctx.Done():
			log.Debug().Msg("Scheduler received close signal, exiting")
			return
		case rqst := <-s.SchedC:
			// Rqst ability to send off a request
			s.concurrencySem <- struct{}{}
			go s.processRqst(rqst)
		}
	}
}

func (s Scheduler) processRqst(rqst Request) {
	//fmt.Printf("INFO:\tReceived request: %+v. STOP PRINTING THIS MESSAGE!!!!\n", rqst)
	if len(rqst.EP.URL) == 0 {
		log.Warn().Msgf("Scheduler - request contains an invalid endpoint %+v, URL is empty", rqst.EP)
		return
	}
	req, err := http.NewRequest(rqst.EP.Method, rqst.EP.URL, bytes.NewBuffer([]byte(rqst.EP.RqstBody)))
	if err != nil {
		log.Warn().Err(err).Msgf("Scheduler unable to create http request")
		return
	}

	start := time.Now()
	resp, err := s.client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		log.Warn().Err(err).Msgf("Scheduler: error sending request")
		<-s.concurrencySem
		return
	}

	s.ResponseC <- Response{
		HTTPStatus:      resp.StatusCode,
		Endpoint:        api.Endpoint{URL: rqst.EP.URL, Method: rqst.EP.Method},
		RequestDuration: time.Since(start),
	}
	// signal a request has completed processing
	<-s.concurrencySem
}
