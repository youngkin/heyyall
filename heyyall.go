package main

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/youngkin/heyyall/api"
	"github.com/youngkin/heyyall/internal"
)

func main() {
	configFile := flag.String("config", "config.json", "path and filename containing the runtime configuration")
	logLevel := flag.Int("loglevel", int(zerolog.WarnLevel), "log level, 0 for debug, 1 info, 2 warn, ...")
	flag.Parse()

	zerolog.SetGlobalLevel(zerolog.Level(*logLevel))
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	log.Info().Msgf("heyyall started with config from %s", *configFile)

	contents, err := ioutil.ReadFile(*configFile)
	if err != nil {
		log.Fatal().Err(err).Msgf("unable to read config file %s", *configFile)
		return
	}

	log.Debug().Msgf("Raw config file contents: %s", string(contents))

	config := api.LoadTestConfig{}
	if err = json.Unmarshal(contents, &config); err != nil {
		log.Fatal().Err(err).Msgf("error unmarshaling test config bytes: %s", string(contents))
		return
	}

	//fmt.Printf("INFO:\tConfig: %+v\n", config)

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
	}
	go scheduler.Start()
	// Give scheduler a bit of time to start
	time.Sleep(time.Millisecond * 20)

	requestor, err := internal.NewRequestor(ctx, doneC, schedC, responseC,
		config.Rate, config.RunDuration, config.NumRequests, config.Endpoints)
	if err != nil {
		log.Fatal().Err(err).Msg("Unexpected error configuring new Requestor")
		close(schedC)
		close(responseC)
		cancel()
		return
	}
	go requestor.Start()
	// Give requestor a bit of time to start
	time.Sleep(time.Millisecond * 20)

	//fmt.Printf("DEBUG:\tListening for completion until time is up in %s\n", config.RunDuration)
	<-doneC
	close(schedC)
	close(responseC)

	//fmt.Printf("INFO:\theyyall stopping goroutines\n")
	cancel()
	// Give scheduler and responseHandler a bit of time to exit
	time.Sleep(time.Millisecond * 20)

	//fmt.Printf("INFO:\theyyall Exiting\n")
	// Call from SIGTERMHandler
	// close(closeC)
}
