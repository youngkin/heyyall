package internal

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/youngkin/heyyall/api"
)

// rqstItem contains the information needed to determine how often to send
// requests to a given endpoint. It also includes the nextRunOffset iteration to
// send the request. This 'nextRunOffset' interation will be recalculated after each
// request is sent.
type rqstItem struct {
	freq int // 1 is every request, 2 is every other request, 10 is every 10th request...
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
	// rate is the desired overall requests per second
	rate int
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
	// rqstSched is a map, keyed by the next scheduled request iteration, of schedFreqs
	rqstSched map[int][]rqstItem
}

// NewRequestor returns a valid Requestor instance
func NewRequestor(ctx context.Context, doneC chan struct{}, schedC chan Request,
	responseC chan Response, rate int, runDur string, numRqsts int, eps []api.Endpoint) (Requestor, error) {

	dur, err := time.ParseDuration(runDur)
	if err != nil {
		return Requestor{}, fmt.Errorf("runDur: %s, must be of the form xs or xm where x is an integer",
			runDur)
	}

	// Sort slice in reverse order to get EPs that should be run more frequently,
	// e.g., 10/sec vs. 1/sec, are first
	sort.Slice(eps, func(i, j int) bool { return eps[i].RqstPercent > eps[j].RqstPercent })

	rSched := make(map[int][]rqstItem)
	totalPct := 0
	for _, ep := range eps {
		totalPct += ep.RqstPercent
		sFreq := rqstItem{
			freq: ep.RqstPercent,
			ep:   ep,
		}
		// Initially start with all requests in the 0th cell. This ensures that
		// the most frequently run requests will be first in the slice, due to the
		// originally sorting of the 'eps' slice above. As the run progresses the
		// requests will eventually spread out across the map according to their
		// relative frequencies. See 'getNextRqst()' for details.
		if _, ok := rSched[0]; !ok {
			rSched[0] = make([]rqstItem, 0)
		}
		rSched[0] = append(rSched[0], sFreq)
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
		rate:      rate,
		runDur:    dur,
		numRqsts:  numRqsts,
		endpoints: eps,
		rqstSched: rSched,
	}
	log.Debug().Msgf("Requestor: %+v", rqstor)

	return rqstor, nil
}

// Start begins the scheduling process
func (r Requestor) Start() {
	log.Debug().Msg("Requestor starting")
	timesUp := time.After(r.runDur)
	// We may have to adjust how many times to run the loop below if we
	// hit empty slice cells (which can happen)
	rqstsToProcess := r.numRqsts
	for i := 0; i < rqstsToProcess; i++ {
		nextRqst, ok := getNextRqst(r.rqstSched, i)
		if !ok {
			log.Debug().Msgf("No requests at Requestor.freqs[%d]", i)
			// increment limit so that we don't count this empty cell as a request
			rqstsToProcess++
			continue
		}
		// fmt.Printf("DEBUG:\tSending request %+v\n", nextRqst)

		start := time.Now()
		select {
		case <-r.ctx.Done():
			return
		case <-timesUp:
			log.Debug().Msgf("Time's up after %d seconds. Sending DONE signal", r.runDur/time.Second)
			close(r.doneC)
			return
		case r.schedC <- Request{EP: nextRqst.ep, ResponseC: r.responseC}:
			// fmt.Printf("DEBUG:\tSent Request for %+v\n", nextRqst.ep)
		}

		// Sleep here to control rate. This will be approximate. If sending
		// a request blocks longer than the rate, i.e., 'delta' < 0 then the
		// rate will be slower.
		since := time.Since(start)
		delta := (time.Second / time.Duration(r.rate)) - since
		if delta < 0 {
			continue
		}
		time.Sleep(delta)
	}
	// Sleep a bit before stopping to allow in-flight requests to complete
	time.Sleep(time.Millisecond * 500)
	log.Debug().Msgf("Sending DONE signal after processing %d requests \n", r.numRqsts)
	close(r.doneC)
}

func getNextRqst(freqs map[int][]rqstItem, idx int) (rqstItem, bool) {
	candidateRqsts, ok := freqs[idx]
	if !ok {
		// TODO:
		// There may be holes in the map, i.e., nothing to send this
		// iteration. For example, for any RqstPercent < 100 there won't
		// be a [0] entry. This isn't great, need to come back and address it.
		return rqstItem{}, false
	}
	var nextRqst rqstItem
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
	nextSchedIteration := idx + nextRqst.freq
	if _, ok := freqs[nextSchedIteration]; !ok {
		freqs[nextSchedIteration] = make([]rqstItem, 0)
	}
	// fmt.Printf("DEBUG:\t!!!!\tSCHEDULING rqst %s from current location %d for freq %d to location %d\n", nextRqst.ep.URL, idx, nextRqst.freq, nextSchedIteration)
	freqs[nextSchedIteration] = append(freqs[nextSchedIteration], nextRqst)

	return nextRqst, true
}
