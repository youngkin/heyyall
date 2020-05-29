// Copyright (c) 2020 Richard Youngkin. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package internal

import (
	"fmt"
	"os"
	"sort"
	"text/template"
	"time"

	"github.com/rs/zerolog/log"
)

var tmpltFuncs = template.FuncMap{
	"formatFloat":      formatFloat,
	"formatSeconds":    formatSeconds,
	"formatPercentile": formatPercentile,
}

func formatFloat(f float64) string {
	return fmt.Sprintf("%4.4f", f)
}

func formatSeconds(d time.Duration) string {
	return fmt.Sprintf("%4.4f", d.Seconds())
}

func formatPercentile(p int, d []time.Duration) string {
	val := calcPercentiles(p, d)
	return formatSeconds(val)
}

var runSummTmplt = `
Run Summary:
	        Total Rqsts: {{ .RqstStats.TotalRqsts }}
	          Rqsts/sec: {{ formatFloat .RqstRatePerSec }}
	Run Duration (secs): {{ formatSeconds .RunDuration }}
`

var rqstSummTmplt = `
Request Latency (secs): Min      Median      P75      P90      P95      P99
	                    {{ formatPercentile 0 .TimingResults }}   {{  formatPercentile 50 .TimingResults }}   {{  formatPercentile 75 .TimingResults }}   {{  formatPercentile 90 .TimingResults }}   {{  formatPercentile 95 .TimingResults }}   {{  formatPercentile 99 .TimingResults }}
`

var rqstHistogramTmplt = `
Request Latency Histogram (secs):
	Latency    Number of Observations
	-------    ----------------------
`

var netSummTmplt = `
Network Details (secs):
					Min      Median      P75      P90      P95      P99
	    DNS Lookup: {{ formatPercentile 0 .DNSLookup }}   {{ formatPercentile 50 .DNSLookup }}   {{ formatPercentile 75 .DNSLookup }}   {{ formatPercentile 90 .DNSLookup }}   {{ formatPercentile 95 .DNSLookup }}   {{ formatPercentile 99 .DNSLookup }}       
	TCP Conn Setup: {{ formatPercentile 0 .TCPConnSetup }}   {{ formatPercentile 50 .TCPConnSetup }}   {{ formatPercentile 75 .TCPConnSetup }}   {{ formatPercentile 90 .TCPConnSetup }}   {{ formatPercentile 95 .TCPConnSetup }}   {{ formatPercentile 99 .TCPConnSetup }}                  
	 TLS Handshake: {{ formatPercentile 0 .TLSHandshake }}   {{ formatPercentile 50 .TLSHandshake }}   {{ formatPercentile 75 .TLSHandshake }}   {{ formatPercentile 90 .TLSHandshake }}   {{ formatPercentile 95 .TLSHandshake }}   {{ formatPercentile 99 .TLSHandshake }}        
	Rqst Roundtrip: {{ formatPercentile 0 .RqstRoundTrip }}   {{ formatPercentile 50 .RqstRoundTrip }}   {{ formatPercentile 75 .RqstRoundTrip }}   {{ formatPercentile 90 .RqstRoundTrip }}   {{ formatPercentile 95 .RqstRoundTrip }}   {{ formatPercentile 99 .RqstRoundTrip }}        
`

func printRunSummary(rs RunSummary) {
	tmplt, err := template.New("runSummary").Funcs(tmpltFuncs).Parse(runSummTmplt)
	if err != nil {
		log.Error().Err(err).Msg("error parsing runResults template")
	}

	err = tmplt.Execute(os.Stdout, rs)
	if err != nil {
		log.Error().Err(err).Msg("error executing runResults template")
	}
}

func printRqstSummary(rh *ResponseHandler) {
	tmplt, err := template.New("rqstSummary").Funcs(tmpltFuncs).Parse(rqstSummTmplt)
	if err != nil {
		log.Error().Err(err).Msg("error parsing rqstSummary template")
	}

	err = tmplt.Execute(os.Stdout, rh)
	if err != nil {
		log.Error().Err(err).Msg("error executing rqstSummary template")
	}
}

func printNetworkSummary(rs RunSummary) {
	tmplt, err := template.New("runSummary").Funcs(tmpltFuncs).Parse(netSummTmplt)
	if err != nil {
		log.Error().Err(err).Msg("error parsing network summarytemplate")
	}

	err = tmplt.Execute(os.Stdout, rs)
	if err != nil {
		log.Error().Err(err).Msg("error executing network summary template")
	}
}

func calcPercentiles(percentile int, results []time.Duration) time.Duration {
	if len(results) == 0 {
		return 0
	}

	if percentile == 0 {
		return calcPMin(results)
	}

	if percentile == 50 {
		return calcPMedian(results)
	}

	sort.Slice(results, func(i, j int) bool { return results[i] < results[j] })
	p := (len(results) - 1) * percentile / 100
	return results[int(p)]
}

func calcPMin(results []time.Duration) time.Duration {
	if len(results) == 0 {
		return 0
	}
	sort.Slice(results, func(i, j int) bool { return results[i] < results[j] })
	return results[0]
}

func calcPMedian(results []time.Duration) time.Duration {
	if len(results) == 0 {
		return 0
	}

	sort.Slice(results, func(i, j int) bool { return results[i] < results[j] })

	isEven := len(results)%2 == 0
	mNumber := len(results) / 2

	if !isEven {
		return results[mNumber]
	}
	return (results[mNumber-1] + results[mNumber]) / time.Duration(2)
}

// func calcP90(results []time.Duration) time.Duration {
// 	if len(results) == 0 {
// 		return 0
// 	}

// 	sort.Slice(results, func(i, j int) bool { return results[i] < results[j] })
// 	p90 := float64(len(results)-1) * 0.90
// 	return results[int(p90)]
// }

// func calcP99(results []time.Duration) time.Duration {
// 	if len(results) == 0 {
// 		return 0
// 	}

// 	sort.Slice(results, func(i, j int) bool { return results[i] < results[j] })
// 	p99 := float64(len(results)-1) * 0.99
// 	return results[int(p99)]
// }
