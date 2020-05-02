package testdata

import (
	"context"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/youngkin/heyyall/api"
	"github.com/youngkin/heyyall/internal"
)

type TestCase struct {
	name          string
	rqstRate      int
	runDur        string
	numRqsts      int
	eps           []api.Endpoint
	expectedRqsts []internal.Request
	expectedDist  map[string]int // how many requests per URL
	shouldFail    bool
}

var debugLevel = flag.Int("debugLvl", int(zerolog.ErrorLevel), "debug level - 0 thru 5 0 being DEBUG")

func init() {
	// flag.Parse()
	// zerolog.SetGlobalLevel(zerolog.Level(*debugLevel))
	// log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
}

func scheduler(ctx context.Context, t *testing.T, tc TestCase, schedC chan internal.Request, completeC chan bool) {
	rqstDist := make(map[string]int)
	iterations := 0
	for {
		select {
		case <-ctx.Done():
			if tc.runDur == "0s" && tc.numRqsts != iterations {
				t.Errorf("expected %d requests, got %d", tc.numRqsts, iterations)
				completeC <- false
			}
			for url := range tc.expectedDist {
				if tc.expectedDist[url] != rqstDist[url] {
					t.Errorf("expected %d requests for %s, got %d", tc.expectedDist[url], url, rqstDist[url])
				}
			}
			completeC <- true
			return
		case rqst := <-schedC:
			rqstDist[rqst.EP.URL]++
			// idx := iterations % len(tc.eps)
			// if tc.eps[idx] != rqst.EP {
			// 	t.Errorf("expected %+v, got %+v", tc.eps[idx], rqst.EP)
			// }
			iterations++
		}
	}

}

func TestRequestor(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.Level(*debugLevel))
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	goFastRate := 100
	// goSlowRate := 1
	url1 := "http://somewhere.com/1"
	url2 := "http://somewhere.com/2"

	tests := []TestCase{
		{
			name:     "FailPath - specify both runDur and numRqsts",
			rqstRate: goFastRate,
			runDur:   "1s",
			numRqsts: 10,
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
			expectedDist: map[string]int{url1: 8, url2: 2},
			shouldFail:   true,
		},
		{
			name:     "FailPath - invalid runDur",
			rqstRate: goFastRate,
			runDur:   "1",
			numRqsts: 0,
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
			expectedDist: map[string]int{url1: 8, url2: 2},
			shouldFail:   true,
		},
		{
			name:     "HappyPath - Hit test's numRqsts, should get 10 requests in less than 1 second",
			rqstRate: goFastRate,
			runDur:   "0s",
			numRqsts: 10,
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
			expectedDist: map[string]int{url1: 8, url2: 2},
		},
		// Since this test depends on the behavior of time.Sleep(), it's non-deterministic. It mostly
		// passes, but it can fail when time.Sleep() sleeps substantially longer than planned, which
		// it can and usually does.
		// {
		// 	name:     "HappyPath - Hit test timeout, should get 2 requests in at most 2.5 seconds",
		// 	rqstRate: goSlowRate,
		// 	runDur:   "1006ms",
		// 	numRqsts: 0,
		// 	eps: []api.Endpoint{
		// 		{
		// 			URL:         url1,
		// 			Method:      "GET",
		// 			RqstBody:    "",
		// 			RqstPercent: 50,
		// 			NumRequests: 5,
		// 		},
		// 		{
		// 			URL:         url2,
		// 			Method:      "PUT",
		// 			RqstBody:    "",
		// 			RqstPercent: 50,
		// 			NumRequests: 5,
		// 		},
		// 	},
		// 	expectedDist: map[string]int{url1: 1, url2: 1},
		// },
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			schedC := make(chan internal.Request)
			doneC := make(chan struct{})
			compC := make(chan bool)
			ctx, cancel := context.WithCancel(context.Background())

			rqstr, err := internal.NewRequestor(ctx, doneC, schedC,
				make(chan internal.Response), tc.rqstRate, tc.runDur, tc.numRqsts, tc.eps)
			if err == nil && tc.shouldFail == true {
				t.Fatalf("unexpected success creating Requestor: %s", err)
			}
			if err != nil && tc.shouldFail == false {
				t.Fatalf("unexpected failure creating Requestor: %s", err)
			}
			if err != nil && tc.shouldFail {
				cancel()
				return
			}

			go scheduler(ctx, t, tc, schedC, compC)

			rqstr.Start()

			<-doneC            //rqstr done
			cancel()           // stop scheduler
			success := <-compC // scheduler response

			if !success {
				t.Error("expected scheduler to complete successfully, it didn't")
			}
		})
	}
}

// for _, tc := range tests {
// 	t.Run(tc.name, func(t *testing.T) {
// 		srvHandler, err := testSrvr{HTTPStatus: tc.expectedStatus}
// 		if err != nil {
// 			t.Fatalf("error '%s' was not expected when getting a customer handler", err)
// 		}

// 		testSrv := httptest.NewServer(http.HandlerFunc(srvHandler.ServeHTTP))
// 		defer testSrv.Close()

// 		// NOTE: As there is no http.PUT creating an update request/PUT requires
// 		//	1.	Creating an http.Client (done at the top of this function)
// 		//	2.	Creating the request
// 		//	3. 	Calling client.DO
// 		//
// 		// Kind of round-about, but it works
// 		url := testSrv.URL + tc.url
// 		req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer([]byte(tc.postData)))
// 		if err != nil {
// 			t.Fatalf("an error '%s' was not expected creating HTTP request", err)
// 		}

// 		req.Header.Set("Content-Type", "application/json")
// 		resp, err := client.Do(req)
// 		if err != nil {
// 			t.Fatalf("an error '%s' was not expected calling (client.Do()) accountd server", err)
// 		}

// 		status := resp.StatusCode
// 		if status != tc.expectedHTTPStatus {
// 			t.Errorf("expected StatusCode = %d, got %d", tc.expectedHTTPStatus, status)
// 		}
// 	})
// }
