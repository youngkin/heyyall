// Copyright (c) 2020 Richard Youngkin. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package internal

import (
	"bytes"
	"context"
	"math"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/youngkin/heyyall/api"
)

// Requestor is the component that will schedule requests to endpoints. It
// expects to be run as a goroutine.
type Requestor struct {
	// Context is used to cancel the goroutine
	Ctx context.Context
	// ResponseC is used to send the results of a request to the response handler
	ResponseC chan Response
	// Client is the target of the test run
	Client http.Client
}

func (r Requestor) processRqst(ep api.Endpoint, numRqsts int, runDur time.Duration, rqstRate int) {
	if len(ep.URL) == 0 || len(ep.Method) == 0 {
		log.Warn().Msgf("Requestor - request contains an invalid endpoint %+v, URL or Method is empty", ep)
		return
	}
	req, err := http.NewRequest(ep.Method, ep.URL, bytes.NewBuffer([]byte(ep.RqstBody)))
	if err != nil {
		log.Warn().Err(err).Msgf("Requestor unable to create http request")
		return
	}

	// At this point we know one of numRqsts or runDur is non-zero. Whichever one
	// is non-zero will be set to a super-high number to effectively disable its
	// test in the for-loop
	if numRqsts == 0 {
		numRqsts = math.MaxInt64
	}
	if runDur == time.Duration(0) {
		runDur = time.Duration(math.MaxInt64)
	}

	for i := 0; i < numRqsts; i++ {
		timesUp := time.After(runDur)
		start := time.Now()
		resp, err := r.Client.Do(req)
		if resp != nil {
			defer resp.Body.Close()
		}
		if err != nil {
			log.Warn().Err(err).Msgf("Requestor: error sending request")
			return
		}
		select {
		case <-timesUp:
			return
		case r.ResponseC <- Response{
			HTTPStatus:      resp.StatusCode,
			Endpoint:        api.Endpoint{URL: ep.URL, Method: ep.Method},
			RequestDuration: time.Since(start),
		}:
		}

		// Zero request rate is completely unthrottled
		if rqstRate == 0 {
			continue
		}
		since := time.Since(start)
		delta := (time.Second / time.Duration(rqstRate)) - since
		if delta < 0 {
			continue
		}
		time.Sleep(delta)

	}
}
