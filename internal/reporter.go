// Copyright (c) 2020 Richard Youngkin. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package internal

import (
	"fmt"
	"math"
	"os"
	"sort"
	"text/template"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/youngkin/heyyall/api"
)

// OutputType specifies the output formate of the final report. There are
// 2 values, 'text' and 'json'. 'text' will present a human readable form.
// 'json' will present the JSON structures that capture the detailed run
// stats.
type OutputType int

const (
	// Text specifies only high level report stats will be produced
	Text OutputType = iota
	// JSON indicates detailed reporting stats will be produced
	JSON
)

var tmpltFuncs = template.FuncMap{
	"formatFloat":      formatFloat,
	"formatSeconds":    formatSeconds,
	"formatPercentile": formatPercentile,
	"formatMethod":     formatMethod,
	"format100Million": format100Million,
}

func formatFloat(f float64) string {
	return fmt.Sprintf("%4.4f", f)
}

func formatSeconds(d time.Duration) string {
	return fmt.Sprintf("%04.4f", d.Seconds())
}

func formatPercentile(p int, d []time.Duration) string {
	val := calcPercentiles(p, d)
	return formatSeconds(val)
}

func formatMethod(m string) string {
	if len(m) == 6 { // length of 'DELETE'
		return m
	}

	if len(m) == 3 {
		return fmt.Sprintf("   %s", m)
	}

	return fmt.Sprintf("  %s", m)
}

func format100Million(i int64) string {
	return fmt.Sprintf("%9v", i)
}

var runSummTmplt = `
Run Summary:
	        Total Rqsts: {{ .RqstStats.TotalRqsts }}
	          Rqsts/sec: {{ formatFloat .RqstRatePerSec }}
	Run Duration (secs): {{ formatSeconds .RunDurationNanos }}
`

var rqstLatencyTmplt = `
Request Latency (secs): Min      Median   P75      P90      P95      P99
	                    {{ formatPercentile 0 .TimingResultsNanos }}   {{  formatPercentile 50 .TimingResultsNanos }}   {{  formatPercentile 75 .TimingResultsNanos }}   {{  formatPercentile 90 .TimingResultsNanos }}   {{  formatPercentile 95 .TimingResultsNanos }}   {{  formatPercentile 99 .TimingResultsNanos }}
`

var netDetailsTmplt = `
Network Details (secs):
					Min      Median      P75      P90      P95      P99
	    DNS Lookup: {{ formatPercentile 0 .DNSLookupNanos }}   {{ formatPercentile 50 .DNSLookupNanos }}   {{ formatPercentile 75 .DNSLookupNanos }}   {{ formatPercentile 90 .DNSLookupNanos }}   {{ formatPercentile 95 .DNSLookupNanos }}   {{ formatPercentile 99 .DNSLookupNanos }}       
	TCP Conn Setup: {{ formatPercentile 0 .TCPConnSetupNanos }}   {{ formatPercentile 50 .TCPConnSetupNanos }}   {{ formatPercentile 75 .TCPConnSetupNanos }}   {{ formatPercentile 90 .TCPConnSetupNanos }}   {{ formatPercentile 95 .TCPConnSetupNanos }}   {{ formatPercentile 99 .TCPConnSetupNanos }}                  
	 TLS Handshake: {{ formatPercentile 0 .TLSHandshakeNanos }}   {{ formatPercentile 50 .TLSHandshakeNanos }}   {{ formatPercentile 75 .TLSHandshakeNanos }}   {{ formatPercentile 90 .TLSHandshakeNanos }}   {{ formatPercentile 95 .TLSHandshakeNanos }}   {{ formatPercentile 99 .TLSHandshakeNanos }}        
	Rqst Roundtrip: {{ formatPercentile 0 .RqstRoundTripNanos }}   {{ formatPercentile 50 .RqstRoundTripNanos }}   {{ formatPercentile 75 .RqstRoundTripNanos }}   {{ formatPercentile 90 .RqstRoundTripNanos }}   {{ formatPercentile 95 .RqstRoundTripNanos }}   {{ formatPercentile 99 .RqstRoundTripNanos }}        
`

// Pass in a EndpointDetails keyed by URL and range over EndpointDetail
// HTTPMethodRqstStats (map[string]*RqstStats keyed by Method)
var endpointDetailsTmplt = `
Endpoint Details(secs): {{ range $url, $epDetails := . }}    
  {{ $url }}:
	            Requests   Min        Median     P75        P90        P95        P99 {{ range $method, $epDetail := .HTTPMethodRqstStats }}
	  {{ formatMethod $method }}:  {{ format100Million .TotalRqsts }}   {{ formatPercentile 0 .TimingResultsNanos }}     {{  formatPercentile 50 .TimingResultsNanos }}     {{  formatPercentile 75 .TimingResultsNanos }}     {{  formatPercentile 90 .TimingResultsNanos }}     {{  formatPercentile 95 .TimingResultsNanos }}     {{  formatPercentile 99 .TimingResultsNanos }} {{ end }}
	{{ end }}
`

func printRunSummary(rs api.RunSummary) {
	tmplt, err := template.New("runSummary").Funcs(tmpltFuncs).Parse(runSummTmplt)
	if err != nil {
		log.Error().Err(err).Msg("error parsing runResults template")
	}

	err = tmplt.Execute(os.Stdout, rs)
	if err != nil {
		log.Error().Err(err).Msg("error executing runResults template")
	}
}

func printRqstLatency(rs api.RqstStats) {
	tmplt, err := template.New("rqstLatency").Funcs(tmpltFuncs).Parse(rqstLatencyTmplt)
	if err != nil {
		log.Error().Err(err).Msg("error parsing rqstLatency template")
	}

	err = tmplt.Execute(os.Stdout, rs)
	if err != nil {
		log.Error().Err(err).Msg("error executing rqstLatency template")
	}
}

func printNetworkDetails(rs api.RunSummary) {
	tmplt, err := template.New("networkDetails").Funcs(tmpltFuncs).Parse(netDetailsTmplt)
	if err != nil {
		log.Error().Err(err).Msg("error parsing networkDetails template")
	}

	err = tmplt.Execute(os.Stdout, rs)
	if err != nil {
		log.Error().Err(err).Msg("error executing networkDetails template")
	}
}

func printEndpointDetails(epd map[string]*api.EndpointDetail) {
	tmplt, err := template.New("endpointDetail").Funcs(tmpltFuncs).Parse(endpointDetailsTmplt)
	if err != nil {
		log.Error().Err(err).Msg("error parsing endpoint detail template")
	}

	err = tmplt.Execute(os.Stdout, epd)
	if err != nil {
		log.Error().Err(err).Msg("error executing endpoint detail template")
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

	// applying math.Ceil to the results of math.Ceil is required to round up
	// to the next results cell when len(results) is a small number, e.g., like
	// 2. Otherwise Median is greater than P99.
	p := math.Ceil(math.Ceil(float64((len(results)-1)*percentile)) / 100)
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
