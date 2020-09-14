// Copyright (c) 2020 Richard Youngkin. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package internal

import (
	"flag"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/youngkin/heyyall/api"
)

var debugLevel = flag.Int("debugLvl", int(zerolog.ErrorLevel), "debug level - 0 thru 5 0 being DEBUG")

type MockRequestor struct {
	responseC         chan Response
	expectedNumRqstrs int
	actualNumRqstrs   int
	mux               *sync.Mutex
}

func (r *MockRequestor) ProcessRqst(ep api.Endpoint, numRqsts int, rqstRate int) {
	r.mux.Lock()
	r.actualNumRqstrs += numRqsts
	r.mux.Unlock()
}

func (r *MockRequestor) ResponseChan() chan Response {
	return r.responseC
}

type expectedEPCalcs struct {
	xnumRqstsPerGoroutine int
	xepConcurrecy         int
	xgoroutineRqstRate    int
}

func TestCalcEPConfig(t *testing.T) {
	tests := []struct {
		name             string
		eps              []api.Endpoint
		schedConcurrency int
		schedRqstRate    int
		schedNumRqsts    int
		xEPCalcs         []expectedEPCalcs
	}{
		{
			name: "1EPNoConcurrencyNoRounding",
			eps: []api.Endpoint{
				{
					URL:         "http://somewhere.com",
					Method:      "GET",
					RqstPercent: 100,
				},
			},
			schedConcurrency: 1,
			schedRqstRate:    10,
			schedNumRqsts:    100,
			xEPCalcs: []expectedEPCalcs{
				{
					xnumRqstsPerGoroutine: 100,
					xepConcurrecy:         1,
					xgoroutineRqstRate:    10,
				},
			},
		},
		{
			name: "1EP2ConcurrencyNoRounding",
			eps: []api.Endpoint{
				{
					URL:         "http://somewhere.com",
					Method:      "GET",
					RqstPercent: 100,
				},
			},
			schedConcurrency: 2,
			schedRqstRate:    10,
			schedNumRqsts:    100,
			xEPCalcs: []expectedEPCalcs{
				{
					xnumRqstsPerGoroutine: 50,
					xepConcurrecy:         2,
					xgoroutineRqstRate:    5,
				},
			},
		},
		{
			name: "1EP3ConcurrencyExpectRounding",
			eps: []api.Endpoint{
				{
					URL:         "http://somewhere.com",
					Method:      "GET",
					RqstPercent: 100,
				},
			},
			schedConcurrency: 3,
			schedRqstRate:    10,
			schedNumRqsts:    100,
			xEPCalcs: []expectedEPCalcs{
				{
					xnumRqstsPerGoroutine: 34,
					xepConcurrecy:         3,
					xgoroutineRqstRate:    4,
				},
			},
		},
		{
			name: "2EP4ConcurrencyNoRounding",
			eps: []api.Endpoint{
				{
					URL:         "http://somewhere.com",
					Method:      "GET",
					RqstPercent: 75,
				},
				{
					URL:         "http://somewhere2.com",
					Method:      "GET",
					RqstPercent: 25,
				},
			},
			schedConcurrency: 4,
			schedRqstRate:    100,
			schedNumRqsts:    100,
			xEPCalcs: []expectedEPCalcs{
				{
					xnumRqstsPerGoroutine: 25,
					xepConcurrecy:         3,
					xgoroutineRqstRate:    25,
				},
				{
					xnumRqstsPerGoroutine: 25,
					xepConcurrecy:         1,
					xgoroutineRqstRate:    25,
				},
			},
		},
		{
			name: "2EP4ConcurrencyExpectRounding",
			eps: []api.Endpoint{
				{
					URL:         "http://somewhere.com",
					Method:      "GET",
					RqstPercent: 80,
				},
				{
					URL:         "http://somewhere2.com",
					Method:      "GET",
					RqstPercent: 20,
				},
			},
			schedConcurrency: 4,
			schedRqstRate:    100,
			schedNumRqsts:    100,
			xEPCalcs: []expectedEPCalcs{
				{
					xnumRqstsPerGoroutine: 20,
					xepConcurrecy:         4,
					xgoroutineRqstRate:    20,
				},
				{
					xnumRqstsPerGoroutine: 20,
					xepConcurrecy:         1,
					xgoroutineRqstRate:    20,
				},
			},
		},
		{
			name: "3EP4ConcurrencyExpectAllCalcsRounded",
			eps: []api.Endpoint{
				{
					URL:         "http://somewhere.com",
					Method:      "GET",
					RqstPercent: 50,
				},
				{
					URL:         "http://somewhere2.com",
					Method:      "GET",
					RqstPercent: 30,
				},
				{
					URL:         "http://somewhere3.com",
					Method:      "GET",
					RqstPercent: 20,
				},
			},
			schedConcurrency: 4,
			schedRqstRate:    99,
			schedNumRqsts:    99,
			xEPCalcs: []expectedEPCalcs{
				{
					xnumRqstsPerGoroutine: 25,
					xepConcurrecy:         2,
					xgoroutineRqstRate:    25,
				},
				{
					xnumRqstsPerGoroutine: 15,
					xepConcurrecy:         2,
					xgoroutineRqstRate:    15,
				},
				{
					xnumRqstsPerGoroutine: 20,
					xepConcurrecy:         1,
					xgoroutineRqstRate:    20,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := Scheduler{
				concurrency: tc.schedConcurrency,
				rqstRate:    tc.schedRqstRate,
				numRqsts:    tc.schedNumRqsts,
			}

			for i, ep := range tc.eps {
				numRqstsPerGoroutine, epConcurrency, goroutineRqstRate := s.calcEPConfig(ep)
				if numRqstsPerGoroutine != tc.xEPCalcs[i].xnumRqstsPerGoroutine ||
					epConcurrency != tc.xEPCalcs[i].xepConcurrecy ||
					goroutineRqstRate != tc.xEPCalcs[i].xgoroutineRqstRate {
					t.Errorf("expected %d, %d, and %d, got %d, %d, and %d",
						tc.xEPCalcs[i].xnumRqstsPerGoroutine, tc.xEPCalcs[i].xepConcurrecy, tc.xEPCalcs[i].xgoroutineRqstRate,
						numRqstsPerGoroutine, epConcurrency, goroutineRqstRate)
				}
			}
		})
	}
}

func TestValidation(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.Level(*debugLevel))
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	goFastRate := 100
	// goSlowRate := 1
	url1 := "http://somewhere.com/1"
	url2 := "http://somewhere.com/2"

	tests := []struct {
		name        string
		rqstRate    int
		runDur      string
		numRqsts    int
		concurrency int
		eps         []api.Endpoint
		rqstr       IRequestor
		shouldFail  bool
	}{
		{
			name:        "FailPath - specify both runDur and numRqsts",
			rqstRate:    goFastRate,
			runDur:      "1s",
			numRqsts:    10,
			concurrency: 100,
			eps: []api.Endpoint{
				{
					URL:         url1,
					Method:      "GET",
					RqstBody:    "",
					RqstPercent: 80,
				},
				{
					URL:         url2,
					Method:      "PUT",
					RqstBody:    "",
					RqstPercent: 20,
				},
			},
			shouldFail: true,
		},
		{
			name:        "FailPath - numRqsts < concurrency",
			rqstRate:    goFastRate,
			runDur:      "0s",
			numRqsts:    20,
			concurrency: 100,
			eps: []api.Endpoint{
				{
					URL:         url1,
					Method:      "GET",
					RqstBody:    "",
					RqstPercent: 80,
					NumRequests: 5,
				},
				{
					URL:         url2,
					Method:      "PUT",
					RqstBody:    "",
					RqstPercent: 20,
					NumRequests: 5,
				},
			},
			shouldFail: true,
		},
		{
			name:        "SuccessPath - numRqsts < concurrency but RunDur > 0",
			rqstRate:    goFastRate,
			runDur:      "1s",
			numRqsts:    0,
			concurrency: 100,
			eps: []api.Endpoint{
				{
					URL:         url1,
					Method:      "GET",
					RqstBody:    "",
					RqstPercent: 80,
					NumRequests: 5,
				},
				{
					URL:         url2,
					Method:      "PUT",
					RqstBody:    "",
					RqstPercent: 20,
					NumRequests: 5,
				},
			},
			shouldFail: false,
		},
		{
			name:        "FailPath - len(eps) > numRqsts",
			rqstRate:    goFastRate,
			runDur:      "0s",
			numRqsts:    2,
			concurrency: 3,
			eps: []api.Endpoint{
				{
					URL:         url1,
					Method:      "GET",
					RqstBody:    "",
					RqstPercent: 80,
					NumRequests: 5,
				},
				{
					URL:         url2,
					Method:      "PUT",
					RqstBody:    "",
					RqstPercent: 20,
					NumRequests: 5,
				},
				{
					URL:         url2,
					Method:      "PUT",
					RqstBody:    "",
					RqstPercent: 20,
					NumRequests: 5,
				},
			},
			shouldFail: true,
		},
		{
			name:        "SuccessPath - len(eps) > numRqsts but RunDur > 0",
			rqstRate:    goFastRate,
			runDur:      "1s",
			numRqsts:    0,
			concurrency: 3,
			eps: []api.Endpoint{
				{
					URL:         url1,
					Method:      "GET",
					RqstBody:    "",
					RqstPercent: 80,
					NumRequests: 5,
				},
				{
					URL:         url2,
					Method:      "PUT",
					RqstBody:    "",
					RqstPercent: 10,
					NumRequests: 5,
				},
				{
					URL:         url2,
					Method:      "PUT",
					RqstBody:    "",
					RqstPercent: 10,
					NumRequests: 5,
				},
			},
			shouldFail: false,
		},
		{
			name:        "FailPath - concurrency must be >= len(eps) - otherwise some concurrency is lost",
			rqstRate:    goFastRate,
			runDur:      "0s",
			numRqsts:    99,
			concurrency: 1,
			eps: []api.Endpoint{
				{
					URL:         url1,
					Method:      "GET",
					RqstBody:    "",
					RqstPercent: 80,
					NumRequests: 5,
				},
				{
					URL:         url2,
					Method:      "PUT",
					RqstBody:    "",
					RqstPercent: 20,
					NumRequests: 5,
				},
			},
			shouldFail: true,
		},
		{
			name:        "FailPath - EP Rqst Percent greater than 100",
			rqstRate:    goFastRate,
			runDur:      "0s",
			numRqsts:    100,
			concurrency: 100,
			eps: []api.Endpoint{
				{
					URL:         url1,
					Method:      "GET",
					RqstBody:    "",
					RqstPercent: 80,
					NumRequests: 5,
				},
				{
					URL:         url2,
					Method:      "PUT",
					RqstBody:    "",
					RqstPercent: 30,
					NumRequests: 5,
				},
			},
			shouldFail: true,
		},
		{
			name:        "FailPath - EP Rqst Percent less than 100",
			rqstRate:    goFastRate,
			runDur:      "0s",
			numRqsts:    100,
			concurrency: 100,
			eps: []api.Endpoint{
				{
					URL:         url1,
					Method:      "GET",
					RqstBody:    "",
					RqstPercent: 10,
					NumRequests: 5,
				},
				{
					URL:         url2,
					Method:      "PUT",
					RqstBody:    "",
					RqstPercent: 30,
					NumRequests: 5,
				},
			},
			shouldFail: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runDir, err := time.ParseDuration(tc.runDur)
			if err != nil {
				t.Fatalf("unable to parse time.Duration from %s", tc.runDur)
			}

			_, err = NewScheduler(tc.concurrency, tc.rqstRate, runDir,
				tc.numRqsts, tc.eps, tc.rqstr)

			if err == nil && tc.shouldFail == true {
				t.Fatalf("unexpected success creating Scheduler")
			}
			if err != nil && tc.shouldFail == false {
				t.Fatalf("unexpected failure creating Scheduler: %s", err)
			}
			if err != nil && tc.shouldFail {
				return
			}
		})
	}
}

// TestRqstrInteractions validates that the Scheduler will schedule 2 rqstr goroutines to process
// the request and that each wil process 3 requests due to rounding. 'numRqsts'/'concurrency' = 2.5
// which will be rounded up to 3 each.
func TestRqstrInteractions(t *testing.T) {
	concurrency := 2
	numRqsts := 5
	expectedRqstsFromRqstr := 6
	responseC := make(chan Response)
	rqstr := &MockRequestor{responseC: responseC, expectedNumRqstrs: expectedRqstsFromRqstr, mux: &sync.Mutex{}}
	eps := []api.Endpoint{
		{
			URL:         "doesn'tMatter",
			RqstPercent: 100,
		},
	}

	s, err := NewScheduler(concurrency, 1000, time.Duration(0), numRqsts, eps, rqstr)
	if err != nil {
		t.Errorf("unexpected error calling NewScheduler(): %s", err)
	}

	go s.Start()

	timesUp := time.After(time.Millisecond * 100)
	select {
	case <-timesUp:
		t.Error("Time expired before test completed")
	case <-responseC:
	}

	if rqstr.actualNumRqstrs != rqstr.expectedNumRqstrs {
		t.Errorf("expected %d requests, got %d", rqstr.expectedNumRqstrs, rqstr.actualNumRqstrs)
	}
}
