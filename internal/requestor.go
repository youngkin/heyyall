package internal

import (
	"time"

	"github.com/youngkin/heystack/heystack/api"
)

// Requestor determines which requests to make over the SchedC
// channel based on each Endpoint's 'RqstPercent'
type Requestor struct {
	CloseC    chan struct{}
	SchedC    chan Request
	ResponseC chan Response
	Endpoints []api.Endpoint
}

// Start begins the scheduling process
func (r Requestor) Start() {
	for i := 0; i < 10; i++ {
		select {
		case <-CloseC:
			return
		default:
			time.Sleep(time.Second * 1)
			r.SchedC <- Request{
				EP:        r.Endpoints[0],
				ResponseC: r.ResponseC,
			}
		}
	}
}
