// Copyright (c) 2020 Richard Youngkin. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package internal

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/youngkin/heyyall/api"
)

// IRequestor declares the functionality needed to make requests to an endpoint
type IRequestor interface {
	ProcessRqst(ep api.Endpoint, numRqsts int, runDur time.Duration, rqstRate int)
	ResponseChan() chan Response
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
	rqstr IRequestor
}

// NewScheduler returns a valid Scheduler instance
func NewScheduler(concurrency int, rate int, runDur string, numRqsts int,
	eps []api.Endpoint, rqstr IRequestor) (*Scheduler, error) {

	dur, err := time.ParseDuration(runDur)
	if err != nil {
		return nil, fmt.Errorf("runDur: %s, must be of the form 'xs' or xm where 'x' is an integer and 's' indicates seconds and 'm' indicates minutes",
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
		ep := ep
		numRqstsPerGoroutine, epConcurrency, goroutineRqstRate := s.calcEPConfig(ep)
		for i := 0; i < epConcurrency; i++ {
			wg.Add(1)
			go func() {

				log.Debug().Msgf("Starting Endpoint Goroutine for EP: %s numRqsts: %d, runDur: %d, and rqstRate: %d", ep.URL,
					numRqstsPerGoroutine, s.runDur/time.Second, goroutineRqstRate)

				s.rqstr.ProcessRqst(ep, numRqstsPerGoroutine, s.runDur, goroutineRqstRate)
				wg.Done()
			}()
		}
	}

	wg.Wait()
	close(s.rqstr.ResponseChan())

	return nil
}

func (s Scheduler) calcEPConfig(ep api.Endpoint) (numRqstsPerGoroutine int, numEPGoroutines int, epGoroutineRqstRate int) {
	numEPGoroutines = int(math.Ceil(float64(s.concurrency) * (float64(ep.RqstPercent) / float64(100))))
	if numEPGoroutines != int(float64(s.concurrency)*(float64(ep.RqstPercent)/float64(100))) {
		log.Warn().Msgf("EP: %s: epConcurrency, %d, was rounded up. The calcuation result was %f", ep.URL, numEPGoroutines,
			float64(s.concurrency)*(float64(ep.RqstPercent)/float64(100)))
	}

	numEPRqsts := int(math.Ceil(float64(s.numRqsts) * (float64(ep.RqstPercent) / float64(100))))
	if numEPRqsts != int(float64(s.numRqsts)*(float64(ep.RqstPercent)/float64(100))) {
		log.Warn().Msgf("EP: %s: numEPRqsts, %d, was rounded up. The calcuation result was %f", ep.URL, numEPRqsts,
			float64(s.numRqsts)*(float64(ep.RqstPercent)/float64(100)))
	}

	numRqstsPerGoroutine = int(math.Ceil((float64(numEPRqsts) / float64(numEPGoroutines))))
	if numRqstsPerGoroutine != int((float64(numEPRqsts) / float64(numEPGoroutines))) {
		log.Warn().Msgf("EP: %s: numGoRoutineRqsts, %d, was rounded up. The calculation result was %f", ep.URL, numRqstsPerGoroutine,
			(float64(numEPRqsts) / float64(numEPGoroutines)))
	}

	epRqstRate := int(math.Ceil(float64(s.rqstRate) * (float64(ep.RqstPercent) / float64(100))))
	if epRqstRate != int(float64(s.rqstRate)*(float64(ep.RqstPercent)/float64(100))) {
		log.Warn().Msgf("EP: %s: epRqstRate, %d, was rounded up. The calculation result was %f", ep.URL, epRqstRate,
			float64(s.concurrency)*(float64(ep.RqstPercent)/float64(100)))
	}

	epGoroutineRqstRate = int(math.Ceil((float64(epRqstRate) / float64(numEPGoroutines))))
	if epGoroutineRqstRate != int((float64(epRqstRate) / float64(numEPGoroutines))) {
		log.Warn().Msgf("EP: %s: epGoroutineRqstRate, %d, was rounded up. The calculation result was %f", ep.URL,
			epGoroutineRqstRate, (float64(epRqstRate) / float64(numEPGoroutines)))
	}
	return numRqstsPerGoroutine, numEPGoroutines, epGoroutineRqstRate
}

func validateConfig(concurrency int, rate int, runDur time.Duration, numRqsts int, eps []api.Endpoint) error {
	if numRqsts > 0 && runDur > 0 {
		return fmt.Errorf("number of requests is %d and requested duration is %s, one must be zero",
			numRqsts, runDur)
	}
	if runDur < 1 && numRqsts < concurrency {
		return fmt.Errorf("number of requests %d, must be greater than the concurrency level %d", numRqsts, concurrency)
	}
	if runDur < 1 && len(eps) > numRqsts {
		return fmt.Errorf("there are more endpoints, %d, than requests, %d", len(eps), numRqsts)
	}
	if concurrency%len(eps) != 0 {
		return fmt.Errorf("each endpoint must run in it's own goroutine and endpoints must distribute evenly across all goroutines. There are %d goroutines and %d endpoints", concurrency, len(eps))
	}

	rqstPct := 0
	for _, ep := range eps {
		rqstPct += ep.RqstPercent
	}
	if rqstPct != 100 {
		return fmt.Errorf("endpoint.RqstPercents must add up to 100 not %d", rqstPct)
	}
	return nil
}
