package internal

import (
	"fmt"
	"net/http"
	"time"
)

// Scheduler is the component that will schedule requests to endpoints. It
// expects to be run as a goroutine.
type Scheduler struct {
	// CloseC is a channel that when closed signals the scheduler to exit
	CloseC chan struct{}
	// DoneC is used by the Scheduler to signal that it has completed the run,
	// either in terms of load run duration or total number of requests.
	DoneC chan struct{}
	// SchedC is used to send requests to the Scheduler
	SchedC chan Request
	// Rate is the desired overall requests per second
	Rate int
	// MaxConcurrentRqsts is the overall number of simulataneously
	// running requests
	MaxConcurrentRqsts int
	// RunDuration is how long the test will run. It can be expressed
	// in seconds or minutes as xs or xm where x is an integer (e.g.,
	// 10s for 10 seconds, 5m for 5 minutes). If both NumRequests and
	// RunDuration are specified then whichever one is met first will
	// cause the run to cease. For example, if RunDuration is 10s and
	// NumRequests is 200,000,000, if RunDuration expires before 200,000,000
	// requests are made then the load test will finish at 10 seconds
	// regardless of the number of requests specified.
	RunDuration time.Duration
	// NumRequests is the total number of requests to make. See RunDuration
	// above for the behavior when both RunDuration and NumRequests are
	// specified.
	NumRequests int
}

// Start begins a loop that schedules each request for forwarding
// to the endpoint in the request
func (s Scheduler) Start() {
	// Semaphore (implemented via channels) to control max
	// concurrent requests
	concurrencySem := make(chan struct{}, s.MaxConcurrentRqsts)
	timesUp := time.After(s.RunDuration)
	for i := 0; i < s.NumRequests; i++ {
		select {
		case <-s.CloseC:
			return
		case <-timesUp:
			close(s.DoneC)
			return
		case rqst := <-s.SchedC:
			// Rqst ability to send off a request
			concurrencySem <- struct{}{}
			go processRqst(rqst, concurrencySem)
		}
	}
	close(s.DoneC)
}

func processRqst(rqst Request, freeResource chan struct{}) {
	fmt.Println("processRqst: URL %s. IMPLEMENT ME!!!!!!", rqst.EP.URL)
	rqst.ResponseC <- Response{
		HTTPStatus:      http.StatusInternalServerError,
		Endpoint:        "http://implementme.now!!!!",
		RequestDuration: time.Duration(time.Second * 1),
	}
	// signal a request has completed processing
	<-freeResource
}
