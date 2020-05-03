package main

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
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
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.StampMilli})

	// TODO: Make this configurable
	runtime.GOMAXPROCS(runtime.NumCPU())

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

	ctx, cancel := context.WithCancel(context.Background())

	responseC := make(chan internal.Response, config.MaxConcurrentRqsts)
	doneC := make(chan struct{})

	responseHandler := internal.ResponseHandler{
		Ctx:       ctx,
		ResponseC: responseC,
		DoneC:     doneC,
	}
	go responseHandler.Start()
	// Give responseHandler a bit of time to start
	time.Sleep(time.Millisecond * 20)

	// TODO: Make Transport configurable, including timeout that's currently on the client below
	t := &http.Transport{
		MaxIdleConnsPerHost: config.MaxConcurrentRqsts,
		DisableCompression:  false,
		DisableKeepAlives:   false,
	}
	client := http.Client{Transport: t, Timeout: time.Second * 15}

	rqstr := internal.Requestor{
		Ctx:       ctx,
		ResponseC: responseC,
		Client:    client,
	}

	scheduler, err := internal.NewScheduler(config.MaxConcurrentRqsts, config.RqstRate, config.RunDuration,
		config.NumRequests, config.Endpoints, rqstr)
	if err != nil {
		log.Fatal().Err(err).Msg("Unexpected error configuring new Requestor")
		cancel()
		return
	}
	go scheduler.Start()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigs:
		cancel()
		close(responseC)
	case <-doneC:
		close(responseC)
	}
}
