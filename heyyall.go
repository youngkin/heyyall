// Copyright (c) 2020 Richard Youngkin. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/youngkin/heyyall/api"
	"github.com/youngkin/heyyall/internal"

	"github.com/vbauerster/mpb/v5"
	"github.com/vbauerster/mpb/v5/decor"
)

func main() {
	usage := `
Usage: heyyall -config <ConfigFileLocation> [flags...]

Options:
  -loglevel  Logging level. Default is 'WARN' (2). 0 is DEBUG, 1 INFO, up to 4 FATAL
  -out       Type of output report, 'text' or 'json'. Default is 'text'
  -nf        Normalization factor used to compress the output histogram by eliminating long tails. 
             Lower values provide a finer grained view of the data at the expense of dropping data
             associated with the tail of the latency distribution. The latter is partly mitigated by 
             including a final histogram bin containing the number of observations between it and
             the previous latency bin. While this doesn't show a detailed distribution of the tail,
             it does indicate how many observations are included in the tail. 10 is generally a good 
             starting number but may vary depending on the actual latency distribution and range
             of latency values. The default is 0 which signifies no normalization will be performed.
             With very small latencies (microseconds) it's possible that smaller normalization values 
             could cause the application to panic. Increasing the normalization factor will eliminate 
             the issue.
  -cpus      Specifies how many CPUs to use for the test run. The default is 0 which specifies that
			 all CPUs should be used.
  -help     This usage message
`

	configFile := flag.String("config", "", "path and filename containing the runtime configuration")
	logLevel := flag.Int("loglevel", int(zerolog.WarnLevel), "log level, 0 for debug, 1 info, 2 warn, ...")
	outputType := flag.String("out", "text", "what type of report is desired, 'text' or 'json'")
	normalizationFactor := flag.Int("nf", 0, "normalization factor used to compress the output histogram by eliminating long tails. If provided, the value must be at least 10. The default is 0 which signifies no normalization will be done")
	cpus := flag.Int("cpus", 0, "number of CPUs to use for the test run. Default is 0 which specifies all CPUs are to be used.")
	help := flag.Bool("help", false, "help will emit detailed usage instructions and exit")
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")

	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal().Err(err).Msg("unable to create cpuprofile file")
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	if *help {
		fmt.Println(usage)
		return
	}

	if *configFile == "" {
		fmt.Println("Config file location not provided")
		fmt.Println(usage)
		os.Exit(1)
	}

	if *normalizationFactor == 1 {
		log.Fatal().Msgf("nf (normalizationFactor) value of 1 was provided. This is an invalid value. It must either be omitted or be at least 2.")
	}

	zerolog.SetGlobalLevel(zerolog.Level(*logLevel))
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.StampMilli})
	log.Info().Msgf("heyyall started with config from %s", *configFile)

	config, err := getConfig(*configFile)
	if err != nil {
		log.Fatal().Err(err).Msg("error loading configuration")
	}

	availCPUs := runtime.NumCPU()
	if *cpus > availCPUs {
		log.Fatal().Msgf("-cpus specfied %d CPUs are to be used. Only %d are available", *cpus, availCPUs)
	}
	runtime.GOMAXPROCS(runtime.NumCPU())
	if *cpus > 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	responseC := make(chan internal.Response, config.MaxConcurrentRqsts)
	doneC := make(chan interface{})
	progressC := make(chan interface{})

	var reportDetail internal.OutputType = internal.JSON
	if *outputType == "text" {
		reportDetail = internal.Text
	}
	responseHandler := &internal.ResponseHandler{
		OutputType: reportDetail,
		ResponseC:  responseC,
		ProgressC:  progressC,
		DoneC:      doneC,
		NumRqsts:   config.NumRequests,
		NormFactor: *normalizationFactor,
	}
	go responseHandler.Start()

	var cert tls.Certificate
	if config.CertFile != "" && config.KeyFile != "" {
		cert, err = tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
		if err != nil {
			log.Fatal().Err(err).Msg("Error creating x509 keypair")
		}
	}

	// TODO: Make Transport configurable, including timeout that's currently on the client below
	t := &http.Transport{
		MaxIdleConnsPerHost: config.MaxConcurrentRqsts,
		DisableCompression:  false,
		DisableKeepAlives:   false,
		TLSClientConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
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
		return
	}

	dur, err := time.ParseDuration(config.RunDuration)
	if err != nil {
		log.Fatal().Err(err).Msg(fmt.Sprintf("runDur: %s, must be of the form 'xs' or xm where 'x' is an integer and 's' indicates seconds and 'm' indicates minutes",
			config.RunDuration))
		return
	}
	go startProgressBar(progressC, doneC, dur, config.NumRequests)

	go scheduler.Start()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigs:
		signal.Stop(sigs)
		log.Debug().Msg("heyyall: SIGTERM caught")
		<-doneC // Wait for graceful shutdown to complete
	case <-doneC:
	}

	log.Info().Msg("heyyall: DONE")
}

func getConfig(fileName string) (api.LoadTestConfig, error) {
	contents, err := ioutil.ReadFile(fileName)
	if err != nil {
		return api.LoadTestConfig{}, fmt.Errorf("unable to read config file %s", fileName)
	}

	log.Debug().Msgf("Raw config file contents: %s", string(contents))

	config := api.LoadTestConfig{}
	if err = json.Unmarshal(contents, &config); err != nil {
		return api.LoadTestConfig{}, fmt.Errorf("error unmarshaling test config bytes: %s", string(contents))
	}
	return config, nil
}

func startProgressBar(progressC chan interface{}, doneC chan interface{}, dur time.Duration, numRqsts int) {
	progress := mpb.New(mpb.WithWidth(64))
	var total int64
	if int64(dur) > 0 {
		total = int64(dur / time.Second)
	} else {
		total = int64(numRqsts)
	}
	// barName := ""
	bar := progress.AddBar(total,
		mpb.PrependDecorators(
			// display our name with one space on the right
			// decor.Name(barName, decor.WC{W: len(barName) + 1, C: decor.DidentRight}),
			decor.OnComplete(decor.Name("Running", decor.WCSyncSpaceR), "Done!"),
			// replace ETA decorator with "done" message, OnComplete event
			// decor.OnComplete(
			// 	decor.AverageETA(decor.ET_STYLE_GO, decor.WC{W: 4}), "done",
			// ),
		),
		mpb.AppendDecorators(decor.Percentage()),
	)

	var tickerC <-chan time.Time
	var ticker *time.Ticker
	pC := progressC
	if dur > 0 {
		ticker = time.NewTicker(time.Second)
		tickerC = ticker.C
		pC = nil
	}

	fmt.Printf("\nBegin load test...\n\n")

LOOP:
	for {
		select {
		case <-doneC:
			break LOOP
		case <-tickerC:
			bar.Increment()
		case <-pC:
			bar.Increment()
		}
	}

	progress.Wait()
	if ticker != nil {
		ticker.Stop()
	}
}
