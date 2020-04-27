package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"

	"github.com/youngkin/heystack/heystack/api"
	"github.com/youngkin/heystack/heystack/internal"
)

func main() {
	configFile := flag.String("config", "config.json", "path and filename containing the runtime configuration")
	flag.Parse()

	contents, err := ioutil.ReadFile(*configFile)
	if err != nil {
		fmt.Printf("Fatal error reading config file: %s, error: %s\n\nEXITING!!!\n", *configFile, err)
		return
	}

	config := api.LoadTestConfig{}
	if err = json.Unmarshal(contents, config); err != nil {
		fmt.Printf("Fatal error loading test config from: %s, error: %s.\n\nEXITING!\n", string(contents), err)
		return
	}

	closeC := make(chan struct{})
	responseC = make(chan internal.Response)
	doneC := make(chan struct{})
	schedC := make(chan internal.Request)

	scheduler := internal.Scheduler{
		MaxConcurrentRqsts: config.MaxConcurrentRqsts,
		CloseC:             closeC,
		DoneC:              doneC,
		Rate:               config.Rate,
		RunDuration:        config.RunDuration,
		NumRequests:        config.NumRequests,
	}

	go scheduler.Start()

	requestor := internal.Requestor {
		CloseC: closeC,
		SchedC: schedC,
		ResponseC: responseC,
		Endpoints: config.Endpoints
	}
}
