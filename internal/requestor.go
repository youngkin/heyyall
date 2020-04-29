package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/youngkin/heyyall/api"
)

// schedFreq contains the information needed to determine how often to send
// requests to a given endpoint. It also includes the next iteration to
// send the request. This 'next' interation will be recalculated after each
// request is sent.
type schedFreq struct {
	freq int
	next int
	ep   api.Endpoint
}

// Requestor determines which requests to make over the schedC
// channel based on each Endpoint's 'RqstPercent'
type Requestor struct {
	ctx context.Context
	// schedC is a channel used to forward a request for sending to
	// and endpoint
	schedC    chan Request
	responseC chan Response
	// doneC is used by the Requestor to signal that it has completed the run,
	// either in terms of load test run duration or total number of requests.
	doneC chan struct{}
	// runDur is how long the test will run. It can be expressed
	// in seconds or minutes as xs or xm where x is an integer (e.g.,
	// 10s for 10 seconds, 5m for 5 minutes). If both numRqsts and
	// runDur are specified then whichever one is met first will
	// cause the run to cease. For example, if runDur is 10s and
	// numRqsts is 200,000,000, if runDur expires before 200,000,000
	// requests are made then the load test will finish at 10 seconds
	// regardless of the number of requests specified.
	runDur time.Duration
	// numRqsts is the total number of requests to make. See runDur
	// above for the behavior when both runDur and numRqsts are
	// specified.
	numRqsts int
	// endpoints represents the set of endpoints getting requests
	endpoints []api.Endpoint
	// freqs is a map, keyed by the next scheduled request iteration, of schedFreqs
	freqs map[int][]schedFreq
}

// NewRequestor returns a valid Requestor instance
func NewRequestor(ctx context.Context, doneC chan struct{}, schedC chan Request,
	responseC chan Response, runDur string, numRqsts int, eps []api.Endpoint) (Requestor, error) {

	dur, err := time.ParseDuration(runDur)
	if err != nil {
		return Requestor{}, fmt.Errorf("runDur: %s, must be of the form xs or xm where x is an integer",
			runDur)
	}

	if numRqsts < 100 {
		fmt.Printf("WARNING: NumRequests: %d is less than 100. Some requests may not be sent\n", numRqsts)
	}

	freqs := make(map[int][]schedFreq)
	totalPct := 0
	for _, ep := range eps {
		totalPct += ep.RqstPercent
		sFreq := schedFreq{
			freq: (100 / ep.RqstPercent) - 1,
			next: (100 / ep.RqstPercent) - 1,
			ep:   ep,
		}
		if _, ok := freqs[sFreq.next]; !ok {
			freqs[sFreq.next] = make([]schedFreq, 0)
		}
		freqs[sFreq.next] = append(freqs[sFreq.next], sFreq)
	}
	if totalPct != 100 {
		return Requestor{}, fmt.Errorf("total RqstPercent across all Endpoints is %d. It must not be greater than 100",
			totalPct)
	}

	rqstor := Requestor{
		ctx:       ctx,
		schedC:    schedC,
		responseC: responseC,
		doneC:     doneC,
		runDur:    dur,
		numRqsts:  numRqsts,
		endpoints: eps,
		freqs:     freqs,
	}
	fmt.Printf("INFO: Requestor: %+v\n", rqstor)

	return rqstor, nil
}

// Start begins the scheduling process
func (r Requestor) Start() {
	fmt.Println("INFO:\tRequestor starting")
	timesUp := time.After(r.runDur)
	// TODO: Need to try at least 1 more than numRequests in case multiple
	// requests map to the same cell
	for i := 0; i <= r.numRqsts; i++ {
		nextRqst, ok := getNextRqst(r.freqs, i)
		if !ok {
			fmt.Printf("INFO:\tNo requests at Requestor.freqs[%d]\n", i)
			continue
		}
		fmt.Printf("DEBUG:\tSending request %+v\n", nextRqst)

		select {
		case <-r.ctx.Done():
			return
		case <-timesUp:
			fmt.Printf("INFO:\tTime's up after %d seconds. Sending DONE signal\n", r.runDur/time.Second)
			close(r.doneC)
			return
		case r.schedC <- Request{EP: nextRqst.ep, ResponseC: r.responseC}:
			fmt.Printf("DEBUG:\tSent Request for %+v\n", nextRqst.ep)
		}
	}
	// Sleep a bit before stopping to allow in-flight requests to complete
	time.Sleep(time.Millisecond * 500)
	fmt.Printf("INFO:\tSending DONE signal after processing %d requests \n", r.numRqsts)
	close(r.doneC)
}

func getNextRqst(freqs map[int][]schedFreq, idx int) (schedFreq, bool) {
	candidateRqsts, ok := freqs[idx]
	if !ok {
		// TODO:
		// There may be holes in the map, i.e., nothing to send this
		// iteration. For example, for any RqstPercent < 100 there won't
		// be a [0] entry. This isn't great, need to come back and address it.
		return schedFreq{}, false
	}
	var nextRqst schedFreq
	if len(candidateRqsts) > 0 {
		nextRqst = candidateRqsts[0]
	}
	if len(candidateRqsts) > 1 {
		// We won't be back this way again, carry any outstanding
		// requests forward. Eventually we'll hit a hole and fill it.
		freqs[idx+1] = freqs[idx][1:]
	}
	delete(freqs, idx)

	// Before moving on, we need to schedule the next time this
	// request should be scheduled.
	nextSchedIteration := nextRqst.next + nextRqst.freq
	if _, ok := freqs[nextSchedIteration]; !ok {
		freqs[nextSchedIteration] = make([]schedFreq, 0)
	}
	freqs[nextSchedIteration] = append(freqs[nextSchedIteration], nextRqst)

	return nextRqst, true
}
