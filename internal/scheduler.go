// Copyright (c) 2020 Richard Youngkin. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package internal

import (
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/youngkin/heyyall/api"
)

// rqstItem contains the information needed to determine how often to send
// requests to a given endpoint. It also includes the nextRunOffset iteration to
// send the request. This 'nextRunOffset' interation will be recalculated after each
// request is sent.
type rqstItem struct {
	freq int // 1 is every request, 2 is every other request, 10 is every 10th request...
	ep   api.Endpoint
}

// Scheduler determines which requests to make over the schedC
// channel based on each Endpoint's 'RqstPercent'
type Scheduler struct {
	// concurrency is the overall number of simulataneously
	// running requests
	concurrency int
	// rqstRate is the desired overall requests per second
	rqstRate int
	// runDur is how long the test will run. It can be expressed
	// in seconds or minutes as xs or xm where x is an integer (e.g.,
	// 10s for 10 seconds, 5m for 5 minutes). If both numRqsts and
	// runDur are specified then whichever one is met first will
	// cause the run to cease. For example, if runDur is 10s and
	// numRqsts is 200,000,000, if runDur expires before 200,000,000
	// requests are made then the load test will finish at 10 seconds
	// regardless of the number of requests specified.
	runDur time.Duration
	// numRqsts is the total number of requests to make. See runDur
	// above for the behavior when both runDur and numRqsts are
	// specified.
	numRqsts int
	// endpoints represents the set of endpoints getting requests
	endpoints []api.Endpoint
	// rqstr is responsible for making client requests to endpoints
	rqstr Requestor
}

// NewScheduler returns a valid Scheduler instance
func NewScheduler(concurrency int, rate int, runDur string, numRqsts int,
	eps []api.Endpoint, rqstr Requestor) (*Scheduler, error) {

	dur, err := time.ParseDuration(runDur)
	if err != nil {
		return nil, fmt.Errorf("Scheduler - runDur: %s, must be of the form xs or xm where x is an integer",
			runDur)
	}
	err = validateConfig(concurrency, rate, dur, numRqsts, eps)
	if err != nil {
		return nil, err
	}

	schedlr := Scheduler{
		concurrency: concurrency,
		rqstRate:    rate,
		runDur:      dur,
		numRqsts:    numRqsts,
		endpoints:   eps,
		rqstr:       rqstr,
	}
	log.Debug().Msgf("Scheduler: %+v", schedlr)

	return &schedlr, nil
}

// Start begins the scheduling process
func (s Scheduler) Start() error {
	var wg sync.WaitGroup

	for _, ep := range s.endpoints {
		numEPRqsts := 0
		if s.numRqsts > 0 {
			numEPRqsts = int(float64(s.numRqsts) * (float64(ep.RqstPercent) / float64(100)))
			if numEPRqsts == 0 {
				numEPRqsts = 1
				log.Warn().Msgf("endpoint %s's request percentage is less than 1, it will be rounded to 1", ep.URL)
			}
		}
		epConcurrency := s.concurrency * (ep.RqstPercent / 100)
		if epConcurrency == 0 {
			return fmt.Errorf("endpoint %s's request percentage %d for the requested concurrency level %d must be at least 1. Concurrency is calcuated as an endpoint's request percentage times the overall requested concurrency level. In this case it is %f * %d = %f",
				ep.URL, ep.RqstPercent, s.concurrency, float64(ep.RqstPercent)/float64(100),
				s.concurrency, float64(ep.RqstPercent)/float64(100)*float64(s.concurrency))
		}
		if float64(epConcurrency) != float64(s.concurrency)*(float64(ep.RqstPercent)/float64(100)) {
			log.Warn().Msgf("endpoint %s's concurrency is not an integer, rounding down so total concurrency will be less than requested. Concurrency is calcuated as an endpoint's request percentage times the overall requested concurrency level. In this case it is %f * %d = %f which was rounded down to %d",
				ep.URL, float64(ep.RqstPercent)/float64(100), s.concurrency, float64(ep.RqstPercent)/float64(100)*float64(s.concurrency), epConcurrency)
		}
		for i := 0; i < epConcurrency; i++ {
			wg.Add(1)
			go func() {
				s.rqstr.processRqst(ep, numEPRqsts/epConcurrency, s.runDur, s.rqstRate/epConcurrency)
				wg.Done()
			}()
		}
	}

	wg.Wait()
	close(s.rqstr.ResponseC)

	return nil
}

func validateConfig(concurrency int, rate int, runDur time.Duration, numRqsts int, eps []api.Endpoint) error {
	if numRqsts > 0 && runDur > 0 {
		return fmt.Errorf("number of requests is %d and requested duration is %s, one must be zero",
			numRqsts, runDur)
	}
	if numRqsts < concurrency {
		return fmt.Errorf("number of requests %d, must be greater than the concurrency level %d", numRqsts, concurrency)
	}
	if len(eps) > numRqsts {
		return fmt.Errorf("there are more endpoints, %d, than requests, %d", len(eps), numRqsts)
	}
	if concurrency%len(eps) != 0 {
		return fmt.Errorf("each endpoint must run in it's own thread and endpoints must distribute evenly across all threads. There are %d threads and %d endpoints", concurrency, len(eps))
	}
	return nil
}
