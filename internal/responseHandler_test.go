// Copyright (c) 2020 Richard Youngkin. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package internal

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

var (
	update           = flag.Bool("update", false, "update .golden files")
	goldenFileDir    = "testdata"
	goldenFileSuffix = ".golden"
)

// TestResponseHapopyPath is interesting given that it captures stdout
// for a later comparison for testing. However, it's not practical due
// to timing differences from run-to-run. As a result run results are
// not comparable between runs.
func TestResponseHandlerHappyPath(t *testing.T) {
	rescueStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("received error capturing stdout: %s", err)
	}
	os.Stdout = w
	testName := "HappyPath"

	// 	maxConcurrentRqsts := 1
	// 	httpStatus := 200
	// 	ep := api.Endpoint{URL: "http://somewhere.com/xyz", Method: http.MethodGet}
	// 	responseC := make(chan Response, maxConcurrentRqsts)
	// 	doneC := make(chan struct{})

	// 	start := time.Now()

	// 	responseHandler := ResponseHandler{
	// 		ResponseC: responseC,
	// 		DoneC:     doneC,
	// 	}
	// 	go responseHandler.Start()

	// 	time.Sleep(time.Millisecond * 100) // delay so RequestDuration isn't 0
	// 	t.Logf("duration in millis since start %d", time.Since(start)/time.Millisecond)
	// 	for i := 0; i < 110; i++ {
	// 		responseC <- Response{
	// 			HTTPStatus:      httpStatus,
	// 			Endpoint:        api.Endpoint{URL: ep.URL, Method: ep.Method},
	// 			RequestDuration: time.Millisecond * 100,
	// 		}
	// 	}

	// 	close(responseC)
	// 	<-doneC

	w.Close()
	actual, err := ioutil.ReadAll(r)
	if err != nil {
		t.Errorf("error reading actual results: %s", err)
	}

	if *update {
		updateGoldenFile(t, testName, string(actual))
	}

	expected := readGoldenFile(t, testName)
	if expected != string(actual) {
		t.Errorf("expected %s, got %s", expected, string(actual))
	}

	os.Stdout = rescueStdout
}

func updateGoldenFile(t *testing.T, testName string, contents string) {
	gf := filepath.Join(goldenFileDir, testName+goldenFileSuffix)
	t.Log("update golden file")
	if err := ioutil.WriteFile(gf, []byte(contents), 0644); err != nil {
		t.Fatalf("failed to update golden file: %s", err)
	}
}

func readGoldenFile(t *testing.T, testName string) string {
	gf := filepath.Join(goldenFileDir, testName+goldenFileSuffix)
	gfc, err := ioutil.ReadFile(gf)
	if err != nil {
		t.Fatalf("failed reading golden file: %s", err)
	}
	return string(gfc)
}
