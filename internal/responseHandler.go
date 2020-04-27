package internal

// ResponseHandler is responsible for accepting, summarizing, and reporting
// on the overall load test results.
type ResponseHandler struct {
	CloseC chan struct{}
	ResponseC chan Response{}
}

func (rh ResponseHandler) Start() {
	// TODO IMPLEMENT
	numResponses := 0
	for {
		select {
		case <-rh.CloseC:
			fmt.Printf("Received %d responses\n", numResponses)
			return
	case resp := <- rh.ResponseC:
		numResponses++
		}
	}
}