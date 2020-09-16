// Copyright (c) 2020 Richard Youngkin. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package internal

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptrace"
	"net/url"
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
// 'numRqsts' times or the configured run duration (set in Requestor.Ctx)
func (r Requestor) ProcessRqst(ep api.Endpoint, numRqsts int, rqstRate int) {
	if len(ep.URL) == 0 || len(ep.Method) == 0 {
		log.Warn().Msgf("Requestor - request contains an invalid endpoint %+v, URL or Method is empty", ep)
		return
	}

	req, err := http.NewRequestWithContext(r.Ctx, ep.Method, ep.URL, bytes.NewBuffer([]byte(ep.RqstBody)))
	if err != nil {
		log.Warn().Err(err).Msgf("Requestor unable to create http request")
		return
	}

	var dnsStart, dnsDone, connStart, connDone, gotResp, tlsStart, tlsDone time.Time

	trace := &httptrace.ClientTrace{
		DNSStart:             func(_ httptrace.DNSStartInfo) { dnsStart = time.Now() },
		DNSDone:              func(_ httptrace.DNSDoneInfo) { dnsDone = time.Now() },
		GetConn:              func(_ string) { connStart = time.Now() },
		GotConn:              func(_ httptrace.GotConnInfo) { connDone = time.Now() },
		GotFirstResponseByte: func() { gotResp = time.Now() },
		TLSHandshakeStart:    func() { tlsStart = time.Now() },
		TLSHandshakeDone:     func(_ tls.ConnectionState, _ error) { tlsDone = time.Now() },
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	if numRqsts == 0 {
		log.Debug().Msgf("ProcessRqst: EP: %s, numRqsts was 0, setting to %d", ep.URL, api.MaxRqsts)
		numRqsts = api.MaxRqsts
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

	for i := 0; i < numRqsts; i++ {
		start := time.Now()
		resp, err := client.Do(req)
		if err != nil {
			switch e := err.(type) {
			case *url.Error:
				if e.Timeout() {
					return
				}
			default:
				log.Warn().Err(err).Msgf("Requestor: error %s sending request, dropping %d remaining requests", err, numRqsts-(i+1))
				return
			}
		}

		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()

		select {
		case <-r.Ctx.Done():
			log.Debug().Msg("Requestor cancelled or the run duration expired, exiting")
			return
		case r.ResponseC <- Response{
			HTTPStatus:           resp.StatusCode,
			Endpoint:             api.Endpoint{URL: ep.URL, Method: ep.Method},
			RequestDuration:      time.Since(start),
			DNSLookupDuration:    dnsDone.Sub(dnsStart),
			TCPConnDuration:      connDone.Sub(connStart),
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
