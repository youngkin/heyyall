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
	w.WriteHeader(http.StatusOK)
}

func TestNumRqsts(t *testing.T) {
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
		rqstr.ProcessRqst(ep, 1, time.Second*0, 1000)
		wg.Done()
	}()
	resp := <-respC
	if resp.HTTPStatus != http.StatusOK {
		t.Errorf("expected HTTP status %d, got %d", 200, resp.HTTPStatus)
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
		rqstr.ProcessRqst(ep, 1, time.Second*0, 1000)
		wg.Done()
	}()

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
	ctx, cancel := context.WithCancel(context.Background())
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
		rqstr.ProcessRqst(ep, 0, time.Millisecond*10, 1000)
		wg.Done()
	}()

	wg.Wait()
}
