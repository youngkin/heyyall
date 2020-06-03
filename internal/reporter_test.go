// Copyright (c) 2020 Richard Youngkin. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package internal

import (
	"testing"
	"time"
)

func TestPercentileCalcs(t *testing.T) {
	var min, median, p75, p90, p95, p99 = "min", "median", "p75", "p90", "p95", "p99"
	tests := []struct {
		testName     string
		startVal     time.Duration
		stepVal      time.Duration
		numVals      int
		expectedVals map[string]time.Duration
	}{
		{
			testName: "no durations",
			startVal: time.Millisecond * 0,
			stepVal:  time.Millisecond * 0,
			numVals:  0,
			expectedVals: map[string]time.Duration{
				min:    time.Millisecond * 0,
				median: time.Millisecond * 0,
				p75:    time.Millisecond * 0,
				p90:    time.Millisecond * 0,
				p95:    time.Millisecond * 0,
				p99:    time.Millisecond * 0,
			},
		},
		{
			testName: "2 durations",
			startVal: time.Millisecond * 100,
			stepVal:  time.Millisecond * 900,
			numVals:  2,
			expectedVals: map[string]time.Duration{
				min:    time.Millisecond * 100,
				median: time.Millisecond * 550,
				p75:    time.Millisecond * 1000,
				p90:    time.Millisecond * 1000,
				p95:    time.Millisecond * 1000,
				p99:    time.Millisecond * 1000,
			},
		},
		{
			testName: "5 durations",
			startVal: time.Millisecond * 100,
			stepVal:  time.Millisecond * 100,
			numVals:  5,
			expectedVals: map[string]time.Duration{
				min:    time.Millisecond * 100,
				median: time.Millisecond * 300,
				p75:    time.Millisecond * 400,
				p90:    time.Millisecond * 500,
				p95:    time.Millisecond * 500,
				p99:    time.Millisecond * 500,
			},
		},
		{
			// NOTE: Due to rounding, P50-99 will be rounded up, i.e., the next higher cell will be chosen
			testName: "100 durations",
			startVal: time.Millisecond * 1,
			stepVal:  time.Millisecond * 1,
			numVals:  100,
			expectedVals: map[string]time.Duration{
				min:    time.Millisecond * 1,
				median: time.Microsecond * 50500,
				p75:    time.Millisecond * 76,
				p90:    time.Millisecond * 91,
				p95:    time.Millisecond * 96,
				p99:    time.Millisecond * 100,
			},
		},
		{
			// NOTE: As with the previous test, P50-99 will be rounded up.
			testName: "1000 durations",
			startVal: time.Millisecond * 1,
			stepVal:  time.Millisecond * 1,
			numVals:  1000,
			expectedVals: map[string]time.Duration{
				min:    time.Millisecond * 1,
				median: time.Microsecond * 500500,
				p75:    time.Millisecond * 751,
				p90:    time.Millisecond * 901,
				p95:    time.Millisecond * 951,
				p99:    time.Millisecond * 991,
			},
		},
	}
	// in := []time.Duration{time.Millisecond * 100, time.Millisecond * 1000}
	// expectedMin := time.Millisecond * 100
	// expectedMedian := time.Millisecond * 550
	// expectedP75 := time.Millisecond * 1000
	// expectedP90 := time.Millisecond * 1000
	// expectedP95 := time.Millisecond * 1000
	// expectedP99 := time.Millisecond * 1000

	for _, tc := range tests {
		t.Run(tc.testName, func(t *testing.T) {
			resultsIn := make([]time.Duration, 0)
			for i, d := 0, tc.startVal; i < tc.numVals; i, d = i+1, d+tc.stepVal {
				resultsIn = append(resultsIn, d)
			}

			actualMin := calcPMin(resultsIn)
			actualMedian := calcPMedian(resultsIn)
			actualP75 := calcPercentiles(75, resultsIn)
			actualP90 := calcPercentiles(90, resultsIn)
			actualP95 := calcPercentiles(95, resultsIn)
			actualP99 := calcPercentiles(99, resultsIn)

			if tc.expectedVals[min] != actualMin {
				t.Errorf("Min: expected %s, got %s", tc.expectedVals[min], actualMin)
			}
			if tc.expectedVals[median] != actualMedian {
				t.Errorf("Median: expected %s, got %s", tc.expectedVals[median], actualMedian)
			}
			if tc.expectedVals[p75] != actualP75 {
				t.Errorf("P75: expected %s, got %s", tc.expectedVals[p75], actualP75)
			}
			if tc.expectedVals[p90] != actualP90 {
				t.Errorf("P90: expected %s, got %s", tc.expectedVals[p90], actualP90)
			}
			if tc.expectedVals[p95] != actualP95 {
				t.Errorf("P95: expected %s, got %s", tc.expectedVals[p95], actualP95)
			}
			if tc.expectedVals[p99] != actualP99 {
				t.Errorf("P99: expected %s, got %s", tc.expectedVals[p99], actualP99)
			}

		})
	}
}
