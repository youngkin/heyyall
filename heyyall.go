package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/youngkin/heyyall/api"
	"github.com/youngkin/heyyall/internal"
)

func main() {
	configFile := flag.String("config", "config.json", "path and filename containing the runtime configuration")
	flag.Parse()

	fmt.Printf("INFO:\tReading config file at %s\n", *configFile)
	contents, err := ioutil.ReadFile(*configFile)
	if err != nil {
		fmt.Printf("FATAL:\terror reading config file: %s, error: %s\n\nEXITING!!!\n", *configFile, err)
		return
	}

	fmt.Printf("INFO:\tRaw configFile contents: %v\n", string(contents))

	config := api.LoadTestConfig{}
	if err = json.Unmarshal(contents, &config); err != nil {
		fmt.Printf("FATAL:\terror loading test config from: %s, error: %s.\n\nEXITING!\n", string(contents), err)
		return
	}

	fmt.Printf("INFO:\tConfig: %+v\n", config)

	ctx, cancel := context.WithCancel(context.Background())

	responseC := make(chan internal.Response)
	doneC := make(chan struct{})
	schedC := make(chan internal.Request)

	responseHandler := internal.ResponseHandler{
		Ctx:       ctx,
		ResponseC: responseC,
	}
	go responseHandler.Start()
	// Give responseHandler a bit of time to start
	time.Sleep(time.Millisecond * 20)

	scheduler := internal.Scheduler{
		MaxConcurrentRqsts: config.MaxConcurrentRqsts,
		Ctx:                ctx,
		SchedC:             schedC,
		Rate:               config.Rate,
	}
	go scheduler.Start()
	// Give scheduler a bit of time to start
	time.Sleep(time.Millisecond * 20)

	requestor, err := internal.NewRequestor(ctx, doneC, schedC, responseC,
		config.RunDuration, config.NumRequests, config.Endpoints)
	if err != nil {
		fmt.Printf("FATAL:\terror creating Requestor: %s\n", err)
		return
	}
	go requestor.Start()
	// Give requestor a bit of time to start
	time.Sleep(time.Millisecond * 20)

	fmt.Printf("INFO:\tListening for completion until time is up in %s\n", config.RunDuration)
	<-doneC

	fmt.Printf("INFO:\theyyall stopping goroutines\n")
	cancel()
	// Give scheduler and responseHandler a bit of time to exit
	time.Sleep(time.Millisecond * 20)

	fmt.Printf("INFO:\theyyall Exiting\n")
	// Call from SIGTERMHandler
	// close(closeC)
}
