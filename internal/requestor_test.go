// Copyright (c) 2020 Richard Youngkin. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package internal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/youngkin/heyyall/api"
)

type srvHandler struct {
	HTTPStatus int
}

func (s *srvHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for headerName, headerValues := range r.Header {
		for _, headerValue := range headerValues {
			w.Header().Add(headerName, headerValue)
		}
	}
	w.WriteHeader(http.StatusOK)
}

// TestHappyPath verifies that the scheduler successfully makes a single
// requestor request, via 'ProcessRqst()' and completes normally. To accomplish
// this the goroutine running 'ProcessRqst()' must get a successful response
// from a test HTTP server, send that response over a ResponseC channel, and
// exit.
func TestHappyPath(t *testing.T) {
	ep := api.Endpoint{
		Method:      "GET",
		RqstPercent: 100,
		Headers: map[string]string{
			"Content-Type":    "application/happy",
			"X-Custom-Header": "customer header value",
		},
	}

	srvHandler := srvHandler{HTTPStatus: 200}

	testSrv := httptest.NewServer(http.HandlerFunc(srvHandler.ServeHTTP))
	defer testSrv.Close()

	url := testSrv.URL + "/testme"
	ep.URL = url

	client := http.Client{}
	ctx := context.Context(context.Background())
	respC := make(chan Response)
	rqstr := Requestor{
		Ctx:       ctx,
		ResponseC: respC,
		Client:    client,
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		rqstr.ProcessRqst(ep, 1, 1000)
		wg.Done()
	}()
	resp := <-respC
	if resp.HTTPStatus != http.StatusOK {
		t.Errorf("expected HTTP status %d, got %d", 200, resp.HTTPStatus)
	}
	if contentType := resp.Header.Get("Content-Type"); contentType != "application/happy" {
		t.Errorf("unexpected HTTP Response header Content-Type, got %s", contentType)
	}
	if customHeader := resp.Header.Get("X-Custom-Header"); customHeader != "customer header value" {
		t.Errorf("unexpected HTTP Response header X-Custom-Header, got %s", customHeader)
	}

	wg.Wait()
}

func TestCtxCancel(t *testing.T) {
	ep := api.Endpoint{
		Method:      "GET",
		RqstPercent: 100,
	}

	srvHandler := srvHandler{HTTPStatus: 200}

	testSrv := httptest.NewServer(http.HandlerFunc(srvHandler.ServeHTTP))
	defer testSrv.Close()

	url := testSrv.URL + "/testme"
	ep.URL = url

	client := http.Client{}
	ctx, cancel := context.WithCancel(context.Background())

	respC := make(chan Response)
	rqstr := Requestor{
		Ctx:       ctx,
		ResponseC: respC,
		Client:    client,
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		rqstr.ProcessRqst(ep, 1, 1000)
		wg.Done()
	}()

	time.Sleep(10 * time.Millisecond)
	cancel()

	wg.Wait()
}

func TestTimeout(t *testing.T) {
	ep := api.Endpoint{
		Method:      "GET",
		RqstPercent: 100,
	}

	srvHandler := srvHandler{HTTPStatus: 200}

	testSrv := httptest.NewServer(http.HandlerFunc(srvHandler.ServeHTTP))
	defer testSrv.Close()

	url := testSrv.URL + "/testme"
	ep.URL = url

	client := http.Client{}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	respC := make(chan Response)
	rqstr := Requestor{
		Ctx:       ctx,
		ResponseC: respC,
		Client:    client,
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		rqstr.ProcessRqst(ep, 0, 1000)
		wg.Done()
	}()

	wg.Wait()
}
