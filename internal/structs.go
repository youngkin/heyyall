// Copyright (c) 2020 Richard Youngkin. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package internal

import (
	"time"

	"github.com/youngkin/heyyall/api"
)

// ReportDetail specifies the level of detail of the final report. There are
// 2 values, 'Short' and 'Long'. 'Short' will only report the high level stats.
// 'Long' will report details for each endpoint in addition to the hibh
// level stats.
type ReportDetail int

const (
	// Short specifies only high level report stats will be produced
	Short ReportDetail = iota
	// Long indicates detailed reporting stats will be produced
	Long
)

// Response contains information describing the results
// of a request to a specific endpoint
type Response struct {
	HTTPStatus      int
	Endpoint        api.Endpoint
	RequestDuration time.Duration
}

// Request contains the information needed to execute a request
// to an endpoint and return the response.
type Request struct {
	EP api.Endpoint
}
