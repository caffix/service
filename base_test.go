// Copyright © by Jeff Foley 2020-2022. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"testing"
	"time"
)

func TestStart(t *testing.T) {
	data := "testData"
	srv := newTestService()

	select {
	case srv.Input() <- data:
		t.Errorf("The service is handling requests before the Start method was called")
	default:
	}

	_ = srv.Start()
	defer func() { _ = srv.Stop() }()
	time.Sleep(500 * time.Millisecond)

	select {
	case srv.Input() <- data:
	default:
		t.Errorf("The service did not start when the Start method was executed")
	}
}

func TestStop(t *testing.T) {
	srv := newTestService()

	_ = srv.Start()
	if err := srv.Stop(); err == nil {
		select {
		case <-srv.Done():
		default:
			t.Errorf("The service did not stop successfully after being started")
		}
	}

	select {
	case srv.Input() <- "testData":
		t.Errorf("The service is handling requests after the Stop method was called")
	default:
	}
}

func TestRequest(t *testing.T) {
	srv := newTestService()

	_ = srv.Start()
	defer func() { _ = srv.Stop() }()
	// Check that the requests are being processed in the correct order
	for _, str := range []string{"str1", "str2", "str3"} {
		srv.Input() <- str
		if result := <-srv.Output(); result != str {
			t.Errorf("Expected %s to be returned and received %s", str, result)
		}
	}
}

func TestRateLimit(t *testing.T) {
	srv := newTestService()
	srv.SetRateLimit(2)

	_ = srv.Start()
	defer func() { _ = srv.Stop() }()

	start := time.Now()
	for _, str := range []string{"1", "2", "3", "4"} {
		srv.Input() <- str
		<-srv.Output()
	}
	finish := time.Now()

	if finish.Sub(start) < time.Second {
		t.Errorf("The rate limit was not enforced between requests")
	}
}

type testService struct {
	BaseService
	done chan struct{}
}

func newTestService() *testService {
	srv := &testService{
		done: make(chan struct{}),
	}

	srv.BaseService = *NewBaseService(srv, "Test")
	return srv
}

func (srv *testService) OnStart() error {
	go srv.handleRequests()
	return nil
}

func (srv *testService) OnStop() error {
	close(srv.done)
	return nil
}

func (srv *testService) handleRequests() {
	for {
		srv.CheckRateLimit()

		select {
		case <-srv.done:
			return
		case req := <-srv.Input():
			srv.Output() <- req
		}
	}
}
