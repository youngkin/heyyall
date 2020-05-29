// Copyright (c) 2020 Richard Youngkin. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package internal

import (
	"bytes"
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptrace"
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

// ResponseChan returns a chan Response
func (r Requestor) ResponseChan() chan Response {
	return r.ResponseC
}

// ProcessRqst runs the requests configured by 'ep' at the requested rate for either
// 'numRqsts' times or 'runDur' duration
func (r Requestor) ProcessRqst(ep api.Endpoint, numRqsts int, runDur time.Duration, rqstRate int) {
	if len(ep.URL) == 0 || len(ep.Method) == 0 {
		log.Warn().Msgf("Requestor - request contains an invalid endpoint %+v, URL or Method is empty", ep)
		return
	}

	// TODO: Add context to request
	req, err := http.NewRequest(ep.Method, ep.URL, bytes.NewBuffer([]byte(ep.RqstBody)))
	if err != nil {
		log.Warn().Err(err).Msgf("Requestor unable to create http request")
		return
	}

	var dnsStart, dnsDone, connDone, gotResp, tlsStart, tlsDone time.Time

	trace := &httptrace.ClientTrace{
		DNSStart: func(_ httptrace.DNSStartInfo) { dnsStart = time.Now() },
		DNSDone:  func(_ httptrace.DNSDoneInfo) { dnsDone = time.Now() },
		ConnectStart: func(_, _ string) {
			if dnsDone.IsZero() {
				// connecting directly to IP
				dnsDone = time.Now()
			}
		},
		ConnectDone: func(net, addr string, err error) {
			if err != nil {
				log.Fatal().Msgf("unable to connect to host %v: %v", addr, err)
			}
			connDone = time.Now()

		},
		GotConn:              func(_ httptrace.GotConnInfo) { connDone = time.Now() },
		GotFirstResponseByte: func() { gotResp = time.Now() },
		TLSHandshakeStart:    func() { tlsStart = time.Now() },
		TLSHandshakeDone:     func(_ tls.ConnectionState, _ error) { tlsDone = time.Now() },
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	// At this point we know one of numRqsts or runDur is non-zero. Whichever one
	// is non-zero will be set to a super-high number to effectively disable its
	// test in the for-loop below
	if numRqsts == 0 {
		log.Debug().Msgf("ProcessRqst: EP: %s, numRqsts %d was 0", ep.URL, numRqsts)
		// TODO: Need to come back here and ensure that each EP goroutine can only
		// run it's share of the api.MaxRqsts limit.
		numRqsts = api.MaxRqsts
	}
	if runDur == time.Duration(0) {
		log.Debug().Msgf("ProcessRqst: EP: %s, runDur %d was 0", ep.URL, runDur/time.Second)
		runDur = api.MaxRunDuration
	}

	client := r.Client
	if ep.CertFile != "" {
		if ep.KeyFile == "" {
			log.Fatal().Msgf("Endpoint: %s, Endpoint.CertFile specified: %s, Endpoint.KeyFile is not", ep.URL, ep.CertFile)
		}
		log.Debug().Msgf("Endpoint %s is overriding SSL certificate using certificate file %s", ep.URL, ep.CertFile)
		cert, err := tls.LoadX509KeyPair(ep.CertFile, ep.KeyFile)
		if err != nil {
			log.Fatal().Err(err).Msg("Error creating x509 keypair")
		}
		t1, ok := r.Client.Transport.(*http.Transport)
		if !ok {
			log.Fatal().Msg("Requestor.ProcessRqst(): Could not cast Client.Transport to *http.Transport")
		}
		t2 := &http.Transport{
			MaxIdleConnsPerHost: t1.MaxConnsPerHost,
			DisableCompression:  t1.DisableCompression,
			DisableKeepAlives:   t1.DisableKeepAlives,
			TLSClientConfig: &tls.Config{
				Certificates: []tls.Certificate{cert},
			},
		}
		client.Transport = t2
	}

	log.Debug().Msgf("Setting 'timesUp' duration to %d seconds", runDur/time.Second)
	timesUp := time.After(runDur)
	for i := 0; i < numRqsts; i++ {
		start := time.Now()
		resp, err := client.Do(req)
		if resp != nil {
			defer resp.Body.Close()
		}
		if err != nil {
			log.Warn().Err(err).Msgf("Requestor: error sending request")
			return // TODO: Should return here? This assumes that the error is persistent
		}
		select {
		case <-r.Ctx.Done():
			log.Debug().Msg("Requestor cancelled, exiting")
			return
		case <-timesUp:
			log.Debug().Msg("Requestor runDur expired, exiting")
			return
		case r.ResponseC <- Response{
			HTTPStatus:           resp.StatusCode,
			Endpoint:             api.Endpoint{URL: ep.URL, Method: ep.Method},
			RequestDuration:      time.Since(start),
			DNSLookupDuration:    dnsDone.Sub(dnsStart),
			TCPConnDuration:      connDone.Sub(dnsDone),
			RoundTripDuration:    gotResp.Sub(connDone),
			TLSHandshakeDuration: tlsDone.Sub(tlsStart),
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
